<p align="center">
  <img src="docs/assets/cover.jpeg" alt="Grund - Local Development Orchestration Tool" width="100%">
</p>

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)
[![Go Report Card](https://img.shields.io/badge/Go%20Report-A-success?style=for-the-badge&logo=go)](https://goreportcard.com/report/github.com/GrundIO/grund)
[![GitHub Release](https://img.shields.io/github/v/release/GrundIO/grund?style=for-the-badge&logo=github)](https://github.com/GrundIO/grund/releases)

**Grund** is a CLI tool for local microservice development. One command spins up any service with all its dependencies—other services, databases, queues, and caches—in the correct order.

## Table of Contents

- [Why Grund?](#why-grund)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Getting Started](#getting-started)
- [Integration Guide](#integration-guide)
- [Commands Reference](#commands-reference)
- [Configuration Reference](#configuration-reference)
- [Shell Completion](#shell-completion)
- [Troubleshooting](#troubleshooting)

## Why Grund?

**The Problem:** You want to run `notification-service` locally. But it needs `order-service`, which needs `user-service`. All three need PostgreSQL. Notification also needs Redis and SQS. You spend 30 minutes writing docker-compose files, setting up LocalStack, and figuring out the right startup order.

**The Solution:** With Grund, each service declares its own dependencies. Run one command:

```bash
grund up notification-service
```

Grund automatically:
- Resolves the full dependency tree (`notification` → `order` → `user`)
- Starts PostgreSQL, Redis, and LocalStack
- Creates the SQS queues and databases
- Boots services in the correct order
- Injects connection URLs into each service

![Grund Full Workflow](docs/assets/grund-full-workflow.gif)

## Prerequisites

- **Docker** and **Docker Compose** v2+
- **Go 1.21+** (for building from source)

## Installation

### Using Go Install

```bash
go install github.com/GrundIO/grund@latest
```

### From Source

```bash
git clone https://github.com/GrundIO/grund.git
cd grund
make install
```

### Verify Installation

```bash
grund --version
```

## Getting Started

### Quick Start (Recommended)

> [!TIP]
> Run `grund init` for an interactive setup wizard that handles all configuration automatically.

```bash
grund init
```

![Grund Init](docs/assets/grund-init.gif)

This walks you through:
1. **Global config** - Creates `~/.grund/config.yaml`
2. **Services setup** - Scans your projects folder and registers services
3. **AI assistant skills** - Installs skills for Claude Code and/or Cursor

### Manual Setup

If you prefer to set up Grund manually instead of using the interactive wizard:

#### Step 1: Create Global Configuration

Create `~/.grund/config.yaml` with your services:

```yaml
version: "1"

services:
  user-service:
    repo: git@github.com:mycompany/user-service.git
    path: ~/projects/user-service

  payment-service:
    repo: git@github.com:mycompany/payment-service.git
    path: ~/projects/payment-service
```

#### Step 2: Initialize Each Service

In each service repository, create a `grund.yaml`:

```bash
cd ~/projects/user-service
grund service init
```

The interactive wizard will guide you through:
- Service name, type, and port
- Infrastructure requirements (PostgreSQL, Redis, etc.)
- Service dependencies

![Grund Service Init](docs/assets/grund-service-init.gif)

#### Step 3: Start Services

```bash
grund up user-service
```

![Grund Up](docs/assets/grund-up.gif)

## Integration Guide

### Adding Grund to an Existing Service

1. **Navigate to your service directory:**
   ```bash
   cd ~/projects/my-service
   ```

2. **Initialize Grund:**
   ```bash
   grund service init
   ```

3. **Or create `grund.yaml` manually** - see [Configuration Reference](#configuration-reference) for full schema.

4. **Add infrastructure as needed:**
   ```bash
   grund service add postgres my_service_db
   grund service add redis
   grund service add queue events-queue
   ```

5. **Register in `~/.grund/config.yaml`:**

   > [!IMPORTANT]
   > Services must be registered before running `grund up`. The path must point to a directory containing `grund.yaml`.

   ```yaml
   services:
     my-service:
       repo: git@github.com:mycompany/my-service.git
       path: ~/projects/my-service
   ```

### Adding Dependencies Between Services

If `payment-service` depends on `user-service`:

```bash
cd ~/projects/payment-service
grund service add dependency user-service
```

This adds to `grund.yaml`:
```yaml
requires:
  services:
    - user-service

env:
  USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"
```

<details>
<summary><h3>Infrastructure Options</h3></summary>

| Type | Command | Notes |
|------|---------|-------|
| PostgreSQL | `grund service add postgres myapp_db` | |
| MongoDB | `grund service add mongodb myapp_db` | |
| Redis | `grund service add redis` | |
| SQS | `grund service add queue order-events` | Via LocalStack |
| SNS | `grund service add topic notifications` | Via LocalStack |
| S3 | `grund service add bucket uploads` | Via LocalStack |
| Tunnel | `grund service add tunnel my-tunnel` | cloudflared or ngrok |

> [!NOTE]
> Tunnels start **before** services, so `${tunnel.localstack.url}` is always available in your `env` section.

</details>

## Commands Reference

| Command | Description |
|---------|-------------|
| `grund up <service...>` | Start services and their dependencies |
| `grund down` | Stop all running services |
| `grund status` | Show running services and their status |
| `grund logs [service...]` | View service logs |
| `grund restart <service>` | Restart a specific service |
| `grund reset [--volumes]` | Stop services and optionally clean up volumes |
| `grund init` | Interactive setup wizard |
| `grund config show [service]` | Show configuration and settings |
| `grund service init` | Initialize `grund.yaml` in current directory |
| `grund service add <type>` | Add infrastructure to existing service |

> [!TIP]
> For detailed command documentation with examples and flags, see the [CLI Commands Reference](docs/wiki/cli-commands.md).

## Configuration Reference

### Service Configuration (`grund.yaml`)

<details>
<summary>📋 Click to expand full grund.yaml example</summary>

```yaml
version: "1"

service:
  name: my-service              # Service name (must match services.yaml)
  type: go                      # go, python, or node
  port: 8080                    # Port the service listens on
  build:
    dockerfile: Dockerfile      # Path to Dockerfile
    context: .                  # Build context
  health:
    endpoint: /health           # Health check endpoint
    interval: 5s                # Check interval
    timeout: 3s                 # Request timeout
    retries: 10                 # Retries before unhealthy

requires:
  services:                     # Service dependencies
    - user-service
    - auth-service
  infrastructure:
    postgres:
      database: my_db           # Database name
      migrations: ./migrations  # Optional: migrations path
    mongodb:
      database: my_mongo_db
    redis: true
    sqs:
      queues:
        - name: order-queue
          dlq: true             # Create dead-letter queue
        - name: notification-queue
    sns:
      topics:
        - name: order-events
    s3:
      buckets:
        - name: uploads
        - name: exports
    tunnel:
      provider: cloudflared     # or "ngrok"
      targets:
        - name: localstack
          host: "${localstack.host}"
          port: "${localstack.port}"

env:                            # Environment variables (supports ${placeholder} syntax)
  APP_ENV: development
  LOG_LEVEL: debug
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"
  USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"
  AWS_ENDPOINT: "http://${localstack.host}:${localstack.port}"
```

</details>

### Global Configuration (`~/.grund/config.yaml`)

```yaml
version: "1"

services:
  user-service:
    path: ~/projects/user-service
    repo: git@github.com:mycompany/user-service.git

# Optional - uses defaults if not specified
docker:
  compose_command: docker compose    # default

localstack:
  endpoint: http://localhost:4566    # default
  region: us-east-1                  # default
```

### Environment Variable Interpolation

Use `${placeholder}` syntax in `env` to reference infrastructure and services.

<details>
<summary>📋 Click to expand full placeholder reference</summary>

| Placeholder | Description |
|-------------|-------------|
| `${postgres.host}` | PostgreSQL hostname |
| `${postgres.port}` | PostgreSQL port (5432) |
| `${mongodb.host}` | MongoDB hostname |
| `${mongodb.port}` | MongoDB port (27017) |
| `${redis.host}` | Redis hostname |
| `${redis.port}` | Redis port (6379) |
| `${localstack.endpoint}` | LocalStack endpoint URL |
| `${localstack.host}` | LocalStack hostname |
| `${localstack.port}` | LocalStack port (4566) |
| `${localstack.region}` | AWS region (us-east-1) |
| `${localstack.account_id}` | AWS account ID (000000000000) |
| `${localstack.access_key_id}` | AWS access key ID |
| `${localstack.secret_access_key}` | AWS secret access key |
| `${sqs.<queue>.url}` | SQS queue URL |
| `${sqs.<queue>.arn}` | SQS queue ARN |
| `${sqs.<queue>.dlq}` | SQS dead-letter queue URL |
| `${sns.<topic>.arn}` | SNS topic ARN |
| `${s3.<bucket>.url}` | S3 bucket URL |
| `${<service>.host}` | Dependent service hostname |
| `${<service>.port}` | Dependent service port |
| `${self.postgres.database}` | This service's database name |
| `${tunnel.<name>.url}` | Public tunnel URL (https://...) |
| `${tunnel.<name>.host}` | Public tunnel hostname |

</details>

## Shell Completion

Enable tab completion for commands and service names.

### Zsh
```bash
# Add to ~/.zshrc
eval "$(grund completion zsh)"
```

### Bash
```bash
# Add to ~/.bashrc
eval "$(grund completion bash)"
```

### Fish
```bash
grund completion fish | source
```

After setup:
```bash
grund u<TAB>        → grund up
grund up pay<TAB>   → grund up payment-service
grund --<TAB>       → --config  --verbose  --help
```

## Troubleshooting

Use this decision tree to diagnose common issues:

```mermaid
flowchart TD
    A[Issue?] --> B{Services not<br/>starting?}
    B -->|Yes| C{Config found?}
    C -->|No| D["Run: grund init"]
    C -->|Yes| E{Docker running?}
    E -->|No| F[Start Docker Desktop]
    E -->|Yes| G["Run: grund -v up service"]

    B -->|No| H{Container<br/>unhealthy?}
    H -->|Yes| I["Run: grund logs service"]

    H -->|No| J{Port in use?}
    J -->|Yes| K["Run: grund reset"]

    J -->|No| L{LocalStack<br/>issues?}
    L -->|Yes| M["Run: grund reset --volumes"]

    L -->|No| N["Run: grund status"]

    style D fill:#4CAF50,color:#fff
    style F fill:#4CAF50,color:#fff
    style G fill:#2196F3,color:#fff
    style I fill:#2196F3,color:#fff
    style K fill:#FFC107,color:#000
    style M fill:#FFC107,color:#000
    style N fill:#9E9E9E,color:#fff
```

---

### "services.yaml not found"

Grund searches for configuration in this order:

```mermaid
flowchart LR
    A["1️⃣ --config flag"] --> B["2️⃣ GRUND_CONFIG env"]
    B --> C["3️⃣ Local search<br/>(up to 5 dirs)"]
    C --> D["4️⃣ ~/.grund/config.yaml"]
    D --> E["❌ Error"]

    style A fill:#1976D2,color:#fff
    style B fill:#388E3C,color:#fff
    style C fill:#F57C00,color:#fff
    style D fill:#7B1FA2,color:#fff
    style E fill:#D32F2F,color:#fff
```

**Fix options:**

Set the `GRUND_CONFIG` environment variable:
```bash
export GRUND_CONFIG=/path/to/services.yaml
```

Or use the `--config` flag:
```bash
grund --config=/path/to/services.yaml up my-service
```

Or run the setup wizard:
```bash
grund init
```

### "Container is unhealthy"

Check container logs:
```bash
grund logs <service>
docker logs grund-<service>
```

### "Port already in use"

Stop existing services:
```bash
grund down
grund reset
```

Or check what's using the port:
```bash
lsof -i :8080
```

### "LocalStack resources not created"

Ensure LocalStack is healthy before provisioning:
```bash
grund status
```

If issues persist:
```bash
grund reset --volumes
grund up <service>
```

### View verbose output

Use `-v` flag for debug information:
```bash
grund -v up my-service
```

## Contributors

Thanks to these wonderful people who have contributed to Grund:

<table>
  <tr>
    <td align="center">
      <a href="https://github.com/vivekkundariya">
        <img src="https://github.com/vivekkundariya.png" width="80px;" alt="Vivek Kundariya"/><br />
        <sub><b>Vivek Kundariya</b></sub>
      </a>
    </td>
  </tr>
</table>

## License

MIT
