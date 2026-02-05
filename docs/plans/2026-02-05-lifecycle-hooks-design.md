# Lifecycle Hooks Design

**Date:** 2026-02-05
**Status:** Approved
**Author:** Vivek Kundariya

## Overview

Add lifecycle hooks to Grund services, allowing users to run custom scripts at various stages of service startup and shutdown. This enables database seeding, migrations, service initialization, and any automation needed during the service lifecycle.

## Configuration Schema

Hooks are defined per-service in `grund.yaml`:

```yaml
service:
  name: user-service
  type: go
  port: 8080

  hooks:
    pre_up:
      - name: "Generate config"
        command: "./scripts/generate-config.sh"
        target: host
        timeout: 30s                    # optional, default 10m
        continue_on_error: true         # optional, default false

    post_infrastructure:
      - name: "Run migrations"
        command: "psql -h localhost -U postgres -f ./migrations/up.sql"
        target: host

    post_up:
      - name: "Create admin user"
        command: "bin/create-admin"
        target: container

    pre_down:
      - name: "Drain connections"
        command: "bin/drain"
        target: container
        timeout: 2m

    post_down:
      - name: "Cleanup"
        command: "rm -rf ./tmp/cache"
        target: host
```

### Field Defaults

| Field | Default | Required |
|-------|---------|----------|
| `name` | - | Yes |
| `command` | - | Yes |
| `target` | - | Yes (no default, forces explicit choice) |
| `timeout` | `10m` | No |
| `continue_on_error` | `false` | No |

## Lifecycle Stages

### Execution Order: `grund up`

```
grund up user-service
│
├─ 1. Load service configs
├─ 2. Resolve dependencies
├─ 3. ▶ PRE_UP hooks (host only)
├─ 4. Generate docker-compose files
├─ 5. Start infrastructure containers (postgres, redis, localstack)
├─ 6. Wait for infrastructure healthy
├─ 7. Provision resources (create DBs, queues, buckets)
├─ 8. ▶ POST_INFRASTRUCTURE hooks (host only)
├─ 9. Start service containers
├─ 10. Wait for services healthy
└─ 11. ▶ POST_UP hooks (host or container)
```

### Execution Order: `grund down`

```
grund down user-service
│
├─ 1. ▶ PRE_DOWN hooks (host or container)
├─ 2. Stop service containers
├─ 3. Stop infrastructure containers
└─ 4. ▶ POST_DOWN hooks (host only)
```

### Target Constraints by Stage

| Stage | `target: host` | `target: container` |
|-------|----------------|---------------------|
| `pre_up` | ✅ | ❌ (no containers yet) |
| `post_infrastructure` | ✅ | ❌ (service not started) |
| `post_up` | ✅ | ✅ |
| `pre_down` | ✅ | ✅ |
| `post_down` | ✅ | ❌ (containers stopped) |

## Hook Execution

### Host Execution (`target: host`)

Runs command in shell on the local machine:

```go
cmd := exec.CommandContext(ctx, "sh", "-c", hook.Command)
cmd.Dir = service.Path    // Service's root directory (where grund.yaml lives)
cmd.Env = resolvedEnvVars // Same env vars the service gets
cmd.Stdout = os.Stdout    // Stream output to terminal
cmd.Stderr = os.Stderr
```

**Example:**
```yaml
post_infrastructure:
  - name: "Seed database"
    command: "./scripts/seed.sh"
    target: host
```

Executes: `sh -c "./scripts/seed.sh"` from `/path/to/user-service/`

### Container Execution (`target: container`)

Runs command inside the service's Docker container:

```go
cmd := exec.CommandContext(ctx, "docker", "exec", containerName, "sh", "-c", hook.Command)
cmd.Stdout = os.Stdout
cmd.Stderr = os.Stderr
```

**Example:**
```yaml
post_up:
  - name: "Create admin"
    command: "bin/rails runner 'User.create_admin!'"
    target: container
```

Executes: `docker exec grund-user-service-1 sh -c "bin/rails runner 'User.create_admin!'"`

### Timeout & Error Handling

```go
ctx, cancel := context.WithTimeout(parentCtx, hook.Timeout) // default 10m
defer cancel()

err := cmd.Run()
if err != nil {
    if hook.ContinueOnError {
        log.Warn("Hook '%s' failed, continuing: %s", hook.Name, err)
    } else {
        return fmt.Errorf("hook '%s' failed: %w", hook.Name, err)
        // Stops entire startup/shutdown
    }
}
```

