package axcrud

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FieldSet map[string]struct{}

func NewFieldSet(fields ...string) FieldSet {
	m := make(FieldSet, len(fields))
	for _, f := range fields {
		m[f] = struct{}{}
	}
	return m
}

func (fs FieldSet) Has(f string) bool {
	_, ok := fs[f]
	return ok
}

type RepoConfig struct {
	// Поля, по которым можно фильтровать: map[field] -> разрешённые операторы
	AllowedFilterOps map[string]FieldSet
	// Поля, по которым можно сортировать
	AllowedSortFields FieldSet
	// Поля, по которым можно искать (LIKE/ILIKE)
	AllowedSearchFields FieldSet
	// Прелоады по умолчанию (если нужно)
	Preloads []string
	// Скоуп для мulti-tenant/ACL, например: func(db) db.Where("user_id = ?", uid)
	Scopes []func(*gorm.DB) *gorm.DB
	// Мягкое удаление: true по умолчанию; UnscopedDelete удаляет физически
	UnscopedDelete bool
}

type GormRepo[T any, ID IDConstraint] struct {
	db    *gorm.DB
	cfg   RepoConfig
	zero  T // zero value для &zero
	idCol string
	table string
}

type TableNamer interface {
	TableName() string
}

// idCol — имя PK (обычно "id"). Можно переопределить через опцию.
type Option func(*GormRepo[struct{}, int])

func NewGormRepo[T any, ID IDConstraint](db *gorm.DB, cfg RepoConfig, opts ...func(*GormRepo[T, ID])) *GormRepo[T, ID] {
	r := &GormRepo[T, ID]{db: db, cfg: cfg, idCol: "id"}
	// определить имя таблицы (важно для некоторых SQL-конструкций)
	var tmp any = new(T)
	if namer, ok := tmp.(TableNamer); ok {
		r.table = namer.TableName()
	}
	for _, o := range opts {
		// generic трюк: обернём в приведение типа при вызове
		o(any(r).(*GormRepo[T, ID]))
	}
	return r
}

func (r *GormRepo[T, ID]) WithTx(tx *gorm.DB) Repo[T, ID] {
	cp := *r
	cp.db = tx
	return &cp
}

func (r *GormRepo[T, ID]) GetOne(ctx context.Context, id ID) (T, error) {
	var out T
	q := r.base(ctx)
	r.applyPreloads(q)
	if err := q.Where(clause.Eq{Column: clause.Column{Name: r.idCol}, Value: id}).First(&out).Error; err != nil {
		return out, err
	}
	return out, nil
}

func (r *GormRepo[T, ID]) Create(ctx context.Context, in *T) error {
	return r.base(ctx).Create(in).Error
}

func (r *GormRepo[T, ID]) Update(ctx context.Context, id ID, patch map[string]any) (T, error) {
	var out T
	if len(patch) == 0 {
		return out, errors.New("empty patch")
	}
	q := r.base(ctx)
	if err := q.Model(&out).
		Where(clause.Eq{Column: clause.Column{Name: r.idCol}, Value: id}).
		// Только указанные ключи; GORM защищает от SQL-инъекций на значения
		Updates(patch).Error; err != nil {
		return out, err
	}
	// вернуть свежую запись
	return r.GetOne(ctx, id)
}

func (r *GormRepo[T, ID]) Save(ctx context.Context, id ID, obj T) (T, error) {
	var out T
	q := r.base(ctx)
	if err := q.Model(&obj).Where(clause.Eq{Column: clause.Column{Name: r.idCol}, Value: id}).
		Save(obj).Error; err != nil {
		return out, err
	}
	return obj, nil
}

func (r *GormRepo[T, ID]) Delete(ctx context.Context, id ID) error {
	q := r.base(ctx)
	if r.cfg.UnscopedDelete {
		q = q.Unscoped()
	}
	var z T
	return q.Where(clause.Eq{Column: clause.Column{Name: r.idCol}, Value: id}).Delete(&z).Error
}

