# Environment Variable Placeholders

Visual reference for all `${...}` placeholders available in `env`.

## How It Works

```mermaid
flowchart LR
    A["grund.yaml<br/>env"] --> B["Environment<br/>Resolver"]
    B --> C["docker-compose.yaml<br/>environment"]

    style A fill:#fff3e0
    style B fill:#e3f2fd
    style C fill:#e8f5e9
```

**Input (grund.yaml):**
```yaml
env:
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
```

**Output (resolved):**
```yaml
environment:
  DATABASE_URL: "postgres://postgres:postgres@postgres:5432/users_db"
```

---

## Infrastructure Placeholders

### PostgreSQL

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${postgres.host}` | `postgres` | Container hostname |
| `${postgres.port}` | `5432` | Default port |
| `${postgres.username}` | `postgres` | Default user |
| `${postgres.password}` | `postgres` | Default password |

**Common pattern:**
```yaml
DATABASE_URL: "postgres://${postgres.username}:${postgres.password}@${postgres.host}:${postgres.port}/${self.postgres.database}"
```

### MongoDB

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${mongodb.host}` | `mongodb` | Container hostname |
| `${mongodb.port}` | `27017` | Default port |

**Common pattern:**
```yaml
MONGODB_URI: "mongodb://${mongodb.host}:${mongodb.port}/${self.mongodb.database}"
```

### Redis

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${redis.host}` | `redis` | Container hostname |
| `${redis.port}` | `6379` | Default port |

**Common pattern:**
```yaml
REDIS_URL: "redis://${redis.host}:${redis.port}"
```

---

## LocalStack (AWS) Placeholders

### General

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${localstack.endpoint}` | `http://localstack:4566` | Full endpoint URL |
| `${localstack.host}` | `localstack` | Container hostname |
| `${localstack.port}` | `4566` | Default port |
| `${localstack.region}` | `us-east-1` | AWS region |
| `${localstack.account_id}` | `000000000000` | Fake account ID |
| `${localstack.access_key_id}` | `test` | Fake credentials |
| `${localstack.secret_access_key}` | `test` | Fake credentials |

**Common pattern:**
```yaml
AWS_ENDPOINT: "${localstack.endpoint}"
AWS_REGION: "${localstack.region}"
AWS_ACCESS_KEY_ID: "${localstack.access_key_id}"
AWS_SECRET_ACCESS_KEY: "${localstack.secret_access_key}"
```

### SQS Queues

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${sqs.<name>.url}` | Queue URL | `http://localstack:4566/000000000000/orders` |
| `${sqs.<name>.arn}` | Queue ARN | `arn:aws:sqs:us-east-1:000000000000:orders` |
| `${sqs.<name>.dlq}` | DLQ URL | `http://localstack:4566/000000000000/orders-dlq` |

**Example:**
```yaml
# For queue named "orders"
ORDERS_QUEUE_URL: "${sqs.orders.url}"
ORDERS_QUEUE_ARN: "${sqs.orders.arn}"
ORDERS_DLQ_URL: "${sqs.orders.dlq}"
```

### SNS Topics

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${sns.<name>.arn}` | Topic ARN | `arn:aws:sns:us-east-1:000000000000:notifications` |

**Example:**
```yaml
# For topic named "notifications"
NOTIFICATIONS_TOPIC_ARN: "${sns.notifications.arn}"
```

### S3 Buckets

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${s3.<name>.url}` | Bucket URL | `http://localstack:4566/uploads` |

**Example:**
```yaml
# For bucket named "uploads"
UPLOADS_BUCKET_URL: "${s3.uploads.url}"
```

---

## Service Placeholders

### Other Services

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${<service>.host}` | Service container name | `user-service` |
| `${<service>.port}` | Service port | `8080` |

**Example:**
```yaml
# For dependency on "user-service"
USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"
```

### Self Reference

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${self.host}` | Current service name | `my-service` |
| `${self.port}` | Current service port | `8080` |
| `${self.postgres.database}` | Service's database | `my_service_db` |
| `${self.mongodb.database}` | Service's MongoDB | `my_service_db` |

**Example:**
```yaml
# Reference this service's own database
DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
```

---

## Tunnel Placeholders

| Placeholder | Resolves To | Example |
|-------------|-------------|---------|
| `${tunnel.<name>.url}` | Public HTTPS URL | `https://abc-xyz.trycloudflare.com` |
| `${tunnel.<name>.host}` | Public hostname | `abc-xyz.trycloudflare.com` |

**Example:**
```yaml
# For tunnel named "localstack"
LOCALSTACK_PUBLIC_URL: "${tunnel.localstack.url}"
```

> [!NOTE]
> Tunnels are started **before** services, so tunnel URLs are always available when your service starts.

---

## Visual Reference

```mermaid
graph TB
    subgraph "Infrastructure"
        PG["${postgres.*}"]
        MG["${mongodb.*}"]
        RD["${redis.*}"]
        LS["${localstack.*}"]
    end

    subgraph "AWS Resources"
        SQS["${sqs.&lt;queue&gt;.*}"]
        SNS["${sns.&lt;topic&gt;.*}"]
        S3["${s3.&lt;bucket&gt;.*}"]
    end

    subgraph "Services"
        SVC["${&lt;service&gt;.*}"]
        SELF["${self.*}"]
    end

    subgraph "Tunnels"
        TUN["${tunnel.&lt;name&gt;.*}"]
    end

    style PG fill:#336791,color:#fff
    style MG fill:#4DB33D,color:#fff
    style RD fill:#DC382D,color:#fff
    style LS fill:#FF9900,color:#fff
    style SQS fill:#FF9900,color:#fff
    style SNS fill:#FF9900,color:#fff
    style S3 fill:#FF9900,color:#fff
    style SVC fill:#2196F3,color:#fff
    style SELF fill:#4CAF50,color:#fff
    style TUN fill:#9C27B0,color:#fff
```

---

See also:
- [Configuration Reference](./configuration.md) - Full config documentation
- [Adding New Infrastructure](./adding-new-infrastructure.md) - How to add new placeholder types
