# Grund Configuration Reference

This document describes all configuration files used by Grund.

## Configuration Files Overview

| File | Location | Purpose |
|------|----------|---------|
| `services.yaml` | `~/.grund/services.yaml` | Service registry - lists all services |
| `grund.yaml` | Each service directory | Service configuration |
| `config.yaml` | `~/.grund/config.yaml` | Global user settings |
| `secrets.env` | `~/.grund/secrets.env` | Secret values (API keys, etc.) |
| `docker-compose.yaml` | `~/.grund/tmp/<service>/` | Auto-generated per service |

---

## services.yaml (Service Registry)

The service registry maps service names to their repository locations.

**Default location:** `~/.grund/services.yaml`

Override with `GRUND_CONFIG` environment variable or `--config` flag.

### Schema

```yaml
version: "1"

services:
  <service-name>:
    repo: <git-url>       # Git repository URL
    path: <local-path>    # Local path to service directory
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | Yes | Schema version (currently "1") |
| `services` | map | Yes | Map of service names to locations |
| `services.<name>.repo` | string | No | Git repository URL for cloning |
| `services.<name>.path` | string | Yes | Local filesystem path to service |

### Example

```yaml
version: "1"

services:
  user-service:
    repo: git@github.com:company/user-service.git
    path: /Users/dev/projects/user-service

  order-service:
    repo: git@github.com:company/order-service.git
    path: /Users/dev/projects/order-service

  notification-service:
    path: /Users/dev/projects/notification-service

  # Services without repo can still be used if path exists
  legacy-api:
    path: ../legacy-api
```

### Path Resolution

- **Absolute paths**: Used as-is
- **Relative paths**: Resolved relative to services.yaml location

---

## grund.yaml (Service Configuration)

Each service has a `grund.yaml` file in its root directory.

### Complete Schema

```yaml
version: "1"

service:
  name: <service-name>
  type: <go|python|node>
  port: <port-number>

  build:
    dockerfile: <path-to-dockerfile>
    context: <build-context>

  # OR for interpreted languages
  run:
    command: <run-command>
    hot_reload: <boolean>

  health:
    endpoint: <health-endpoint>
    interval: <duration>
    timeout: <duration>
    retries: <number>

requires:
  services:
    - <service-name>
    - <service-name>

  infrastructure:
    postgres:
      database: <db-name>
      migrations: <migrations-path>
      seed: <seed-file-path>

    mongodb:
      database: <db-name>
      seed: <seed-file-path>

    redis: true

    sqs:
      queues:
        - name: <queue-name>
          dlq: <boolean>

    sns:
      topics:
        - name: <topic-name>
          subscriptions:
            - protocol: sqs
              endpoint: "${sqs.<queue-name>.arn}"
              attributes:
                FilterPolicy: '<json-filter>'
                FilterPolicyScope: <MessageBody|MessageAttributes>

    s3:
      buckets:
        - name: <bucket-name>
          seed: <seed-directory>

# Environment variables (all values support ${placeholder} syntax)
env:
  KEY: value
  ANOTHER_KEY: value
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"
```

### Service Section

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Service name (must match services.yaml) |
| `type` | string | Yes | `go`, `python`, or `node` |
| `port` | integer | Yes | Port the service listens on (1-65535) |

### Build Section

For compiled/containerized services.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `dockerfile` | string | Yes | Path to Dockerfile |
| `context` | string | Yes | Build context directory (usually `.`) |

### Run Section

For interpreted languages (alternative to build).

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `command` | string | Yes | Command to run the service |
| `hot_reload` | boolean | No | Enable hot reloading |

### Health Section

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `endpoint` | string | Yes | | Health check URL path |
| `interval` | duration | No | `5s` | Time between health checks |
| `timeout` | duration | No | `3s` | Timeout for each check |
| `retries` | integer | No | `10` | Retries before unhealthy |

### Requires Section

#### Services

List of other services this service depends on.

```yaml
requires:
  services:
    - user-service
    - auth-service