func (r *GormRepo[T, ID]) DeleteMany(ctx context.Context, ids []ID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	q := r.base(ctx)
	if r.cfg.UnscopedDelete {
		q = q.Unscoped()
	}
	var z T
	tx := q.Where(clause.IN{Column: clause.Column{Name: r.idCol}, Values: toAnySlice(ids)}).Delete(&z)
	return tx.RowsAffected, tx.Error
}

func (r *GormRepo[T, ID]) GetMany(ctx context.Context, ids []ID) ([]T, error) {
	var out []T
	if len(ids) == 0 {
		return out, nil
	}
	q := r.base(ctx)
	q = r.applyPreloads(q)
	// IN ? — для всех диалектов
	if err := q.Where(fmt.Sprintf("%s IN ?", r.idCol), ids).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *GormRepo[T, ID]) Count(ctx context.Context) (int64, error) {
	var total int64
	if err := r.base(ctx).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *GormRepo[T, ID]) GetList(ctx context.Context, p ListParams) (items []T, total int64, err error) {
	q := r.base(ctx) // base() должен делать db.Model(new(T))

	// 1) Фильтры
	if q, err = r.applyFilters(q, p.Filters); err != nil {
		return nil, 0, err
	}

	// 2) Поиск
	if s := strings.TrimSpace(p.Search); s != "" {
		if q, err = r.applySearch(q, s, p.SearchFields); err != nil {
			return nil, 0, err
		}
	}

	// 3) Сортировка (не влияет на COUNT, но уже можно навесить здесь)
	if p.Sort != nil {
		if q, err = r.applySort(q, *p.Sort); err != nil {
			return nil, 0, err
		}
	}

	// 4) Подсчёт total (без пагинации и без прелоадов)
	countQ := q.Session(&gorm.Session{}) // клон текущего стейтмента
	if err = countQ.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 5) Пагинация
	page, per := sanitizePage(p.Pagination.Page, p.Pagination.PerPage)
	offset := (page - 1) * per

	// 6) Прелоады и выборка
	q = r.applyPreloads(q)
	if err = q.Limit(per).Offset(offset).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// ---------- Внутренности ----------

//func (r *GormRepo[T, ID]) base() *gorm.DB {
//	q := r.db.Model(new(T))
//	for _, s := range r.cfg.Scopes {
//		q = q.Scopes(s)
//	}
//	return q
//}

func (r *GormRepo[T, ID]) base(ctx context.Context) *gorm.DB {
	q := r.db.Model(new(T)).WithContext(ctx)
	for _, s := range r.cfg.Scopes {
		q = q.Scopes(s)
	}
	return q
}

func (r *GormRepo[T, ID]) applyPreloads(db *gorm.DB) *gorm.DB {
	for _, p := range r.cfg.Preloads {
		db = db.Preload(p)
	}
	return db
}

func (r *GormRepo[T, ID]) applySort(db *gorm.DB, s Sort) (*gorm.DB, error) {
	field := strings.TrimSpace(s.Field)
	if field == "" {
		return db, nil
	}
	if !r.cfg.AllowedSortFields.Has(field) {
		return db, fmt.Errorf("sorting by field '%s' is not allowed", field)
	}
	desc := strings.EqualFold(s.Order, "desc")
	db = db.Order(clause.OrderByColumn{
		Column: clause.Column{Name: field},
		Desc:   desc,
	})
	return db, nil
}

// applySearch: без callback-функций, чистая строка + args
func (r *GormRepo[T, ID]) applySearch(db *gorm.DB, search string, requested []string) (*gorm.DB, error) {
	// пересечение с allowed:
	fields := make([]string, 0, len(requested))
	if len(requested) > 0 {
		for _, f := range requested {
			if r.cfg.AllowedSearchFields.Has(f) {
				fields = append(fields, f)
			}
		}
	} else {
		for f := range r.cfg.AllowedSearchFields {
			fields = append(fields, f)
		}
	}
	if len(fields) == 0 {
		return db, nil
	}

	like := "%" + search + "%"
	conds := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))

	// Кросс-диалектная регистронезависимость:
	// - MySQL/SQLite: LOWER() работает
	// - Postgres: тоже ок
	for _, f := range fields {
		conds = append(conds, fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", clause.Column{Name: f}.Name))
		args = append(args, like)
	}

	// (LOWER(f1) LIKE LOWER(?) OR LOWER(f2) LIKE LOWER(?) ...)
	db = db.Where("("+strings.Join(conds, " OR ")+")", args...)
	return db, nil
}

