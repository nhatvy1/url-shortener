# Go Project Structure Standard

## Phân tích dự án hiện tại

### Vấn đề 1: Interface phình to (Fat Interface)

Hiện tại `ShortLinkService` có 4 method, tương lai sẽ thêm: `GetStats`, `BulkCreate`, `ListByUser`, `Search`...
→ Interface sẽ 10+ method → vi phạm **Interface Segregation Principle**.

```go
// ❌ Hiện tại: 1 interface chứa tất cả
type ShortLinkService interface {
    CreateShortLink(...)
    GetOriginalUrl(...)
    UpdateOriginalUrl(...)
    DeleteShortLink(...)
    // tương lai thêm 10 method nữa...
}
```

**Giải pháp**: Tách thành nhiều interface nhỏ theo use-case, mỗi handler chỉ depend vào interface nó cần.

```go
// ✅ Đề xuất: tách theo use-case
type ShortLinkCreator interface {
    CreateShortLink(ctx context.Context, data *vo.CreateShortLinkReq) (string, error)
}

type ShortLinkResolver interface {
    GetOriginalUrl(ctx context.Context, code string) (string, error)
}

type ShortLinkUpdater interface {
    UpdateOriginalUrl(ctx context.Context, code string, newURL string) error
}

type ShortLinkDeleter interface {
    DeleteShortLink(ctx context.Context, code string) error
}
```

Handler chỉ nhận interface nhỏ:
```go
type RedirectHandler struct {
    resolver ShortLinkResolver // chỉ cần 1 method
}
```

---

### Vấn đề 2: DI Container sẽ phình to

Hiện tại `Container` struct chứa tất cả handler + infra. Khi thêm feature: analytics, user management, billing...
→ `NewContainer()` sẽ 200+ dòng, khó maintain.

```go
// ❌ Hiện tại
type Container struct {
    queries          *sqlc.Queries
    cache            cache.Cache
    bloom            cache.BloomFilter
    userHandler      *handlers.UserHandler
    shortLinkHandler *handlers.ShortLinkHandler
    // tương lai: analyticsHandler, billingHandler, adminHandler...
}
```

**Giải pháp**: Dùng **Wire** hoặc tách container theo module/domain.

```go
// ✅ Đề xuất: tách module
func NewShortLinkModule(queries *sqlc.Queries, cache cache.Cache) *ShortLinkModule { ... }
func NewUserModule(queries *sqlc.Queries) *UserModule { ... }
func NewAnalyticsModule(queries *sqlc.Queries) *AnalyticsModule { ... }
```

---

### Vấn đề 3: cmd/api và cmd/worker dùng chung code

Hiện tại `cmd/api` và `cmd/worker` là 2 binary riêng nhưng cần share:
- database connection
- config
- services
- models

Nếu mỗi cmd tự init riêng → duplicate code.

**Giải pháp**: Shared infra layer trong `internal/`, mỗi cmd chỉ compose những gì cần.

```go
// cmd/api/main.go
func main() {
    cfg := config.Load()
    db := database.Connect(cfg.Database)
    r := api.NewServer(db, cfg)
    r.Run()
}

// cmd/worker/main.go
func main() {
    cfg := config.Load()
    db := database.Connect(cfg.Database)
    w := worker.NewWorker(db, cfg)
    w.Start()
}
```

---

### Vấn đề 4: Context không truyền xuyên suốt

Hiện tại service tự tạo `context.Background()`. Nếu cần timeout, tracing, cancel → không làm được.

```go
// ❌ Hiện tại
func (s *shortLinkService) CreateShortLink(data *vo.CreateShortLinkReq) (string, error) {
    ctx := context.Background() // không nhận từ handler
}

// ✅ Đề xuất: truyền context từ handler
func (s *shortLinkService) CreateShortLink(ctx context.Context, data *vo.CreateShortLinkReq) (string, error) {
    // ctx đã có timeout, trace ID, request ID...
}
```