```

#### Infrastructure

##### PostgreSQL

```yaml
infrastructure:
  postgres:
    database: myservice_db     # Required: database name
    migrations: ./migrations   # Optional: path to migration files
    seed: ./seed.sql          # Optional: path to seed data
```

##### MongoDB

```yaml
infrastructure:
  mongodb:
    database: myservice_db     # Required: database name
    seed: ./seed.json         # Optional: path to seed data
```

##### Redis

```yaml
infrastructure:
  redis: true                  # Just needs to be present
```

##### SQS (Simple Queue Service)

```yaml
infrastructure:
  sqs:
    queues:
      - name: orders           # Queue name
        dlq: true              # Create dead-letter queue
      - name: notifications
        dlq: false
```

##### SNS (Simple Notification Service)

```yaml
infrastructure:
  sns:
    topics:
      - name: order-events
        subscriptions:
          # Subscribe SQS queue to topic with filter policy
          - protocol: sqs
            endpoint: "${sqs.orders.arn}"
            attributes:
              FilterPolicy: '{"event_type":["ORDER_CREATED","ORDER_UPDATED"]}'
              FilterPolicyScope: MessageBody

          # Simple subscription without filters
          - protocol: sqs
            endpoint: "${sqs.analytics.arn}"

      - name: user-events
```

**Subscription Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `protocol` | string | Yes | Subscription protocol (`sqs`) |
| `endpoint` | string | Yes | Queue ARN using template syntax |
| `attributes` | map | No | AWS subscription attributes |

**Common Attributes:**

| Attribute | Description |
|-----------|-------------|
| `FilterPolicy` | JSON filter rules for message routing |
| `FilterPolicyScope` | `MessageBody` or `MessageAttributes` |
| `RawMessageDelivery` | `"true"` to skip SNS envelope |

##### S3 (Simple Storage Service)

```yaml
infrastructure:
  s3:
    buckets:
      - name: uploads
        seed: ./seed-data      # Optional: directory to upload
      - name: documents
```

##### Tunnel (cloudflared or ngrok)

Expose local endpoints to the internet via secure tunnels. Useful for:
- Making LocalStack S3 presigned URLs accessible to cloud LLMs
- Testing webhooks from external services
- Sharing local development servers temporarily

```yaml
infrastructure:
  tunnel:
    provider: cloudflared      # or "ngrok"
    targets:
      - name: localstack       # Identifier for ${tunnel.localstack.url}
        host: "${localstack.host}"
        port: "${localstack.port}"
      - name: api
        host: localhost
        port: "8080"
```

**Tunnel Fields:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `provider` | string | Yes | `cloudflared` or `ngrok` |
| `targets` | list | Yes | Endpoints to expose |
| `targets[].name` | string | Yes | Identifier for placeholders |
| `targets[].host` | string | Yes | Local host (supports placeholders) |
| `targets[].port` | string | Yes | Local port (supports placeholders) |

**Prerequisites:**

- **cloudflared**: Install with `brew install cloudflared` (no account required)
- **ngrok**: Install from https://ngrok.com/download (free account required)

**Usage in env:**

```yaml
env:
  # Use tunnel URL for presigned S3 URLs accessible by external services
  AWS_PUBLIC_ENDPOINT: "${tunnel.localstack.url}"
  WEBHOOK_BASE_URL: "${tunnel.api.url}"
```

### Environment Variables

All values in the `env` section support `${placeholder}` syntax for dynamic resolution at startup.

```yaml
env:
  # Static values
  APP_ENV: development
  LOG_LEVEL: debug
  FEATURE_FLAG: "true"

  # Database connections (dynamic)
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  MONGO_URL: "mongodb://${mongodb.host}:${mongodb.port}/${self.mongodb.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"

  # AWS/LocalStack
  AWS_ENDPOINT: "${localstack.endpoint}"
  AWS_REGION: "${localstack.region}"

  # Queue URLs
  ORDERS_QUEUE_URL: "${sqs.orders.url}"
  ORDERS_DLQ_URL: "${sqs.orders.dlq}"

  # Topic ARNs
  EVENTS_TOPIC_ARN: "${sns.order-events.arn}"

  # Service dependencies
  USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"
  AUTH_SERVICE_URL: "http://${auth-service.host}:${auth-service.port}"
