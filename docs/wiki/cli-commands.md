# Grund CLI Commands Reference

This document provides detailed documentation for all Grund CLI commands.

> [!TIP]
> Enable shell completion for faster command entry. See [Shell Completion](#shell-completion) section below.

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to services registry file (overrides auto-detection) |
| `--verbose` | `-v` | Enable verbose/debug output |
| `--help` | `-h` | Show help for any command |

## Configuration Resolution

Grund automatically finds configuration in this priority order:

1. **CLI flag**: `--config /path/to/services.yaml`
2. **Environment variable**: `GRUND_CONFIG=/path/to/services.yaml`
3. **Local file search**: Searches current and parent directories (up to 5 levels) for:
   - `services.yaml`
   - `grund-services.yaml`
   - `grund.services.yaml`
4. **Global config**: Uses `default_orchestration_repo` from `~/.grund/config.yaml`

---

## Commands

### `grund up`

Start services and their dependencies.

![Grund Up](../assets/grund-up.gif)

```bash
grund up [services...] [flags]
```

**Arguments:**
- `services` (required): One or more service names to start

**Flags:**
| Flag | Description |
|------|-------------|
| `--no-deps` | Only start specified services, skip dependencies |
| `--infra-only` | Only start infrastructure (postgres, redis, etc.), skip application services |

> [!NOTE]
> Use `--infra-only` when you want to run your service locally (outside Docker) but still need the infrastructure.
| `--build` | Force rebuild containers |
| `--local` | Run service locally (not in container) - *planned* |

**What it does:**
1. Loads service configurations from `grund.yaml` files
2. Builds dependency graph (circular dependencies are allowed)
3. Loads all transitive dependencies
4. Aggregates infrastructure requirements from all services
5. **Starts tunnels** (if configured) - cloudflared/ngrok for exposing LocalStack, etc.
6. Generates per-service compose files in `~/.grund/tmp/` (with tunnel URLs resolved)
7. Starts infrastructure containers (postgres, mongodb, redis, localstack)
8. Waits for infrastructure health checks
9. Provisions resources (creates databases, SQS queues, SNS topics, S3 buckets)
10. Starts all services in parallel (services handle reconnection)

**Examples:**
```bash
# Start a single service with all dependencies
grund up user-service

# Start multiple services
grund up user-service order-service

# Start only the specified service (skip dependencies)
grund up user-service --no-deps

# Only start infrastructure, useful for local development
grund up user-service --infra-only

# Force rebuild containers
grund up user-service --build

# Verbose output for debugging
grund up user-service -v

# Service with tunnel (LocalStack exposed via cloudflared)
# Tunnel URL available as ${tunnel.localstack.url} in env
grund up s3-upload-service
```

**Tunnel Output Example:**
```
→ Starting tunnels...
  localstack: https://abc-xyz.trycloudflare.com -> localhost:4566
✓ Tunnels started
→ Generating docker-compose configuration...
```

---

### `grund sync`

Sync service repositories defined in services.yaml. Missing repos are cloned, existing repos are pulled to the latest.

```bash
grund sync [services...] [flags]
```

**Arguments:**
- `services` (optional): One or more service names to sync. If omitted, syncs all services that have a `repo` field.

**Flags:**
| Flag | Description |
|------|-------------|
| `--no-pull` | Only clone missing repositories, skip pulling existing ones |

**What it does:**
1. Reads all service entries from `services.yaml`
2. Filters to requested services (or all with a `repo` field)
3. For each service:
   - **Path doesn't exist** → runs `git clone`
   - **Path exists + is git repo** → runs `git pull`
   - **Path exists + is git repo + `--no-pull`** → skips
   - **Path exists + NOT a git repo** → reports error
4. Prints a summary of cloned, pulled, skipped, and errored services

**Examples:**
```bash
# Sync all service repositories (clone missing + pull existing)
grund sync

# Sync specific services
grund sync user-service order-service

# Only clone missing repos, don't pull existing
grund sync --no-pull

# Clone missing specific service only
grund sync user-service --no-pull
```

**Sample output:**
```
[INFO] Syncing 3 service(s)...
  → user-service: cloning git@github.com:company/user-service.git...
  → order-service: pulling latest...
  → notification-service: cloning git@github.com:company/notification-service.git...

╭──────────────────────┬───────────┬──────────────────────────────────────────┬─────────╮
│ Service              │ Status    │ Path                                     │ Comment │
├──────────────────────┼───────────┼──────────────────────────────────────────┼─────────┤
│ user-service         │ ● cloned  │ /Users/dev/projects/user-service         │ -       │
│ order-service        │ ● pulled  │ /Users/dev/projects/order-service        │ -       │
│ notification-service │ ● cloned  │ /Users/dev/projects/notification-service │ -       │
╰──────────────────────┴───────────┴──────────────────────────────────────────┴─────────╯

[INFO] Cloned: 2, Pulled: 1, Skipped: 0, Errors: 0
```

> [!TIP]
> Run `grund sync` after `grund init` to set up all service repositories in one command.

---

### `grund graph`

Visualize the service dependency graph.

```bash
grund graph [services...] [flags]
```

**Arguments:**
- `services` (optional): One or more service names. If omitted, graphs all registered services.

**Flags:**
| Flag | Short | Description |
|------|-------|-------------|
| `--infra` | | Show infrastructure dependencies as separate nodes (postgres, redis, sqs, etc.) |
| `--format` | | Output format: `svg` (default) or `dot` |
| `--output` | `-o` | Output file path (default: `graph.svg`) |

**What it does:**
1. Loads all service configurations (skips services with missing `grund.yaml`)
2. Builds the dependency graph (circular dependencies are handled gracefully)
3. Renders the graph as SVG (or DOT text)
4. Mutual dependencies are shown as a single bidirectional arrow
5. With `--infra`, infrastructure resources become separate nodes — services sharing the same resource (e.g. same postgres database) connect to the same node
6. SQS edges are bidirectional to reflect produce/consume nature

**Examples:**
```bash
# Render full dependency graph to graph.svg
grund graph

# Render to custom file
grund graph --output deps.svg

# Graph for a specific service and its transitive dependencies
grund graph user-service

# Include infrastructure nodes (postgres, redis, sqs, etc.)
grund graph --infra

# Print DOT format to stdout (pipe to graphviz tools)
grund graph --format dot

# Pipe DOT to external graphviz for PDF
grund graph --format dot | dot -Tpdf -o graph.pdf
```

**Sample DOT output:**
```dot
digraph dependencies {
  rankdir=LR;
  node [shape=box, style="filled,rounded", fillcolor="#E8F4FD", ...];
  "user-service" -> "auth-service";
  "auth-service" -> "token-service";
}
```

---

### `grund down`

Stop all running services and infrastructure.

```bash
grund down
```

**What it does:**
1. Executes `docker compose down`
2. Stops all containers defined in the compose file
3. Removes containers and networks (volumes preserved)

**Examples:**
```bash
# Stop all services
grund down
```

---

### `grund status`

Show the status of all services and infrastructure.

![Grund Status](../assets/grund-status.gif)

```bash
grund status
```

**Output:**
Displays a table with:
- Service name
- Status (running/exited/not running)
- Health indicator (color-coded)

**Examples:**
```bash
# Show all service statuses
grund status
```

**Sample output:**
```
╭─────────────────┬──────────────╮
│ Service         │ Status       │
├─────────────────┼──────────────┤
│ postgres        │ ● running    │
│ redis           │ ● running    │
│ user-service    │ ● running    │
│ order-service   │ ○ not running│
╰─────────────────┴──────────────╯
```

---

### `grund logs`

View logs from services.

![Grund Logs](../assets/grund-logs.gif)

```bash
grund logs [services...] [flags]
```

**Arguments:**
- `services` (optional): One or more services to view logs for. If omitted, shows aggregated logs from all services.

**Flags:**
| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--follow` | `-f` | false | Follow log output (like `tail -f`) |
| `--tail` | | 100 | Number of lines to show from the end |

**Examples:**
```bash
# View all service logs
grund logs

# View logs for a specific service
grund logs user-service

# View logs for multiple services
grund logs user-service order-service

# Follow logs in real-time
grund logs user-service -f

# Follow logs for multiple services
grund logs user-service order-service -f

# Show last 500 lines
grund logs user-service --tail 500

# Follow all logs
grund logs -f
```

---

### `grund restart`

Restart a specific service.

```bash
grund restart <service>
```

**Arguments:**
- `service` (required): The service to restart

**What it does:**
1. Executes `docker compose restart <service>`
2. Restarts the container without rebuilding

**Examples:**
```bash
# Restart a service after code changes
grund restart user-service
```

---

### `grund reset`

Stop all services and optionally clean up resources.

```bash
grund reset [flags]
```

> [!WARNING]
> Using `--volumes` will **permanently delete** all database data. This cannot be undone!

**Flags:**
| Flag | Description |
|------|-------------|
| `--volumes` | Remove named volumes (database data will be lost) |
| `--images` | Remove locally built images |

**Examples:**
```bash
# Stop all services (preserves data)
grund reset

# Stop and remove volumes (fresh start, loses all data)
grund reset --volumes

# Full cleanup: stop, remove volumes and images
grund reset --volumes --images
```

---

### `grund init`

Interactive setup wizard for first-time users.

![Grund Init](../assets/grund-init.gif)

```bash
grund init
```

**What it does:**
Walks through a 3-step setup process:

1. **Global config** - Creates `~/.grund/config.yaml`
2. **Services setup** - Scans your projects folder and registers services
3. **AI assistant skills** - Installs skills for Claude Code and/or Cursor

**Examples:**
```bash
# Run the interactive setup wizard
grund init
```

---

### `grund config show`

Show Grund configuration and service details.

![Grund Config Show](../assets/grund-config-show.gif)

```bash
grund config show [service]
```

**Arguments:**
- `service` (optional): Service name to show detailed configuration for

**Without arguments:**
Shows:
- Services file location
- Orchestration root directory
- Global config path
- Global settings with source (config vs default)
- List of registered services

**With service name:**
Shows:
- Service name, type, and port
- Dependencies (other services)
- Infrastructure requirements
- Resolved environment variables

**Examples:**
```bash
# Show overall Grund setup
grund config show

# Show specific service configuration
grund config show user-service
```

---

### `grund service init`

Initialize Grund configuration for a new service.

![Grund Service Init](../assets/grund-service-init.gif)

```bash
grund service init
```

**What it does:**
Interactive wizard that creates a `grund.yaml` file in the current directory.

**Prompts for:**
1. **Service basics**: Name, type (go/python/node), port
2. **Build configuration**: Dockerfile path, health check endpoint
3. **Infrastructure**: PostgreSQL, MongoDB, Redis, SQS, SNS, S3
4. **Service dependencies**: Other services this service depends on

**Examples:**
```bash
# Initialize in a service directory
cd my-service
grund service init
```

---

### `grund service add`

Add infrastructure to an existing service.

![Grund Service Add](../assets/grund-service-add.gif)

```bash
grund service add <type> [name]
```

**Supported types:**
- `postgres <database>` - PostgreSQL database
- `mongodb <database>` - MongoDB database
- `redis` - Redis cache
- `queue <name>` - SQS queue (with optional DLQ)
- `topic <name>` - SNS topic
- `bucket <name>` - S3 bucket
- `tunnel <name>` - Tunnel (cloudflared/ngrok)
- `dependency <service>` - Service dependency

**Examples:**
```bash
# Add PostgreSQL to current service
grund service add postgres mydb

# Add SQS queue
grund service add queue orders

# Add S3 bucket
grund service add bucket uploads

# Add service dependency
grund service add dependency user-service

# Add tunnel for LocalStack
grund service add tunnel localstack
```

---

### `grund service validate`

Validate `grund.yaml` configuration.

```bash
grund service validate
```

**What it does:**
Checks the `grund.yaml` in the current directory for errors.

**Examples:**
```bash
cd my-service
grund service validate
```

---

### `grund secrets`

Manage secrets required by services.

**Subcommands:**

#### `grund secrets list <service...>`

Show all secrets required by the specified services and their dependencies.

```bash
$ grund secrets list user-service

Secrets for user-service (and dependencies):

╭──────────────────────┬───────────────┬─────────────────────────────────╮
│ Secret               │ Status        │ Description                     │
├──────────────────────┼───────────────┼─────────────────────────────────┤
│ OPENAI_API_KEY       │ ✓ found (file)│ OpenAI API key for embeddings   │
│ STRIPE_SECRET_KEY    │ ✗ missing     │ Stripe secret key for payments  │
│ ANALYTICS_KEY        │ ○ optional    │ Mixpanel key for tracking       │
╰──────────────────────┴───────────────┴─────────────────────────────────╯

Source: ~/.grund/secrets.env (3 keys loaded)

Missing required secrets: 1
Run 'grund secrets init' to generate a template.
```

#### `grund secrets init <service...>`

Generate `~/.grund/secrets.env` with placeholders for missing secrets.

```bash
$ grund secrets init user-service

Created ~/.grund/secrets.env with 2 placeholders:

  OPENAI_API_KEY
  STRIPE_SECRET_KEY

Edit the file and add your secret values.
```

If the file already exists, only missing secrets are appended. Existing values are preserved.

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (check message) |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GRUND_CONFIG` | Path to services registry file |
| `GRUND_VERBOSE` | Enable verbose output (set to `1` or `true`) |

---

## See Also

- [Architecture Overview](./architecture.md)
- [Algorithms](./algorithms.md)
- [Configuration Reference](./configuration.md)
- [Adding New Infrastructure](./adding-new-infrastructure.md)