---

## Cấu trúc đề xuất

```
shortlink/
├── cmd/
│   ├── api/
│   │   └── main.go                 # HTTP server entry point
│   └── worker/
│       └── main.go                 # Background worker entry point
│
├── configs/
│   ├── dev.yaml
│   └── prod.yaml
│
├── internal/
│   ├── config/                     # Gộp: load config + settings + globals
│   │   ├── config.go               #   type Config struct { ... }
│   │   └── loader.go               #   func Load() Config
│   │
│   ├── server/                     # HTTP server setup
│   │   ├── server.go               #   func NewServer(deps) *gin.Engine
│   │   ├── routes.go               #   register all routes
│   │   └── middleware.go           #   logger, auth, cors...
│   │
│   ├── worker/                     # Background job setup
│   │   └── worker.go               #   func NewWorker(deps) *Worker
│   │
│   ├── shortlink/                  # ← Domain: Short Link
│   │   ├── handler.go              #   HTTP handlers
│   │   ├── service.go              #   Business logic + interfaces
│   │   ├── repository.go           #   DB access (wrap sqlc)
│   │   └── dto.go                  #   Request/Response structs
│   │
│   ├── user/                       # ← Domain: User
│   │   ├── handler.go
│   │   ├── service.go
│   │   ├── repository.go
│   │   └── dto.go
│   │
│   ├── analytics/                  # ← Domain: Analytics (tương lai)
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   │
│   ├── infra/                      # Shared infrastructure
│   │   ├── database/
│   │   │   └── postgres.go         #   pgxpool connection
│   │   ├── cache/
│   │   │   ├── redis.go            #   Redis client init
│   │   │   ├── cache.go            #   Cache interface + impl
│   │   │   └── bloom.go            #   Bloom filter
│   │   └── logger/
│   │       └── logger.go           #   zap logger
│   │
│   ├── common/                     # Shared utilities
│   │   ├── response.go             #   API response helpers
│   │   ├── errors.go               #   AppError, error codes
│   │   ├── validation.go           #   Custom validators
│   │   └── convert.go              #   String/type helpers
│   │
│   └── container/                  # DI wiring
│       ├── container.go            #   NewContainer() - compose all
│       ├── shortlink_module.go     #   wire shortlink domain
│       └── user_module.go          #   wire user domain
│
├── sqlc/
│   ├── queries/
│   │   ├── shortlink.sql
│   │   └── user.sql
│   ├── schema/
│   │   ├── short_link.sql
│   │   └── user.sql
│   └── db/                         # Generated code
│       ├── db.go
│       ├── models.go
│       ├── shortlink.sql.go
│       └── user.sql.go
│
├── goose/
│   └── migrations/
│       └── 00001_short_links.sql
│
├── sqlc.yaml
├── go.mod
├── go.sum
└── Makefile
```

---

## Nguyên tắc thiết kế

### 1. Domain-based package (thay vì layer-based)

```
# ❌ Layer-based (hiện tại)          # ✅ Domain-based (đề xuất)
internal/                            internal/
  handlers/                            shortlink/
    short_link_handler.go                handler.go
    user_handler.go                      service.go
  services/                              repository.go
    url_shortlink.go                     dto.go
    user_service.go                    user/
  vo/                                    handler.go
    link.go                              service.go
    user.go                              ...
```

**Lý do**:
- Thêm 1 feature mới → thêm 1 folder, không sửa 5 folder
- Mỗi domain tự quản lý interface, impl, dto → giảm coupling
- Dễ tách thành microservice nếu cần

### 2. Interface nhỏ, define ở consumer

```go
// internal/shortlink/service.go
// Interface nhỏ, tách theo use-case
type Creator interface {
    Create(ctx context.Context, req *CreateRequest) (*ShortLink, error)
}

type Resolver interface {
    Resolve(ctx context.Context, code string) (string, error)
}

// Nếu cần gộp (cho DI):
type Service interface {
    Creator
    Resolver
}
```

