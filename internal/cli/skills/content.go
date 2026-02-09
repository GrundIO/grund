package skills

// GrundSkillContent contains the SKILL.md content for AI assistants
const GrundSkillContent = `---
name: using-grund
description: Orchestrate local development services with Grund CLI. Use when starting services, managing dependencies, configuring infrastructure (Postgres, MongoDB, Redis, SQS, SNS, S3), or working with grund.yaml files.
allowed-tools:
  - Bash(grund *)
---

# Using Grund

Grund is a local development orchestration tool. It starts services with their dependencies (databases, queues, other services) using Docker Compose.

**Config location:** ` + "`~/.grund/`" + `
- ` + "`services.yaml`" + ` - Service registry
- ` + "`secrets.env`" + ` - Secret values (API keys)
- ` + "`config.yaml`" + ` - Global settings

## Quick Reference

### Cloning Repositories
` + "```bash" + `
grund clone                     # Clone all service repos from registry
grund clone svc-a svc-b         # Clone specific services
grund clone --sync              # Clone missing + pull existing repos
grund clone svc-a --sync        # Pull latest for a specific service
` + "```" + `

### Starting Services
` + "```bash" + `
grund up <service>              # Start service with all dependencies
grund up svc-a svc-b            # Start multiple services
grund up <service> --no-deps    # Start without dependencies
grund up <service> --infra-only # Only start infrastructure
` + "```" + `

### Checking Status
` + "```bash" + `
grund status                    # Show all services and URLs
grund logs <service>            # View logs
grund logs -f                   # Follow all logs
` + "```" + `

### Managing Services
` + "```bash" + `
grund down                      # Stop all services
grund restart <service>         # Restart specific service
grund reset                     # Stop and clean up
grund reset -v                  # Also remove volumes (data)
` + "```" + `

### Secrets
` + "```bash" + `
grund secrets list <service>    # Check which secrets are needed
grund secrets init <service>    # Generate ~/.grund/secrets.env template
` + "```" + `

### Configuration
` + "```bash" + `
grund config init               # Initialize global config (~/.grund/)
grund config show               # Show global configuration
grund config show <service>     # Show resolved config and env vars
` + "```" + `

### Service Configuration
` + "```bash" + `
grund service init              # Create grund.yaml interactively
grund service add postgres <db> # Add PostgreSQL database
grund service add mongodb <db>  # Add MongoDB database
grund service add redis         # Add Redis cache
grund service add queue <name>  # Add SQS queue
grund service add topic <name>  # Add SNS topic
grund service add bucket <name> # Add S3 bucket
grund service add tunnel <name> # Add tunnel for external access
grund service add dependency <svc> # Add service dependency
grund service validate          # Validate grund.yaml
` + "```" + `

---

## Configuration (grund.yaml)

### Minimal Example
` + "```yaml" + `
version: "1"
service:
  name: my-service
  type: go
  port: 8080
  build:
    dockerfile: Dockerfile
    context: .
  health:
    endpoint: /health

requires:
  services: []
  infrastructure: {}

env:
  APP_ENV: development
` + "```" + `

### SQS Queues
` + "```yaml" + `
requires:
  infrastructure:
    sqs:
      queues:
        - name: order-events
          dlq: true              # Creates order-events-dlq automatically
        - name: notifications
          dlq: false
` + "```" + `

### SNS Topics with Subscriptions
` + "```yaml" + `
requires:
  infrastructure:
    sns:
      topics:
        - name: order-events
          subscriptions:
            - protocol: sqs
              endpoint: "${sqs.order-notifications.arn}"

            # With filter policy (MessageBody filtering)
            - protocol: sqs
              endpoint: "${sqs.high-priority-orders.arn}"
              attributes:
                FilterPolicyScope: MessageBody
                FilterPolicy: |
                  {
                    "priority": ["high"],
                    "status": ["pending", "processing"]
                  }

            # With filter policy (MessageAttributes filtering)
            - protocol: sqs
              endpoint: "${sqs.region-specific.arn}"
              attributes:
                FilterPolicyScope: MessageAttributes
                FilterPolicy: |
                  {
                    "region": ["us-east-1", "us-west-2"]
                  }
` + "```" + `

### Other Infrastructure
` + "```yaml" + `
requires:
  infrastructure:
    postgres:
      database: myapp_db
      migrations: ./migrations

    mongodb:
      database: myapp_mongo

    redis: true

    s3:
      buckets:
        - name: uploads
        - name: exports
` + "```" + `

### Template Variables Reference

Use ` + "`${variable}`" + ` syntax in ` + "`env`" + `. All values are resolved at startup.

**PostgreSQL:**
| Variable | Example Value |
|----------|---------------|
| ` + "`${postgres.host}`" + ` | ` + "`postgres`" + ` |
| ` + "`${postgres.port}`" + ` | ` + "`5432`" + ` |
| ` + "`${self.postgres.database}`" + ` | Your configured database name |

**MongoDB:**
| Variable | Example Value |
|----------|---------------|
| ` + "`${mongodb.host}`" + ` | ` + "`mongodb`" + ` |
| ` + "`${mongodb.port}`" + ` | ` + "`27017`" + ` |
| ` + "`${self.mongodb.database}`" + ` | Your configured database name |

**Redis:**
| Variable | Example Value |
|----------|---------------|
| ` + "`${redis.host}`" + ` | ` + "`redis`" + ` |
| ` + "`${redis.port}`" + ` | ` + "`6379`" + ` |

**LocalStack (AWS):**
| Variable | Example Value |
|----------|---------------|
| ` + "`${localstack.endpoint}`" + ` | ` + "`http://localstack:4566`" + ` |
| ` + "`${localstack.host}`" + ` | ` + "`localstack`" + ` |
| ` + "`${localstack.port}`" + ` | ` + "`4566`" + ` |
| ` + "`${localstack.region}`" + ` | ` + "`us-east-1`" + ` |
| ` + "`${localstack.account_id}`" + ` | ` + "`000000000000`" + ` |
| ` + "`${localstack.access_key_id}`" + ` | ` + "`test`" + ` |
| ` + "`${localstack.secret_access_key}`" + ` | ` + "`test`" + ` |

**SQS Queues** (replace ` + "`<queue>`" + ` with your queue name):
| Variable | Example Value |
|----------|---------------|
| ` + "`${sqs.<queue>.url}`" + ` | ` + "`http://localstack:4566/000000000000/order-events`" + ` |
| ` + "`${sqs.<queue>.arn}`" + ` | ` + "`arn:aws:sqs:us-east-1:000000000000:order-events`" + ` |
| ` + "`${sqs.<queue>.dlq}`" + ` | ` + "`http://localstack:4566/000000000000/order-events-dlq`" + ` |

**SNS Topics** (replace ` + "`<topic>`" + ` with your topic name):
| Variable | Example Value |
|----------|---------------|
| ` + "`${sns.<topic>.arn}`" + ` | ` + "`arn:aws:sns:us-east-1:000000000000:order-events`" + ` |

**S3 Buckets** (replace ` + "`<bucket>`" + ` with your bucket name):
| Variable | Example Value |
|----------|---------------|
| ` + "`${s3.<bucket>.url}`" + ` | ` + "`http://localstack:4566/uploads`" + ` |

**Service Dependencies** (replace ` + "`<service>`" + ` with service name):
| Variable | Example Value |
|----------|---------------|
| ` + "`${<service>.host}`" + ` | ` + "`user-service`" + ` (container name) |
| ` + "`${<service>.port}`" + ` | ` + "`8080`" + ` (container port) |

### Environment Variables Example
` + "```yaml" + `
# All values support ${placeholder} syntax
env:
  LOG_LEVEL: debug
  APP_ENV: development

  # Database
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  MONGO_URL: "mongodb://${mongodb.host}:${mongodb.port}/${self.mongodb.database}"
  REDIS_URL: "redis://${redis.host}:${redis.port}"

  # AWS/LocalStack
  AWS_ENDPOINT: "${localstack.endpoint}"
  AWS_REGION: "${localstack.region}"
  AWS_ACCESS_KEY_ID: "${localstack.access_key_id}"
  AWS_SECRET_ACCESS_KEY: "${localstack.secret_access_key}"

  # Queues & Topics
  ORDER_QUEUE_URL: "${sqs.order-events.url}"
  ORDER_TOPIC_ARN: "${sns.order-events.arn}"

  # Service dependencies (inter-container communication)
  USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"
  AUTH_SERVICE_URL: "http://${auth-service.host}:${auth-service.port}"

secrets:
  OPENAI_API_KEY:
    description: "OpenAI API key for embeddings"
    required: true
  STRIPE_SECRET_KEY:
    description: "Stripe API key"
    required: true
` + "```" + `

---

## Workflows

### Starting a Service for Development
` + "```bash" + `
# 1. Clone service repositories
grund clone

# 2. Check what secrets are needed
grund secrets list my-service

# 3. Set up missing secrets (if any)
grund secrets init my-service
# Edit ~/.grund/secrets.env with actual values

# 4. Start the service
grund up my-service

# 5. Verify it's running
grund status
` + "```" + `

### Adding a New Dependency to Your Service

1. Edit ` + "`grund.yaml`" + ` to add infrastructure:
` + "```yaml" + `
requires:
  infrastructure:
    redis: true  # Add this
` + "```" + `

Or use the CLI:
` + "```bash" + `
grund service add redis
` + "```" + `

2. Add environment variable:
` + "```yaml" + `
env:
  REDIS_URL: "redis://${redis.host}:${redis.port}"
` + "```" + `

3. Restart to pick up changes:
` + "```bash" + `
grund down && grund up my-service
` + "```" + `

### Debugging a Service That Won't Start
` + "```bash" + `
# 1. Check status for errors
grund status

# 2. Check logs
grund logs my-service

# 3. Verify secrets are configured
grund secrets list my-service

# 4. Check resolved config
grund config show my-service

# 5. Nuclear option - reset everything
grund reset -v
grund up my-service
` + "```" + `

### Working with Multiple Services
` + "```bash" + `
# Start specific services
grund up user-service order-service

# Start one service without its dependencies
grund up order-service --no-deps

# View logs from all services
grund logs -f

# Restart just one service after code change
grund restart order-service
` + "```" + `

### Setting Up a New Service with Grund
` + "```bash" + `
# 1. Navigate to your service
cd ~/projects/my-new-service

# 2. Initialize interactively
grund service init

# 3. Register in ~/.grund/services.yaml
# Add:
#   my-new-service:
#     repo: git@github.com:company/my-new-service.git
#     path: ~/projects/my-new-service

# 4. Start it
grund up my-new-service
` + "```" + `

### Onboarding a New Team Member
` + "```bash" + `
# 1. Run interactive setup
grund init

# 2. Clone all service repos at once
grund clone

# 3. Start working
grund up my-service
` + "```" + `

---

## Troubleshooting

### "services file not found"
` + "```bash" + `
# Check if services.yaml exists
cat ~/.grund/services.yaml

# If not, create it:
mkdir -p ~/.grund
cat > ~/.grund/services.yaml << 'EOF'
version: "1"
services:
  my-service:
    path: ~/projects/my-service
EOF
` + "```" + `

### "missing required secrets"
` + "```bash" + `
# See what's missing
grund secrets list my-service

# Generate template
grund secrets init my-service

# Edit and add values
vim ~/.grund/secrets.env
` + "```" + `

### Service stuck in "unhealthy"
` + "```bash" + `
# Check logs for errors
grund logs my-service

# Verify health endpoint works inside container
docker exec grund-my-service curl -s http://localhost:8080/health

# Common fixes:
# - Health endpoint returns non-200
# - Service crashes on startup
# - Database connection failing
` + "```" + `

### Port already in use
` + "```bash" + `
# Check what's using the port
lsof -i :8080

# Grund auto-assigns ports on conflict, check actual port:
grund status

# Or reset everything
grund reset
` + "```" + `

### LocalStack resources not created
` + "```bash" + `
# Ensure LocalStack is healthy first
grund status  # Should show localstack as running

# Reset and retry
grund reset -v
grund up my-service
` + "```" + `

### Database connection refused
` + "```bash" + `
# Check postgres is running
grund status

# Verify DATABASE_URL is correct
grund config show my-service | grep DATABASE

# Common issue: using localhost instead of container name
# Wrong: postgres://localhost:5432
# Right: postgres://postgres:5432 (in env use ${postgres.host})
` + "```" + `

### Changes to grund.yaml not taking effect
` + "```bash" + `
# Must restart to pick up config changes
grund down
grund up my-service

# For infrastructure changes, may need full reset
grund reset -v
grund up my-service
` + "```" + `

---

## Skill Feedback

If while using this skill you notice any of the following:
- Missing instructions or edge cases
- Outdated information or better approaches
- Unclear or ambiguous guidance
- Common errors that should be documented

Please include a brief "Skill Improvement Suggestion" at the end of your response so the user can consider updating this skill.
`
