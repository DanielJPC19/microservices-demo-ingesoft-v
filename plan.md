# Implementation Plan: CI/CD & DevOps Strategy

## 1. Project Understanding

### Architecture

This is a **polyglot microservices voting application** (Tacos vs Burritos) with three services:

| Service | Language | Role |
|---------|----------|------|
| `vote` | Java 22 / Spring Boot 3.4.1 | Web frontend — receives votes, publishes to Kafka |
| `worker` | Go 1.24.1 | Kafka consumer — reads votes, writes to PostgreSQL |
| `result` | Node.js 22 / Express | Web frontend — reads PostgreSQL, streams results via Socket.io |

**Infrastructure dependencies:** Apache Kafka 3.7.0 (messaging), PostgreSQL 16 (storage), both deployed via Helm.

**Data flow (CQRS):**
```
[User] → Vote Service → Kafka "votes" topic → Worker → PostgreSQL
                                                           ↑
[User] ← Result Service (Socket.io) ←────────────── (polling 1s)
```

### Current State of Tests

| Service | Status |
|---------|--------|
| vote (Java) | Test dependencies declared in `pom.xml` (`spring-boot-starter-test`, `spring-kafka-test`) but `-DskipTests` is passed in the Dockerfile. No test classes found. |
| result (Node.js) | `"test"` script in `package.json` is `echo \"Error: no test specified\" && exit 1`. No test framework installed. |
| worker (Go) | No `*_test.go` files exist anywhere. |

**Conclusion:** Zero active test coverage. Tests must be written before CI gates can be meaningful.

### Current State of CI/CD

- No `.github/workflows/` directory exists.
- The only automation config is `okteto.yml`, which handles local development hot-reload via Okteto Cloud — **not used going forward**.
- Docker multi-stage builds exist and are production-ready for all three services.
- Helm charts exist for all services and for infrastructure.

### Notable Code Issues to Address

- `worker/main.go:39` — Worker **drops the votes table on every startup** (`DROP TABLE IF EXISTS votes`). This is destructive in any persistent environment and must be fixed before deploying to staging/prod.
- `worker/main.go:26-29` — Database credentials are hardcoded (`user = "okteto"`, `password = "okteto"`). Must be moved to environment variables before CI/CD wires secrets.

---

## 2. Strategy Summary (from `strategy.md`)

### Two-Repository Model

| Repository | Purpose | Branching Model |
|------------|---------|-----------------|
| **Dev repo** (this one) | Source code, Dockerfiles, CI pipelines | Git Flow |
| **Infra repo** (to be created) | Terraform, environment configs, CD pipelines | Trunk-Based Development |

### Git Flow — Dev Repo

```
feature/* ──→ develop ──→ release/* ──→ main
```

- `feature/*`: individual feature branches, PR into `develop`
- `develop`: integration branch — CI builds and pushes `app:dev` image
- `release/*`: pre-production branch — integration + Kafka validation
- `main`: production-ready — CI builds and pushes `app:prod`, then triggers infra repo

### Trunk-Based — Infra Repo

- Single `main` branch
- Small, frequent commits
- Every push to `main` triggers `terraform apply`

### Environment Strategy

| Environment | Triggered By | Docker Tag |
|-------------|-------------|------------|
| DEV | merge to `develop` | `<user>/service:dev` |
| STAGING | merge to `release/*` | `<user>/service:staging` |
| PROD | merge to `main` | `<user>/service:prod` (also `:<sha>` for rollback) |

### Cloud Design Patterns (from `strategy.md`)

| Pattern | Where Applied | Status |
|---------|--------------|--------|
| CQRS | Write path (Vote→Kafka→Worker→PG) + Read path (Result←PG) | Already implemented architecturally |
| Retry + Idempotency | Worker Kafka consumer + DB writes | Partially: DB uses `ON CONFLICT DO UPDATE` — retry loop missing |
| Circuit Breaker | Worker→PostgreSQL | Not implemented |
| Bulkhead | Service-level container isolation | Structural — each service is independent |
| Rate Limiting | Vote API layer | Optional — one-vote-per-client via cookie exists |