### Environment Variables

Hooks receive the fully resolved environment (same as the service):

```bash
$DATABASE_URL      # postgres://postgres:postgres@localhost:5432/users_db
$REDIS_URL         # redis://localhost:6379
$SQS_ORDER_QUEUE   # http://localhost:4566/000000000000/order-queue
$SERVICE_PORT      # 8080
```

**Note:** `pre_up` hooks have limited env vars since infrastructure isn't provisioned yet.

## Use Cases by Stage

### `pre_up` (before anything starts)

```yaml
pre_up:
  - name: "Generate config"
    command: "envsubst < config.template.yaml > config.yaml"
    target: host
  - name: "Build assets"
    command: "npm run build"
    target: host
  - name: "Validate prerequisites"
    command: "which psql && which docker"
    target: host
```

### `post_infrastructure` (after DBs/queues ready, before services)

```yaml
post_infrastructure:
  - name: "Run migrations"
    command: "psql -h localhost -U postgres -d mydb -f ./migrations/up.sql"
    target: host
  - name: "Seed database"
    command: "./scripts/seed.sh"
    target: host
  - name: "Create SQS test messages"
    command: "aws --endpoint-url=http://localhost:4566 sqs send-message ..."
    target: host
```

### `post_up` (after services healthy)

```yaml
post_up:
  - name: "Create admin user"
    command: "bin/create-admin"
    target: container
  - name: "Warm cache"
    command: "bin/warm-cache"
    target: container
  - name: "Open browser"
    command: "open http://localhost:8080"
    target: host
```

### `pre_down` (before stopping)

```yaml
pre_down:
  - name: "Drain connections"
    command: "bin/drain-connections"
    target: container
  - name: "Export logs"
    command: "docker logs grund-user-service-1 > ./logs/shutdown.log"
    target: host
```

### `post_down` (after containers stopped)

```yaml
post_down:
  - name: "Cleanup temp files"
    command: "rm -rf ./tmp/cache ./tmp/uploads"
    target: host
  - name: "Reset state"
    command: "./scripts/reset-local-state.sh"
    target: host
```

## Implementation Architecture

```
internal/
├── domain/
│   └── service/
│       ├── service.go          # Add Hooks field to Service struct
│       └── hook.go             # NEW: Hook, HookStage, HookTarget value objects
│
├── application/
│   ├── ports/
│   │   └── hook_executor.go    # NEW: HookExecutor interface
│   └── commands/
│       ├── up_command.go       # Add hook execution calls at each stage
│       └── down_command.go     # Add pre_down, post_down hook calls
│
├── infrastructure/
│   ├── hooks/
│   │   └── executor.go         # NEW: HookExecutor implementation
│   └── config/
│       └── service_parser.go   # Parse hooks from grund.yaml
│
└── cli/
    └── up.go                   # No changes (hooks are transparent)
```

## Domain Models

```go
// internal/domain/service/hook.go

package service

import "time"

// HookStage represents when a hook executes
type HookStage string

const (
    HookStagePreUp              HookStage = "pre_up"
    HookStagePostInfrastructure HookStage = "post_infrastructure"
    HookStagePostUp             HookStage = "post_up"
    HookStagePreDown            HookStage = "pre_down"
    HookStagePostDown           HookStage = "post_down"
)

// HookTarget represents where a hook executes
type HookTarget string

const (
    HookTargetHost      HookTarget = "host"
    HookTargetContainer HookTarget = "container"
)

// Hook represents a lifecycle hook configuration
type Hook struct {
    Name            string
    Command         string
    Target          HookTarget
    Timeout         time.Duration  // default 10m
    ContinueOnError bool           // default false
}

// Hooks contains all lifecycle hooks for a service
type Hooks struct {
    PreUp              []Hook
    PostInfrastructure []Hook
    PostUp             []Hook
    PreDown            []Hook
    PostDown           []Hook
}
```

```go
// internal/domain/service/service.go (modified)

type Service struct {
    Name         string
    Type         ServiceType
    Port         Port
    Build        *BuildConfig
    Run          *RunConfig
    Health       HealthConfig
    Dependencies ServiceDependencies
    Environment  Environment
    Hooks        Hooks              // NEW FIELD
}
```

## Future Extensions

- **Global hooks:** Project-level hooks in a root config file
- **Hook templates:** Reusable hook definitions
- **Parallel hooks:** Run multiple hooks concurrently within a stage
- **Hook conditions:** Only run hooks based on environment or flags
