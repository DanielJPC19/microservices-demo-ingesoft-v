# DevOps Strategy for Microservices Project

## Overview

This project implements a **DevOps workflow for a microservices architecture**, using:

- Microservices (multi-language system: Go, Java, JavaScript, etc.)
- Docker (artifact generation)
- Terraform (Infrastructure as Code)
- GitHub Actions (CI/CD)
- Two repositories:
  - Development repository
  - Infrastructure (operations) repository

---

## Core Principles

1. **Separation of concerns**
   - Development produces artifacts (Docker images)
   - Operations deploys artifacts (via Terraform)

2. **Artifact-based deployment**
   - No source code is transferred between repositories
   - Only Docker images are used as deployable units

3. **Environment isolation**
   - DEV, STAGING, and PROD are clearly separated

---

## Repository Structure

### 1. Development Repository

Contains:
- Microservices source code
- Dockerfiles
- CI pipelines (GitHub Actions)

### 2. Infrastructure Repository

Contains:
- Terraform code
- Environment configurations
- CD pipelines (GitHub Actions)

---

## Branching Strategies

### Development Repository → Git Flow

Branches:

- `feature/*` → new features
- `develop` → integration branch
- `release/*` → pre-production validation
- `main` → production-ready code

---

### Infrastructure Repository → Trunk-Based Development

Branches:

- Only `main`

Rules:

- Small, frequent commits
- Direct deployment from `main`
- No long-lived branches

---

## Development Workflow

### Step 1: Feature Development

- Create branch from `develop`: `feature/new-feature`
- Implement changes
- Open Pull Request to `develop`

---

### Step 2: Integration (develop)

On merge to `develop`, GitHub Actions runs:

1. Install dependencies
2. Build services
3. Run tests
4. Build Docker images
5. Push Docker images with tag: `<docker_user>/app:dev`

Optional:
- Deploy to DEV environment

---

### Step 3: Release Preparation

- Create branch: `release/x.x.x`

Actions:
- Integration testing
- Microservices communication validation
- Event/Kafka validation (if applicable)
- Bug fixes

---

### Step 4: Production Ready (main)

On merge to `main`, GitHub Actions runs:

1. Build application
2. Run full test suite
3. Build Docker image
4. Push Docker image with tag: `<docker_user>/app:prod`
5. Trigger Infrastructure Repository

---

## Inter-Repository Communication

The development repository triggers the infrastructure repository using GitHub API:

- Event: push to `main`
- Action: trigger workflow in infra repo

No source code is transferred — only Docker image references are used.

---

## Infrastructure Workflow (Operations)

Triggered by:

- External event (from dev repo)
- Or manual trigger

---

### Step-by-step process:

1. Checkout repository
2. Initialize Terraform: terraform init
3. Validate configuration: terraform validate
4. Plan changes: terraform plan
5. (Optional) Manual approval
6. Apply changes: terraform apply -auto-approve

---

## Terraform Responsibilities

Terraform is responsible for:

- Pulling the Docker image: `<docker_user>/app:prod`
- Creating/updating infrastructure:
- Containers
- Ports
- Networking
- Replacing previous versions

---

## Environment Strategy

Defined environments:

- **DEV**
- Triggered from `develop`
- Uses image: `app:dev`
- Used for integration testing

- **STAGING**
- Triggered from `release/*`
- Production-like validation

- **PROD**
- Triggered from `main`
- Uses image: `app:prod`

---

### Terraform Structure

```bash
infra/
├── envs/
│    ├── dev/
│    ├── staging/
│    └── prod/
```

Each environment has its own configuration.

---

## Secrets Management

Sensitive data must be stored in GitHub Secrets:

Examples:

- Docker credentials
- Cloud provider credentials
- API tokens

---

## Failure Handling

### CI Failures

If any step fails:

- Build fails → pipeline stops
- Tests fail → pipeline stops
- Docker build fails → pipeline stops

No image is published.