---

## 3. Implementation Plan

### Phase 0: Prerequisites (code fixes before CI)

**Goal:** Make the codebase safe for automation.

- [ ] **P0.1** — Remove `DROP TABLE IF EXISTS votes` from `worker/main.go`. Replace with proper migration (create table only if not exists, which already follows on line 43).
- [ ] **P0.2** — Externalize hardcoded DB credentials in `worker/main.go` to environment variables (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`). Use `os.Getenv`.
- [ ] **P0.3** — Remove `-DskipTests` from `vote/Dockerfile` once tests are written.

---

### Phase 1: Write Tests

**Goal:** Every CI pipeline must have a working test gate.

#### 1.1 Vote Service (Java)

- Write unit tests for `VoteController` using `spring-boot-starter-test` + MockMvc.
- Write a Kafka producer integration test using `spring-kafka-test` (embedded Kafka).
- Target: at least `mvn test` passes without `-DskipTests`.

#### 1.2 Worker Service (Go)

- Write unit tests for the database write logic (`main_test.go`).
- Use `database/sql` with a SQLite or mock DB for unit scope.
- Write an integration test using a real PostgreSQL instance (Docker-based in CI).
- Run with `go test ./...`.

#### 1.3 Result Service (Node.js)

- Install a test framework: `jest` + `supertest`.
- Write tests for the Express API routes and Socket.io event emission.
- Run with `npm test`.

---

### Phase 2: CI Pipelines — Dev Repository

**Goal:** GitHub Actions workflows for every trigger point in Git Flow.

Each service has its own workflow file. All three share the same structure per branch event.

#### 2.1 Workflow: `ci-develop.yml`

Triggered by: push or PR merge to `develop`.

```
Steps per service (vote, worker, result):
  1. Checkout code
  2. Set up language runtime (Java 22 / Go 1.24 / Node 22)
  3. Install dependencies
  4. Build service
  5. Run tests
  6. Build Docker image
  7. Push Docker image → Docker Hub (tag: <user>/<service>:dev)
```

On failure at any step: pipeline stops, no image is pushed.

#### 2.2 Workflow: `ci-release.yml`

Triggered by: push to `release/*` branches.

```
Steps per service:
  1-5. Same as develop (build + test)
  6. Integration tests (services communicate over docker-compose network)
  7. Kafka message flow validation (produce a vote → verify worker consumes → verify result reads)
  8. Build Docker image
  9. Push → tag: <user>/<service>:staging
```

#### 2.3 Workflow: `ci-main.yml`

Triggered by: push to `main`.

```
Steps per service:
  1-5. Same as develop (build + full test suite)
  6. Build Docker image
  7. Push → tags:
       <user>/<service>:prod
       <user>/<service>:<git-sha>   ← versioned, enables rollback
  8. Trigger infrastructure repository via GitHub API (repository_dispatch)
```

The trigger to the infra repo sends the image tag as payload so Terraform knows which version to deploy.

#### 2.4 Secrets Required (GitHub Secrets — Dev Repo)

| Secret | Purpose |
|--------|---------|
| `DOCKER_USERNAME` | Docker Hub login |
| `DOCKER_PASSWORD` | Docker Hub password/token |
| `INFRA_REPO_TOKEN` | GitHub PAT to trigger infra repo workflow |
| `INFRA_REPO_NAME` | Target infra repository (`owner/repo`) |

---

### Phase 3: Infrastructure Repository Setup

**Goal:** Create a separate repository that Terraform uses to deploy to each environment.

#### 3.1 Repository Structure

```
infra-repo/
├── .github/workflows/
│   └── cd-deploy.yml
├── envs/
│   ├── dev/
│   │   ├── main.tf
│   │   └── terraform.tfvars
│   ├── staging/
│   │   ├── main.tf
│   │   └── terraform.tfvars
│   └── prod/
│       ├── main.tf
│       └── terraform.tfvars
└── modules/
    └── microservices-stack/
        ├── main.tf       ← defines containers, ports, networking
        ├── variables.tf
        └── outputs.tf
```

#### 3.2 Terraform Responsibilities

- Pull Docker images: `<user>/vote:<tag>`, `<user>/result:<tag>`, `<user>/worker:<tag>`
- Provision containers/services with correct port bindings
- Wire environment variables (DB credentials, Kafka broker address) from secrets
- Replace previous version on re-apply

#### 3.3 Workflow: `cd-deploy.yml`

Triggered by:
- `repository_dispatch` event from dev repo (automatic, on main push)
- `workflow_dispatch` (manual trigger)

```
Steps:
  1. Checkout infra repo
  2. Configure cloud provider credentials
  3. terraform init
  4. terraform validate
  5. terraform plan  (output saved as artifact for review)
  6. [Optional] Manual approval gate for PROD
  7. terraform apply -auto-approve
```

#### 3.4 Secrets Required (GitHub Secrets — Infra Repo)

| Secret | Purpose |
|--------|---------|
| `TF_CLOUD_TOKEN` or provider credentials | Cloud provider auth (AWS/GCP/Azure/Docker) |
| `DB_PASSWORD` | PostgreSQL password (passed to containers as env var) |
| `DOCKER_USERNAME` | Pull images from Docker Hub |
| `DOCKER_PASSWORD` | Docker Hub auth for pull |

---

### Phase 4: Cloud Pattern Implementation

**Goal:** Implement the patterns selected in `strategy.md` that are not yet in code.

#### 4.1 Retry + Idempotency (Worker — Go)

- The DB `ON CONFLICT DO UPDATE` already provides idempotency at the DB level.
- Add retry logic around Kafka consumption errors and DB write failures using exponential backoff.
- Add a max retry count with dead-letter logging so the service doesn't loop forever.

#### 4.2 Circuit Breaker (Worker → PostgreSQL)

- Use a Go circuit breaker library (e.g., `sony/gobreaker`) around `db.Exec()` calls.
- States: Closed (normal) → Open (on N failures) → Half-Open (after cooldown).
- When open: log the failure, skip the message (or re-queue), do not crash the process.

#### 4.3 Bulkhead (Structural — already implemented)

- Each service runs in its own container with isolated resources.
- Helm chart resource limits (`resources.limits`) should be explicitly set per service to enforce the bulkhead boundary.

#### 4.4 Rate Limiting (Optional — Vote Service)

- Spring Boot already enforces one-vote-per-client via cookie (`voter_id`).
- Optionally add `spring-boot-starter-actuator` rate-limit endpoint or a simple `RateLimiter` per IP.

---

### Phase 5: Rollback Procedure

**Goal:** Define operational process for reverting a bad deployment.

Since every `main` push produces a versioned tag (`<user>/<service>:<git-sha>`):

1. Identify the last stable SHA from Docker Hub or the GitHub Actions run history.
2. Update the `image` variable in the relevant `envs/prod/terraform.tfvars` file.
3. Commit directly to infra repo `main` (trunk-based).
4. The `cd-deploy.yml` workflow triggers automatically and runs `terraform apply` with the previous image tag.

No source code changes needed in the dev repo.

---

## 4. Execution Order

```
Phase 0  →  Fix worker (drop table, hardcoded secrets)
Phase 1  →  Write tests for all three services
Phase 2  →  Create GitHub Actions CI workflows in dev repo
Phase 3  →  Create infra repo + Terraform + CD workflow
Phase 4  →  Implement cloud patterns (Retry, Circuit Breaker)
Phase 5  →  Document and test rollback procedure
```

Phases 0 and 1 are blockers for Phase 2. Phases 2 and 3 can proceed in parallel once Phase 1 is done. Phase 4 is independent and can run in parallel with Phase 3.

---

## 5. What Is NOT Changing

- Application logic (voting, Kafka messaging, result streaming) — untouched.
- Helm charts structure — Terraform will reference the same images the charts currently use.
- Okteto-specific tooling — removed from the CI/CD path but the `okteto.yml` file can remain for local developer experience if desired.
- Docker multi-stage build structure — already production-ready, just needs `-DskipTests` removed from vote.
