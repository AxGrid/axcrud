# Transport Layer for Refine + GORM Repository

Этот пакет реализует **универсальный транспортный слой** для работы с [refine](https://refine.dev) и вашим backend-API на Go.  
Он строится поверх generic-репозитория (GORM) и добавляет:

- поддержку **refine query** (пагинация, сортировка, фильтры, поиск);
- адаптер `RefineListRequest → ListParams`;
- HTTP-хендлеры для популярных фреймворков (**Chi / Gin / Fiber**);
- универсальный механизм **трансформации DTO** для скрытия/изменения полей (например, замена автоинкрементного `ID` на `HashID`).

---

## 1. Репозиторий (Repo)

```go
type Repo[T any, ID IDConstraint] interface {
    GetList(ctx context.Context, p ListParams) ([]T, int64, error)
    GetOne(ctx context.Context, id ID) (T, error)
    GetMany(ctx context.Context, ids []ID) ([]T, error)
    Create(ctx context.Context, in *T) error
    Update(ctx context.Context, id ID, patch map[string]any) (T, error)
    Delete(ctx context.Context, id ID) error
    DeleteMany(ctx context.Context, ids []ID) (int64, error)
}
```

Реализация для GORM (`GormRepo`) обрабатывает фильтры, поиск, сортировку и пагинацию.  
`IDConstraint` ограничивает поддерживаемые типы идентификаторов:

```go
type IDConstraint interface {
    ~uint | ~uint64 | ~int | ~int64 | ~string
}
```

---

## 2. Refine адаптер

Фронтенд **refine** шлёт запросы в формате:

```json
{
  "pagination": { "current": 1, "pageSize": 20 },
  "sorters": [ { "field": "created_at", "order": "desc" } ],
  "filters": [ { "field": "role", "operator": "in", "value": ["admin", "manager"] } ],
  "search": "alex",
  "searchFields": ["name","email"]
}
```

Адаптер преобразует их в `ListParams`:

```go
req := RefineListRequest{ ... }
params := AdaptRefineList(req)
items, total, err := userRepo.GetList(ctx, params)
```

Для `GET`-запросов доступен парсер query-параметров:

```
/users?current=1&pageSize=20&sorters[0][field]=created_at&sorters[0][order]=desc&q=alex
```

```go
req := ParseRefineQuery(r.URL.Query())
params := AdaptRefineList(req)
```

---

## 3. Трансформации (DTO)

Иногда нужно скрыть/переименовать поля.  
Например: отдавать наружу не `ID`, а `HashID`.

```go
type User struct {
    ID    uint
    Name  string
    Email string
    Role  string
}

type WebUser struct {
    HashID string `json:"id"`
    Name   string `json:"name"`
    Email  string `json:"email"`
    Role   string `json:"role"`
}

// Хешер ID → строка
var hashID = transport.NewIDHasher("SUPER-SECRET-SALT", 11)

// Маппер
func WebUserMapper(ctx context.Context, u User) (WebUser, error) {
    return WebUser{
        HashID: hashID(fmt.Sprint(u.ID)),
        Name:   u.Name,
        Email:  u.Email,
        Role:   u.Role,
    }, nil
}
```

---

## 4. Transport Handlers

Транспорт содержит готовые хендлеры для всех CRUD-операций:

- `getList` / `postList`
- `create`
- `getOne`
- `getMany`
- `update`
- `delete`
- `deleteMany`

### DTO-варианты (`*-T`)

Все хендлеры имеют версию `*-T`, которая принимает `TransformFn[T, DTO]`.  
Таким образом, можно выдавать наружу DTO, а не доменную модель.

---

## 5. Chi

```go
r := chi.NewRouter()

r.Get("/users",              transport.ChiGetListT[User, uint, WebUser](userRepo, WebUserMapper))
r.Post("/users/list",        transport.ChiPostListT[User, uint, WebUser](userRepo, WebUserMapper))
r.Post("/users",             transport.ChiCreateT[User, uint, WebUser](userRepo, WebUserMapper))
r.Get("/users/{id}",         transport.ChiGetOneT[User, uint, WebUser](userRepo, WebUserMapper))
r.Get("/users/many",         transport.ChiGetManyT[User, uint, WebUser](userRepo, WebUserMapper))
r.Post("/users/getMany",     transport.ChiGetManyT[User, uint, WebUser](userRepo, WebUserMapper))
r.Patch("/users/{id}",       transport.ChiUpdateT[User, uint, WebUser](userRepo, WebUserMapper))
r.Delete("/users/{id}",      transport.ChiDeleteT[User, uint, WebUser](userRepo))
r.Post("/users/deleteMany",  transport.ChiDeleteManyT[User, uint, WebUser](userRepo))
```

---

## 6. Gin

```go
r := gin.Default()

r.GET("/users",              transport.GinGetListT[User, uint, WebUser](userRepo, WebUserMapper))
r.POST("/users/list",        transport.GinPostListT[User, uint, WebUser](userRepo, WebUserMapper))
r.POST("/users",             transport.GinCreateT[User, uint, WebUser](userRepo, WebUserMapper))
r.GET("/users/:id",          transport.GinGetOneT[User, uint, WebUser](userRepo, WebUserMapper))
r.GET("/users/many",         transport.GinGetManyT[User, uint, WebUser](userRepo, WebUserMapper))
r.POST("/users/getMany",     transport.GinGetManyT[User, uint, WebUser](userRepo, WebUserMapper))
r.PATCH("/users/:id",        transport.GinUpdateT[User, uint, WebUser](userRepo, WebUserMapper))
r.DELETE("/users/:id",       transport.GinDeleteT[User, uint, WebUser](userRepo))
r.POST("/users/deleteMany",  transport.GinDeleteManyT[User, uint, WebUser](userRepo))
```

---

## 7. Fiber

```go
app := fiber.New()

app.Get("/users",             transport.FiberGetListT[User, uint, WebUser](userRepo, WebUserMapper))
app.Post("/users/list",       transport.FiberPostListT[User, uint, WebUser](userRepo, WebUserMapper))
app.Post("/users",            transport.FiberCreateT[User, uint, WebUser](userRepo, WebUserMapper))
app.Get("/users/:id",         transport.FiberGetOneT[User, uint, WebUser](userRepo, WebUserMapper))
app.Get("/users/many",        transport.FiberGetManyT[User, uint, WebUser](userRepo, WebUserMapper))
app.Post("/users/getMany",    transport.FiberGetManyT[User, uint, WebUser](userRepo, WebUserMapper))
app.Patch("/users/:id",       transport.FiberUpdateT[User, uint, WebUser](userRepo, WebUserMapper))
app.Delete("/users/:id",      transport.FiberDeleteT[User, uint, WebUser](userRepo))
app.Post("/users/deleteMany", transport.FiberDeleteManyT[User, uint, WebUser](userRepo))
```

---

## 8. Пример запроса из refine

### Список пользователей

```ts
const { data, total } = await dataProvider.getList({
  resource: "users",
  pagination: { current: 1, pageSize: 10 },
  sorters: [{ field: "created_at", order: "desc" }],
  filters: [{ field: "role", operator: "eq", value: "admin" }],
  meta: { q: "alex" }
});
```

→ Backend получит:

```
GET /users?current=1&pageSize=10&sorters[0][field]=created_at&sorters[0][order]=desc&filters[0][field]=role&filters[0][operator]=eq&filters[0][value]=admin&q=alex
```

### Ответ

```json
{
  "data": [
    { "id": "h9AjskPq11a", "name": "Alex", "email": "alex@example.com", "role": "admin" }
  ],
  "total": 1
}
```

---

## 9. Итог

- Репозиторий работает с **чистыми доменными моделями** (`User`).
- Транспортный слой адаптирует **refine-запросы** и превращает их в `ListParams`.
- Хендлеры автоматически маппят доменные модели в **DTO** (`WebUser`).
- Есть поддержка **Chi / Gin / Fiber**.
- Добавить новый ресурс = определить DTO + маппер + прописать маршруты.