```

#### Secrets (`secrets`)

Declare secrets required by the service. Values are loaded from `~/.grund/secrets.env` or shell environment.

```yaml
secrets:
  OPENAI_API_KEY:
    description: "OpenAI API key for embeddings"
    required: true
  STRIPE_SECRET_KEY:
    description: "Stripe secret key for payments"
    required: true
  ANALYTICS_KEY:
    description: "Mixpanel key for tracking"
    required: false  # Optional secrets don't block startup
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | No | | Human-readable description |
| `required` | boolean | No | `true` | Fail startup if missing |

**Secret Resolution Order** (highest priority first):
1. `~/.grund/secrets.env` file
2. Shell environment variables

**Managing Secrets:**

```bash
# List secrets and their status
grund secrets list user-service

# Generate template for missing secrets
grund secrets init user-service
```

### Complete Example

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
    interval: 5s
    timeout: 3s
    retries: 10

requires:
  services:
    - user-service
    - notification-service
  infrastructure:
    postgres:
      database: orders_db
      migrations: ./migrations
    redis: true
    sqs:
      queues:
        - name: orders
          dlq: true
        - name: order-confirmations
          dlq: false
    sns:
      topics:
        - name: order-events
          subscriptions:
            - protocol: sqs
              endpoint: "${sqs.order-confirmations.arn}"

env:
  APP_ENV: development
  LOG_LEVEL: debug
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"
  ORDERS_QUEUE_URL: "${sqs.orders.url}"
  ORDER_EVENTS_TOPIC_ARN: "${sns.order-events.arn}"
  USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"
  NOTIFICATION_SERVICE_URL: "http://${notification-service.host}:${notification-service.port}"

secrets:
  STRIPE_SECRET_KEY:
    description: "Stripe API key for payment processing"
    required: true
```

---

## ~/.grund/secrets.env (Secrets File)

Store secret values that should not be committed to version control.

### Location

`~/.grund/secrets.env`

### Format

```bash
# Grund secrets file
OPENAI_API_KEY=sk-xxxxxxxxxxxxx
STRIPE_SECRET_KEY=sk_test_xxxxxxxxxxxxx
ANALYTICS_KEY=mp_xxxxxxxxxxxxx
```

### Usage

1. **Generate template**: `grund secrets init <service>` creates placeholders for missing secrets
2. **Check status**: `grund secrets list <service>` shows which secrets are found/missing
3. **Validation**: `grund up` fails fast if required secrets are missing

### Best Practices

- Never commit this file to version control
- Use descriptive comments for each secret
- Share a `secrets.env.example` template with your team (values redacted)

---

## ~/.grund/config.yaml (Global Configuration)

User-level settings for Grund.

### Schema

```yaml
# Default filename to search for
default_services_file: services.yaml

# Default orchestration repository path
default_orchestration_repo: ~/projects/orchestration

# Where to clone service repositories
services_base_path: ~/projects

# Docker settings
docker:
  compose_command: docker compose

# LocalStack settings
localstack:
  endpoint: http://localhost:4566
  region: us-east-1
```

### Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `default_services_file` | string | `services.yaml` | Filename to search for |
| `default_orchestration_repo` | string | | Path to orchestration repo |
| `services_base_path` | string | `~/projects` | Where to clone services |
| `docker.compose_command` | string | `docker compose` | Docker compose command |
| `localstack.endpoint` | string | `http://localhost:4566` | LocalStack endpoint |
| `localstack.region` | string | `us-east-1` | AWS region for LocalStack |

### Path Expansion

