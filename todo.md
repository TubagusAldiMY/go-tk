# go-tk TODO — Audit Hasil Cek Kelengkapan Proyek

> Audit terakhir: 2026-03-10
> Status: Phase 1–3 implemented, tapi ada **44 gap** teridentifikasi

---

## CRITICAL (Harus dikerjakan sebelum v1.0)

### 1. CRUD Generator Broken untuk Fiber
- [ ] `templates/crud/handler.go.tmpl` hardcoded ke Gin (`*gin.Context`, `gin.RouterGroup`)
- [ ] Buat `templates/crud/handler_fiber.go.tmpl` untuk Fiber (`*fiber.Ctx`, `fiber.Router`)
- [ ] `internal/command/gen/crud/generator.go` harus pilih handler template berdasarkan `stack.framework` dari gotk.yaml
- [ ] Migration template tidak bedakan syntax PostgreSQL vs MySQL (`SERIAL` vs `AUTO_INCREMENT`)
- [ ] Buat `templates/crud/migration.up.postgres.sql.tmpl` dan `migration.up.mysql.sql.tmpl`

### 2. Analyze — Missing Check Implementations
- [ ] **Dead Routes Detection (FN-06.1)**: Buat `checks/dead_routes.go` — cross-reference route definitions dengan handler implementations
- [ ] **Missing Auth Check (FN-06.2)**: Buat `checks/missing_auth.go` — detect routes tanpa auth middleware
- [ ] **Circular Import Detection (FN-06.3)**: Buat `checks/circular_imports.go` — detect import cycles di target project

### 3. Security Features di Generated Code
- [ ] Rate limiting middleware template (`middleware/ratelimit.go.tmpl`) — 10 req/min pada auth endpoints
- [ ] Account lockout mechanism (5 failed attempts → lock 15 min)
- [ ] Password hashing utility (`pkg/crypto/password.go.tmpl`) — bcrypt cost=12
- [ ] Refresh token support — JWT 24h expiry + refresh 7d

### 4. CI/CD Workflow Not Generated
- [ ] Buat `templates/project/*/github/workflows/ci.yml.tmpl` — GitHub Actions workflow
- [ ] Generated project harus punya CI pipeline kalau `--cicd` flag aktif

---

## HIGH (Penting untuk production-readiness)

### 5. Analyze Command — Missing Flags
- [ ] `--output=json` flag — machine-readable output untuk CI pipelines
- [ ] `--output=html` flag — HTML report (type sudah ada di reporter tapi belum di-wire)

### 6. Test Command — Missing Output Format
- [ ] `--output=json` untuk `go-tk test` — CI integration
- [ ] HTML report integration cek ulang apakah sudah connected ke cmd.go

### 7. Missing `--skip` Flag pada CRUD Generator
- [ ] Tambah `--skip` flag di `internal/command/gen/crud/cmd.go` — skip existing files, hanya generate yang belum ada

### 8. Post-Generation Validation (FN-01.5)
- [ ] `go-tk new` harus run `go build ./...` pada generated project untuk validasi
- [ ] Error handling kalau build gagal (with clear message + hint)

### 9. Env Pre-flight Check Incomplete (FN-04.4)
- [ ] `go-tk env check` harus test koneksi database (DSN connectivity, 5s timeout)
- [ ] Bukan cuma cek variable ada/tidak, tapi juga reachable

### 10. Error Handling Inconsistent
- [ ] `test/cmd.go` line 74: `cfg, _ := config.Load(cwd)` — silent error, harus handle
- [ ] `analyze/cmd.go`: config.Load error bisa lebih informatif
- [ ] Semua error harus format: `[ERROR] <what>: <why>. Hint: <action>`

---

## MEDIUM (Quality & completeness)

### 11. Test Coverage Gaps
- [ ] `internal/command/new/` — 0% test coverage (core functionality!)
- [ ] `internal/command/gen/crud/` — hanya `fields_test.go`, generator logic belum di-test
- [ ] `internal/command/migrate/` — hanya `status_test.go`, runner belum di-test
- [ ] `internal/command/test/` — hanya `runner_test.go`, route discovery belum di-test
- [ ] `internal/command/analyze/types/` — 0% coverage

### 12. Missing `--quiet` Flag (Global)
- [ ] Semua command harus support `--quiet` — output hanya errors dan final status
- [ ] Diperlukan untuk CI/CD pipelines yang butuh clean output

### 13. `golang.org/x/tools` Dependency
- [ ] Cek apakah `golang.org/x/tools` ada di go.mod (diperlukan untuk goimports di `formatter.go`)
- [ ] Jika tidak ada, tambahkan — generated Go files butuh auto-import fix

### 14. Go Version Mismatch
- [ ] `go.mod` says go 1.22, `.github/workflows/ci.yml` matrix harus konsisten
- [ ] Pastikan toolchain directive sesuai

### 15. Generated Makefile Missing `pre-deploy` Target
- [ ] Tambah target `make pre-deploy` di semua `Makefile.tmpl`: `go-tk env check`
- [ ] FRD BR-04.4.1 mengharuskan ini ada

### 16. Dependency Analysis (FN-06.4 — Extended)
- [ ] Detect unused packages di target project
- [ ] Detect outdated dependencies
- [ ] Ini bisa jadi Phase 4 / nice-to-have

---

## LOW (Nice-to-have / polish)

### 17. Entity Name Validation
- [ ] CRUD generator harus validasi entity name PascalCase
- [ ] Reject kalau lowercase atau snake_case

### 18. `--force` Confirmation Prompt
- [ ] `go-tk new --force` seharusnya minta konfirmasi interaktif sebelum overwrite
- [ ] Skip konfirmasi kalau bukan TTY (piped/CI)

### 19. Idempotency Edge Cases
- [ ] Test idempotency: `go-tk new` dua kali di directory yang sama
- [ ] Test idempotency: `go-tk gen crud` dua kali dengan entity yang sama

### 20. Documentation
- [ ] README.md belum ada (atau belum update)
- [ ] CHANGELOG.md untuk v1.0 release
- [ ] `go-tk --help` output review — pastikan semua command terdaftar

---

## STATUS SUMMARY

| Area | Status | Gap Count |
|------|--------|-----------|
| Project Generator (go-tk new) | 85% | 3 gaps |
| CRUD Generator (go-tk gen crud) | 60% | 6 gaps (Fiber broken!) |
| Migration Manager (go-tk migrate) | 90% | 2 gaps |
| Env Manager (go-tk env) | 80% | 2 gaps |
| API Tester (go-tk test) | 75% | 3 gaps |
| Project Analyzer (go-tk analyze) | 50% | 8 gaps (3 checks missing!) |
| Template Stacks | 70% | 4 gaps (CRUD stack-aware) |
| Test Coverage | 40% | 5 packages tanpa/minim test |
| Generated Code Security | 30% | 4 security features missing |
| CI/CD | 80% | 2 gaps |

**Build & test saat ini:** `go build ./...` ✅ `go test ./...` ✅ (semua pass)

---

## PRIORITAS PENGERJAAN (Rekomendasi)

1. **Sprint 1**: Fix CRUD Fiber support + missing analyze checks (Critical #1, #2)
2. **Sprint 2**: Security features in generated code (Critical #3, #4)
3. **Sprint 3**: Missing flags + output formats (High #5–#7)
4. **Sprint 4**: Test coverage + polish (Medium #11, Low items)
