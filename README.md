# go-tk — Go Backend Toolkit

[![Go Version](https://img.shields.io/badge/go-1.24+-00ADD8?logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Build Status](https://github.com/TubagusAldiMY/go-tk/workflows/CI/badge.svg)](https://github.com/TubagusAldiMY/go-tk/actions)

**go-tk** is a production-ready CLI toolkit for Go backend developers — analogous to Laravel Artisan or Rails CLI for the Go ecosystem. It scaffolds projects with Clean Architecture, OWASP security compliance, and opinionated best practices.

## 🎯 Features

### Project Scaffolding
```bash
go-tk new my-api
```
- **4 stack combinations**: Gin/Fiber × PostgreSQL/MySQL
- **24 templates per stack**: Full project structure with Clean Architecture
- **Production-ready**: Docker, Makefile, CI/CD templates, .env validation
- **Security defaults**: JWT auth, rate limiting, password hashing (bcrypt cost=12)

### CRUD Generation
```bash
go-tk gen crud Product --fields="name:string,price:decimal,stock:int"
```
Generates **8 files**:
- Entity (domain layer)
- Repository interface + implementation (GORM)
- Use case (business logic)
- HTTP handler (Gin or Fiber)
- DTOs (request/response)
- Migrations (up/down, PostgreSQL or MySQL)

### Database Migrations
```bash
go-tk migrate up      # Apply pending migrations
go-tk migrate status  # Check migration state
go-tk migrate create add_user_roles  # Create new migration
```

### Environment Management
```bash
go-tk env validate    # Check .env consistency
go-tk env sync        # Sync .env.example from code
```

### API Testing
```bash
go-tk test  # Auto-discover routes → generate smoke tests
```

### Static Analysis
```bash
go-tk analyze --fail-under=75  # Enforce quality in CI
```

**7 Checks**:
1. Unhandled errors (errors discarded with `_`)
2. Missing input validation (bind without validate)
3. N+1 query patterns (DB call inside loop)
4. Hardcoded credentials and secrets
5. Dead routes & orphaned handlers
6. Missing auth middleware (mutable endpoints without auth)
7. Circular imports

---

## 🚀 Quick Start

### Installation
```bash
# From source
go install github.com/TubagusAldiMY/go-tk/cmd/go-tk@latest

# Or build locally
git clone https://github.com/TubagusAldiMY/go-tk.git
cd go-tk
make build
sudo mv go-tk /usr/local/bin/
```

### Create Your First Project
```bash
# Interactive mode (recommended for first time)
go-tk new my-api

# Or use flags
go-tk new my-api \
  --module=github.com/user/my-api \
  --framework=gin \
  --database=postgres \
  --auth=jwt

cd my-api
go-tk migrate up
make dev  # Start with hot reload
```

### Generate a CRUD Resource
```bash
go-tk gen crud Product \
  --fields="name:string:required,price:decimal:required,stock:int"
go-tk migrate up
```

---

## 📋 Commands Reference

| Command | Description |
|---------|-------------|
| `go-tk new <name>` | Scaffold new Go backend project |
| `go-tk gen crud <Entity>` | Generate full CRUD (entity, repo, usecase, handler, DTO, migrations) |
| `go-tk migrate up/down/status/create/validate` | Database migration management |
| `go-tk env validate/sync/generate-example/check` | Environment variable management |
| `go-tk test` | Auto-discover routes and generate/run HTTP tests |
| `go-tk analyze` | Static analysis (7 checks: errors, N+1, auth, dead routes, etc.) |

Run `go-tk <command> --help` for detailed options.

---

## 🏗️ Architecture

### Generated Project Structure

```
my-api/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── domain/                  # Domain layer (entities + repository interfaces)
│   │   ├── entity/
│   │   └── repository/
│   ├── application/             # Use cases (business logic)
│   │   └── usecase/
│   ├── infrastructure/          # Infrastructure (DB, cache, external services)
│   │   ├── repository/          # Repository implementations
│   │   └── database/
│   │       └── migrations/
│   └── interfaces/              # Delivery layer (HTTP handlers, middleware)
│       ├── http/
│       │   ├── handler/
│       │   ├── middleware/
│       │   └── router/
│       └── dto/
├── pkg/                         # Reusable utilities
│   ├── logger/
│   ├── validator/
│   └── response/
├── gotk.yaml                    # Project config
├── Makefile                     # Build & run commands
├── docker-compose.yml           # Local development services
└── .env.example                 # Environment template
```

### Dependency Rule (Non-Negotiable)

```
domain        → No external dependencies
application   → Depends on domain only
infrastructure → Implements domain interfaces
interfaces    → Uses application layer
```

Violations are detected by `go-tk analyze --check=circular-imports`.

---

## 🔐 Security Standards

All generated code follows OWASP Top 10 best practices:

- **A01 — Broken Access Control**: RBAC middleware on all routes
- **A02 — Cryptographic Failures**: Bcrypt (cost=12) for passwords, AES-256-GCM for PII
- **A03 — Injection**: Parameterized queries only (GORM prepared statements)
- **A07 — Auth Failures**: JWT with short expiry (15m access + 7d refresh), rate limiting
- **A09 — Logging Failures**: Structured logging with PII masking

See [AGENTS.md](AGENTS.md) for full compliance standards.

---

## 🧪 Development

### Prerequisites
- Go 1.24+
- PostgreSQL 15+ or MySQL 8+ (for testing)
- golangci-lint (for linting)

### Build & Test
```bash
make build        # Build go-tk binary
make test         # Run all tests (with race detector)
make lint         # Run golangci-lint
make coverage     # Generate coverage report
```

### Project Commands
```bash
make dev          # Start with hot reload (air)
make migrate-up   # Apply migrations
make check        # fmt + vet + lint + test + security scan
```

---

## 📖 Documentation

- [Architecture Decision Records](docs/adr/) — Technical architecture and trade-offs
- [CLAUDE.md](CLAUDE.md) — AI assistant guidance (project context)
- [AGENTS.md](AGENTS.md) — Compliance and coding standards
- [FRD](dokumen/go-tk-FRD.md) — Functional Requirements
- [SRS](dokumen/go-tk-SRS.md) — System Requirements
- [BRD](dokumen/go-tk-BRD.md) — Business Requirements

---

## 🛣️ Roadmap

See [todo.md](todo.md) for detailed task list.

**v1.0 (Current)**
- ✅ Project scaffolding (4 stacks)
- ✅ CRUD generation (Gin + Fiber support)
- ✅ Migration management
- ✅ Static analysis (7 checks)
- ⏳ Security features (rate limiting, lockout, refresh tokens) — **In Progress**

**v1.1 (Q2 2026)**
- JSON/HTML output for analyze & test commands
- `--skip` flag for CRUD generator
- DB connectivity pre-flight check
- CI/CD workflow templates

**v2.0 (Future)**
- GraphQL support
- OpenAPI spec generation
- Kubernetes deployment templates
- sqlc ORM support

---

## 🤝 Contributing

Contributions are welcome! Please:
1. Read [AGENTS.md](AGENTS.md) for coding standards
2. Run `make check` before committing
3. Follow [Conventional Commits](https://www.conventionalcommits.org/)
4. Add tests for new features

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

## 💬 Support

- **Issues**: [GitHub Issues](https://github.com/TubagusAldiMY/go-tk/issues)
- **Discussions**: [GitHub Discussions](https://github.com/TubagusAldiMY/go-tk/discussions)
- **Email**: [your-email@example.com](mailto:your-email@example.com)

---

**Built with ❤️ for the Go community**
