package axcrud

import (
	"context"

	"gorm.io/gorm"
)

type IDConstraint interface {
	~uint | ~uint64 | ~int | ~int64 | ~string
}

type Repo[T any, ID IDConstraint] interface {
	GetList(ctx context.Context, p ListParams) (items []T, total int64, err error)
	GetOne(ctx context.Context, id ID) (T, error)
	Create(ctx context.Context, in *T) error
	Update(ctx context.Context, id ID, patch map[string]any) (T, error)
	Delete(ctx context.Context, id ID) error
	DeleteMany(ctx context.Context, ids []ID) (affected int64, err error)
	// Транзакции опционально
	WithTx(tx *gorm.DB) Repo[T, ID]
	GetMany(context.Context, []ID) ([]T, error)
}

type Filter struct {
	Field    string
	Operator string
	Value    any
}

type Sort struct {
	Field string // безопасно только через whitelist
	Order string // "asc"|"desc"
}

type Pagination struct {
	Page    int
	PerPage int
}

type ListParams struct {
	Filters      []Filter
	Sort         *Sort
	Search       string
	SearchFields []string // по каким полям делать поисковый OR ... LIKE
	Pagination   Pagination
}