- `~` is expanded to home directory
- Relative paths resolved from config file location

---

## Per-Service Compose Files (Auto-Generated)

Generated by `grund up`. **Do not edit manually.**

Files are organized in `~/.grund/tmp/`:

```
~/.grund/tmp/
├── infrastructure/
│   └── docker-compose.yaml    # postgres, mongodb, redis, localstack
├── order-service/
│   └── docker-compose.yaml    # order-service
└── payment-service/
    └── docker-compose.yaml    # payment-service
```

### Infrastructure Compose Example

`~/.grund/tmp/infrastructure/docker-compose.yaml`:

```yaml
# AUTO-GENERATED by grund - DO NOT EDIT
services:
  postgres:
    image: postgres:15-alpine
    container_name: grund-postgres
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres-data:/var/lib/postgresql/data
    networks:
      - grund-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: grund-redis
    ports:
      - "6379:6379"
    networks:
      - grund-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

networks:
  grund-network:
    driver: bridge
    name: grund-network

volumes:
  postgres-data: {}
```

### Service Compose Example

`~/.grund/tmp/order-service/docker-compose.yaml`:

```yaml
# AUTO-GENERATED by grund - DO NOT EDIT
services:
  order-service:
    container_name: grund-order-service
    build:
      context: /Users/dev/projects/order-service
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      APP_ENV: development
      DATABASE_URL: postgres://postgres:postgres@postgres:5432/orders_db
      REDIS_URL: redis://redis:6379
    networks:
      - grund-network
    healthcheck:
      test: ["CMD-SHELL", "curl -sf http://localhost:8080/health || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10

networks:
  grund-network:
    external: true
    name: grund-network
```

---

## Environment Variable Reference

### Available Placeholders

| Category | Placeholder | Description |
|----------|-------------|-------------|
| **PostgreSQL** | `${postgres.host}` | Container hostname (`postgres`) |
| | `${postgres.port}` | Port (`5432`) |
| | `${postgres.database}` | Database name |
| | `${postgres.username}` | Username (`postgres`) |
| | `${postgres.password}` | Password (`postgres`) |
| **MongoDB** | `${mongodb.host}` | Container hostname (`mongodb`) |
| | `${mongodb.port}` | Port (`27017`) |
| | `${mongodb.database}` | Database name |
| **Redis** | `${redis.host}` | Container hostname (`redis`) |
| | `${redis.port}` | Port (`6379`) |
| **LocalStack** | `${localstack.endpoint}` | Full endpoint URL |
| | `${localstack.host}` | Hostname (`localstack`) |
| | `${localstack.port}` | Port (`4566`) |
| | `${localstack.region}` | AWS region |
| | `${localstack.account_id}` | Account ID (`000000000000`) |
| **SQS** | `${sqs.<name>.url}` | Queue URL |
| | `${sqs.<name>.arn}` | Queue ARN |
| | `${sqs.<name>.dlq}` | Dead-letter queue URL |
| | `${sqs.<name>.name}` | Queue name |
| **SNS** | `${sns.<name>.arn}` | Topic ARN |
| | `${sns.<name>.name}` | Topic name |
| **S3** | `${s3.<name>.url}` | Bucket URL |
| | `${s3.<name>.name}` | Bucket name |
| **Tunnel** | `${tunnel.<name>.url}` | Public HTTPS URL |
| | `${tunnel.<name>.host}` | Public hostname only |
| **Services** | `${<service>.host}` | Service container name |
| | `${<service>.port}` | Service port |
| **Self** | `${self.host}` | Current service name |
| | `${self.port}` | Current service port |
| | `${self.postgres.database}` | This service's database |
| | `${self.mongodb.database}` | This service's MongoDB |

---

## See Also

- [CLI Commands](./cli-commands.md)
- [Algorithms](./algorithms.md)
- [Architecture Overview](./architecture.md)
- [Adding New Infrastructure](./adding-new-infrastructure.md)