---

### CD Failures

If deployment fails:

- Infrastructure remains unchanged
- Logs must be reviewed

---

## Rollback Strategy

Use versioned Docker images:

```
app:v1
app:v2
app:v3
```

Rollback process:

- Update Terraform image reference
- Re-run `terraform apply`

---

## Expected Behavior

- Every change in `develop` is validated automatically
- Only stable code reaches `main`
- Only `main` triggers production deployment
- Infrastructure is always managed via Terraform
- Operations does not modify application code

---

## Conceptual Flow

```
[Development Repo]

feature → develop → release → main
↓
CI pipeline
↓
Docker image created
↓
Image pushed to registry
↓
Trigger infra repo
```

---

```
[Infrastructure Repo]

Trigger received
↓
Terraform plan
↓
Approval (optional)
↓
Terraform apply
↓
Production updated
```

---

## Key Takeaways

- CI and CD are fully automated
- Deployment is artifact-based (Docker images)
- Development and operations are decoupled
- Infrastructure is reproducible and version-controlled
- Trunk-based enables fast and controlled deployments

---

Aquí tienes el resumen en **un solo bloque markdown**, optimizado para una IA:

---

# Cloud Design Patterns - Applicability Analysis

## System Context

- Event-driven architecture (Kafka)
- Asynchronous processing
- Microservices (Java, Go, Node.js)
- Constraint: **one vote per client**
- Not a mission-critical system (simple demo application)

---

## Selected Patterns

### 1. Retry (with Idempotency)

**Applicability:** YES

**Where:**
- Worker (Kafka consumer)
- Database writes (PostgreSQL)

**Purpose:**
- Handle transient failures (network, DB downtime)

**Risk:**
- Duplicate votes due to retries

**Mitigation:**
- Enforce idempotency:
  - Unique constraint on `client_id`
  - Deduplication logic

---

### 2. CQRS (Command Query Responsibility Segregation)

**Applicability:** YES (Already implemented)

**Command flow (write):**

Frontend → Kafka → Worker → PostgreSQL


**Query flow (read):**

Node.js → PostgreSQL → Results UI


**Benefits:**
- Separation of concerns
- Independent scaling
- Reduced coupling

**Tradeoff:**
- Eventual consistency (delayed updates in UI)

---

### 3. Circuit Breaker

**Applicability:** YES

**Where:**
- Worker → PostgreSQL
- (Optional) Frontend → Kafka

**Purpose:**
- Prevent cascading failures
- Stop repeated failing calls

**Behavior:**
- Open circuit on repeated failures
- Retry after cooldown

**Tradeoff:**
- Increased complexity (state management)

---

### 4. Bulkhead

**Applicability:** YES (Conceptual)

**Where:**
- Service-level isolation:
  - Frontend
  - Kafka
  - Worker
  - Database
  - Results service

**Purpose:**
- Isolate failures
- Prevent system-wide collapse

**Tradeoff:**
- More infrastructure complexity

---

## Optional Pattern

### 5. Rate Limiting

**Applicability:** OPTIONAL

**Where:**
- Frontend / API layer

**Purpose:**
- Prevent abuse (spam votes)
- Reduce load

**Note:**
- Not strictly required due to "one vote per client" rule

---

## Not Recommended

### 6. Asynchronous Request-Reply

**Applicability:** NO

**Reason:**
- System does not require response correlation
- Kafka already provides async communication
- No need for request-response tracking

---

## Final Selection

**Recommended patterns:**

- Retry (with idempotency)
- CQRS
- Circuit Breaker
- Bulkhead

**Optional:**

- Rate Limiting

**Excluded:**

- Asynchronous Request-Reply

---

## 🎯 Key Design Insight

The system naturally aligns with **CQRS and event-driven architecture**, and is enhanced with **resilience patterns (Retry, Circuit Breaker)** and **fault isolation (Bulkhead)** to ensure robustness in a distributed environment.
```