```go
// internal/shortlink/handler.go
// Handler chỉ depend vào interface nhỏ nhất nó cần
type RedirectHandler struct {
    resolver Resolver // không cần biết Create, Delete...
}

type CreateHandler struct {
    creator Creator
}
```

### 3. Repository pattern wrap sqlc

```go
// internal/shortlink/repository.go
type Repository interface {
    Insert(ctx context.Context, code string, url string, expiresAt time.Time) error
    FindByCode(ctx context.Context, code string) (*ShortLink, error)
}

type pgRepository struct {
    queries *sqlc.Queries
}

func NewRepository(q *sqlc.Queries) Repository {
    return &pgRepository{queries: q}
}
```

**Lý do**: Service không depend trực tiếp vào sqlc → dễ test, dễ swap DB.

### 4. DI Container tách module

```go
// internal/container/shortlink_module.go
type ShortLinkModule struct {
    Handler *shortlink.Handler
}

func NewShortLinkModule(q *sqlc.Queries, c cache.Cache) *ShortLinkModule {
    repo := shortlink.NewRepository(q)
    svc := shortlink.NewService(repo, c)
    handler := shortlink.NewHandler(svc)
    return &ShortLinkModule{Handler: handler}
}

// internal/container/container.go
type Container struct {
    ShortLink *ShortLinkModule
    User      *UserModule
}

func New(cfg config.Config) (*Container, error) {
    db, _ := database.Connect(cfg.Database)
    queries := sqlc.New(db)
    redis, _ := cache.NewRedis(cfg.Redis)

    return &Container{
        ShortLink: NewShortLinkModule(queries, redis),
        User:      NewUserModule(queries),
    }, nil
}
```

### 5. Context truyền xuyên suốt

```
Handler(c *gin.Context)
  → ctx := c.Request.Context()
  → service.Create(ctx, req)
    → repo.Insert(ctx, ...)
      → queries.CreateShortLink(ctx, ...)
```

Mọi method từ service trở xuống đều nhận `context.Context` làm param đầu tiên.

### 6. cmd/ chỉ là entry point

```go
// cmd/api/main.go
func main() {
    cfg := config.Load()
    c, err := container.New(cfg)
    srv := server.New(c)
    srv.Run(cfg.Server.Port)
}

// cmd/worker/main.go
func main() {
    cfg := config.Load()
    c, err := container.New(cfg) // dùng chung container
    w := worker.New(c)
    w.Start()
}
```

---

## So sánh hiện tại vs đề xuất

| Tiêu chí | Hiện tại | Đề xuất |
|---|---|---|
| Package organization | Layer-based | Domain-based |
| Interface size | 1 fat interface/service | Nhiều interface nhỏ theo use-case |
| DI Container | 1 file lớn | Tách theo module |
| Config | 2 hệ thống (viper + godotenv) | 1 hệ thống duy nhất |
| Context | Tạo mới trong service | Truyền từ handler |
| DB access | Service gọi sqlc trực tiếp | Repository wrap sqlc |
| cmd/api vs cmd/worker | Khác nhau hoàn toàn | Share container + infra |
| Globals | `database.DB`, `globals.Config` | Inject qua constructor |
| Test | Khó mock (fat interface) | Dễ mock (small interface) |
| Thêm domain mới | Sửa 5+ folder | Thêm 1 folder |

---

## Khi nào cần thay đổi?

- **Ngay bây giờ**: Truyền `context.Context`, fix Redis hardcode, check err `InitRedis()`
- **Khi thêm domain thứ 2** (user, analytics): Chuyển sang domain-based package
- **Khi service > 6 methods**: Tách interface
- **Khi container > 100 dòng**: Tách module
- **Khi cần unit test**: Thêm repository layer

Không cần refactor tất cả cùng lúc. Chuyển dần khi complexity tăng.