// applyFilters: IN/NIN через "field IN ?" и "field NOT IN (?)"
func (r *GormRepo[T, ID]) applyFilters(db *gorm.DB, filters []Filter) (*gorm.DB, error) {
	for _, f := range filters {
		field := strings.TrimSpace(f.Field)
		if field == "" {
			continue
		}
		allowedOps, ok := r.cfg.AllowedFilterOps[field]
		if !ok {
			return db, fmt.Errorf("filtering by field '%s' is not allowed", field)
		}
		op := strings.ToLower(strings.TrimSpace(f.Operator))
		if !allowedOps.Has(op) {
			return db, fmt.Errorf("operator '%s' is not allowed on field '%s'", op, field)
		}

		col := clause.Column{Name: field}.Name

		switch op {
		case "eq":
			db = db.Where(fmt.Sprintf("%s = ?", col), f.Value)
		case "ne":
			db = db.Where(fmt.Sprintf("%s <> ?", col), f.Value)
		case "lt":
			db = db.Where(fmt.Sprintf("%s < ?", col), f.Value)
		case "lte":
			db = db.Where(fmt.Sprintf("%s <= ?", col), f.Value)
		case "gt":
			db = db.Where(fmt.Sprintf("%s > ?", col), f.Value)
		case "gte":
			db = db.Where(fmt.Sprintf("%s >= ?", col), f.Value)
		case "in":
			db = db.Where(fmt.Sprintf("%s IN ?", col), toAnySliceFromValue(f.Value))
		case "nin":
			db = db.Where(fmt.Sprintf("%s NOT IN ?", col), toAnySliceFromValue(f.Value))
		case "between":
			vals := toAnySliceFromValue(f.Value)
			if len(vals) != 2 {
				return db, fmt.Errorf("between expects exactly two values")
			}
			db = db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", col), vals[0], vals[1])
		case "contains":
			db = db.Where(fmt.Sprintf("%s LIKE ?", col), "%"+fmt.Sprint(f.Value)+"%")
		case "icontains":
			db = db.Where(fmt.Sprintf("LOWER(%s) LIKE LOWER(?)", col), "%"+fmt.Sprint(f.Value)+"%")
		case "startswith":
			db = db.Where(fmt.Sprintf("%s LIKE ?", col), fmt.Sprint(f.Value)+"%")
		case "endswith":
			db = db.Where(fmt.Sprintf("%s LIKE ?", col), "%"+fmt.Sprint(f.Value))
		case "isnull":
			b, _ := f.Value.(bool)
			if b {
				db = db.Where(fmt.Sprintf("%s IS NULL", col))
			} else {
				db = db.Where(fmt.Sprintf("%s IS NOT NULL", col))
			}
		default:
			return db, fmt.Errorf("unsupported operator: %s", op)
		}
	}
	return db, nil
}

func sanitizePage(p, per int) (int, int) {
	if p <= 0 {
		p = 1
	}
	if per <= 0 {
		per = 10
	}
	if per > 1000 {
		per = 1000
	}
	return p, per
}

func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i := range in {
		out[i] = in[i]
	}
	return out
}

// принимает либо []T, либо любой массив/срез, либо одиночное значение — приводит к []any
func toAnySliceFromValue(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []string:
		return toAnySlice(t)
	case []int:
		return toAnySlice(t)
	case []int64:
		return toAnySlice(t)
	case []uint:
		return toAnySlice(t)
	case []uint64:
		return toAnySlice(t)
	case []float64:
		return toAnySlice(t)
	case []time.Time:
		return toAnySlice(t)
	default:
		// попытка обернуть единичное значение
		return []any{v}
	}
}
