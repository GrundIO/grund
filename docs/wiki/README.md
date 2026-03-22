# Grund Documentation

Welcome to the Grund documentation. Grund is a local development orchestration tool that enables developers to selectively spin up services and their dependencies with a single command.

## Quick Start

```bash
# Start a service with all its dependencies
grund up user-service

# Check status of running services
grund status

# View logs
grund logs -f

# Stop everything
grund down
```

## Documentation

### Core Documentation

| Document | Description |
|----------|-------------|
| [Architecture](./architecture.md) | System design, DDD layers, SOLID principles, design patterns |
| [CLI Commands](./cli-commands.md) | Complete command reference with examples |
| [Configuration](./configuration.md) | All configuration files and schemas |
| [Algorithms](./algorithms.md) | Dependency resolution, topological sort, env resolution |

### Guides

| Document | Description |
|----------|-------------|
| [Adding New Infrastructure](./adding-new-infrastructure.md) | Step-by-step guide to extend Grund with new infrastructure types |

### Decision Logs

| Document | Description |
|----------|-------------|
| [System Design](../plans/2026-01-22-localdev-system-design.md) | Original architecture decisions and rationale |
| [Testing Strategy](../plans/2026-01-22-testing-strategy.md) | Testing approach and patterns |
| [Secrets Management](../plans/2026-01-28-secrets-management-design.md) | Secrets handling design |
| [SNS Filter Policy](../plans/2026-01-28-sns-filter-policy-design.md) | SNS subscription filtering |

## Key Concepts

### Service Registry (`services.yaml`)

Maps service names to their local paths:

```yaml
version: "1"
services:
  user-service:
    path: /path/to/user-service
  order-service:
    path: /path/to/order-service
```

### Service Configuration (`grund.yaml`)

Each service defines its requirements:

```yaml
version: "1"
service:
  name: order-service
  type: go
  port: 8080
  build:
    dockerfile: Dockerfile
    context: .
  health:
    endpoint: /health

requires:
  services:
    - user-service
  infrastructure:
    postgres:
      database: orders_db
    redis: true
    sqs:
      queues:
        - name: orders
          dlq: true

env:
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
```

### How It Works

```
┌─────────────────────────────────────────────────────────────┐
│  grund up order-service                                     │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  1. Load service configs from grund.yaml files              │
│  2. Build dependency graph                                  │
│  3. Detect circular dependencies                            │
│  4. Topological sort for startup order                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  5. Aggregate infrastructure requirements                   │
│  6. Generate per-service compose files                      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  7. Start infrastructure (postgres, redis, localstack)      │
│  8. Wait for health checks                                  │
│  9. Provision resources (databases, queues, buckets)        │
│  10. Start application services in dependency order         │
└─────────────────────────────────────────────────────────────┘
```

## Supported Infrastructure

| Infrastructure | Container | Use Case |
|---------------|-----------|----------|
| PostgreSQL | `postgres:15-alpine` | Relational database |
| MongoDB | `mongo:6` | Document database |
| Redis | `redis:7-alpine` | Cache, pub/sub |
| LocalStack | `localstack/localstack` | AWS services locally |

### AWS Services (via LocalStack)

- **SQS** - Message queues with optional DLQ
- **SNS** - Topics with SQS subscriptions
- **S3** - Object storage buckets

## Environment Variable Placeholders

Dynamic values resolved at startup. All values in `env` support `${placeholder}` syntax:

```yaml
env:
  # Infrastructure
  DATABASE_URL: "postgres://${postgres.host}:${postgres.port}/${self.postgres.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"

  # AWS/LocalStack
  QUEUE_URL: "${sqs.orders.url}"
  TOPIC_ARN: "${sns.events.arn}"

  # Service dependencies
  USER_API: "http://${user-service.host}:${user-service.port}"
```

See [Configuration Reference](./configuration.md#environment-variable-reference) for all placeholders.

## Project Structure

```
grund/
├── main.go              # CLI entry point
├── internal/
│   ├── cli/             # Cobra commands
│   ├── application/     # Use cases (CQRS)
│   │   ├── commands/    # Write operations
│   │   ├── queries/     # Read operations
│   │   ├── ports/       # Interfaces
│   │   └── wiring/      # Dependency injection
│   ├── domain/          # Business logic
│   │   ├── service/     # Service entity
│   │   ├── infrastructure/
│   │   └── dependency/  # Graph algorithms
│   ├── infrastructure/  # External adapters
│   │   ├── docker/      # Orchestration
│   │   ├── aws/         # LocalStack
│   │   ├── config/      # Repositories
│   │   └── generator/   # Compose generation
│   ├── config/          # Config parsing
│   └── ui/              # Logging, colors
└── docs/                # This documentation
```

## Contributing

When adding new features:

1. **New Infrastructure**: Follow [Adding New Infrastructure](./adding-new-infrastructure.md)
2. **New Commands**: Add to `internal/cli/`, create handler in `internal/application/commands/`
3. **Architecture Changes**: Update [Architecture](./architecture.md) documentation

## See Also

- [GitHub Repository](https://github.com/GrundIO/grund)
- [Architecture Deep Dive](./architecture.md)
- [Full CLI Reference](./cli-commands.md)
