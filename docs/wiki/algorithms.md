# Grund Algorithms

This document explains the key algorithms used in Grund.

## Table of Contents

1. [Dependency Resolution](#dependency-resolution)
2. [Circular Dependency Detection](#circular-dependency-detection)
3. [Topological Sort (Startup Order)](#topological-sort-startup-order)
4. [Infrastructure Aggregation](#infrastructure-aggregation)
5. [Environment Variable Resolution](#environment-variable-resolution)
6. [Configuration Resolution](#configuration-resolution)

---

## Dependency Resolution

**Location:** `internal/domain/dependency/graph.go`

### Overview

Grund builds a directed acyclic graph (DAG) of service dependencies to determine:
1. Which services need to be started
2. The correct order to start them
3. Whether any circular dependencies exist

### Data Structure

```go
type Graph struct {
    nodes map[service.ServiceName]*Node
}

type Node struct {
    Service      *service.Service
    Dependencies []service.ServiceName  // Services this node depends ON
    Dependents   []service.ServiceName  // Services that depend ON this node
}
```

### Algorithm: GetAllDependencies

Collects all transitive dependencies for a service using depth-first search.

```
Algorithm: GetAllDependencies(serviceName)
───────────────────────────────────────────

Input: serviceName - the starting service
Output: list of all services that serviceName depends on (directly or indirectly)

1. Initialize empty visited set
2. Initialize empty dependencies list

3. Define recursive function collectDeps(name):
   a. If name in visited, return (already processed)
   b. Add name to visited
   c. Get node for name
   d. For each dependency of node:
      - Add dependency to dependencies list
      - Recursively call collectDeps(dependency)

4. Call collectDeps(serviceName)
5. Return dependencies list
```

**Time Complexity:** O(V + E) where V = vertices (services), E = edges (dependencies)

**Example:**

```mermaid
graph LR
    subgraph "User Request"
        order["order-service"]
    end

    subgraph "Dependencies Resolved"
        order --> user["user-service"]
        order --> notif["notification-service"]
        user --> auth["auth-service"]
    end

    style order fill:#4CAF50,color:#fff
    style user fill:#2196F3,color:#fff
    style notif fill:#2196F3,color:#fff
    style auth fill:#2196F3,color:#fff
```

```
GetAllDependencies("order-service"):
  Result: [user-service, auth-service, notification-service]
```

---

## Circular Dependency Detection

**Location:** `internal/domain/dependency/graph.go`

### Overview

Detects circular dependencies using DFS with a recursion stack. **Note:** Circular dependencies are now allowed - services start in parallel and are expected to handle reconnection to dependencies that start concurrently.

### Algorithm: DetectCycle

Uses depth-first search with three states for each node:
- **Unvisited**: Not yet processed
- **In recursion stack**: Currently being processed (path from start)
- **Visited**: Fully processed

```
Algorithm: DetectCycle(startService)
────────────────────────────────────

Input: startService - where to start detection
Output: cycle path if found, nil otherwise

1. Initialize empty visited set
2. Initialize empty recursion stack set
3. Initialize empty cycle list

4. Define recursive function dfs(name, path):
   a. Mark name as visited
   b. Add name to recursion stack
   c. Append name to current path

   d. For each dependency of name:
      - If dependency not visited:
        * Recursively call dfs(dependency, path)
      - Else if dependency in recursion stack:
        * CYCLE FOUND!
        * Extract cycle from path starting at dependency
        * Return error with cycle path

   e. Remove name from recursion stack
   f. Return nil (no cycle on this path)

5. Call dfs(startService, [])
6. Return cycle if found, nil otherwise
```

**Time Complexity:** O(V + E)

**Example:**
```
Circular dependency detected:
  A → B → C → A

DetectCycle("A"):
  Returns: ["A", "B", "C", "A"]  // Cycle path for logging
  Note: This is informational only - services will start in parallel
```

---

## Topological Sort (Startup Order)

**Location:** `internal/domain/dependency/graph.go`

### Overview

Determines the correct startup order so that dependencies start before the services that need them. Uses Kahn's algorithm for topological sorting.

### Algorithm: TopologicalSort

```
Algorithm: TopologicalSort(startServices)
─────────────────────────────────────────

Input: startServices - list of services to start
Output: ordered list of services (dependencies first)

Phase 1: Collect all services needed
──────────────────────────────────────
1. Initialize allServices set
2. For each service in startServices:
   a. Get all dependencies (using GetAllDependencies)
   b. Add all dependencies to allServices
   c. Add service itself to allServices

Phase 2: Calculate in-degrees
─────────────────────────────
3. For each service in allServices:
   a. Count how many dependencies it has (within allServices)
   b. Store as inDegree[service]

Phase 3: Kahn's Algorithm
─────────────────────────
4. Initialize queue with services where inDegree = 0
   (these have no dependencies, can start first)

5. Initialize empty result list

6. While queue is not empty:
   a. Remove first item (current) from queue
   b. Add current to result
   c. For each service that depends on current:
      - Decrement its inDegree
      - If inDegree becomes 0, add to queue

7. If result.length ≠ allServices.length:
   Error: cycle detected (not all nodes processed)

8. Return result
```

**Time Complexity:** O(V + E)

**Visual Example:**

```mermaid
graph TD
    subgraph "Service Dependencies"
        order["order-service<br/>(in-degree: 2)"]
        notif["notification-service<br/>(in-degree: 1)"]
        user["user-service<br/>(in-degree: 0)"]

        order --> user
        order --> notif
        notif --> user
    end

    style user fill:#4CAF50,color:#fff
    style notif fill:#FFC107,color:#000
    style order fill:#F44336,color:#fff
```

**Kahn's Algorithm Execution:**

```mermaid
sequenceDiagram
    participant Q as Queue
    participant R as Result

    Note over Q,R: Initial: in-degrees = {user:0, notif:1, order:2}

    Q->>Q: Add user-service (in-degree=0)

    rect rgb(200, 230, 200)
        Note over Q,R: Step 1
        Q->>R: Process user-service
        Note over Q: notif: 1→0, order: 2→1
        Q->>Q: Add notification-service
    end

    rect rgb(255, 243, 200)
        Note over Q,R: Step 2
        Q->>R: Process notification-service
        Note over Q: order: 1→0
        Q->>Q: Add order-service
    end

    rect rgb(255, 200, 200)
        Note over Q,R: Step 3
        Q->>R: Process order-service
    end

    Note over R: Final: [user, notification, order]
```

**Startup Order:** `user-service` → `notification-service` → `order-service`

---

## Infrastructure Aggregation

**Location:** `internal/domain/infrastructure/infrastructure.go`

### Overview

When multiple services need infrastructure, Grund aggregates their requirements:
- **Single-instance** (Postgres, MongoDB, Redis): One shared container
- **Multi-instance** (SQS, SNS, S3): Deduplicated resources

### Algorithm: Aggregate

```
Algorithm: Aggregate(requirements...)
─────────────────────────────────────

Input: List of InfrastructureRequirements from multiple services
Output: Single aggregated InfrastructureRequirements

1. Initialize empty aggregated result
2. Initialize tracking sets: seenQueues, seenTopics, seenBuckets

3. For each requirement in requirements:

   # Single-instance resources (first config wins)
   a. If requirement has Postgres AND aggregated doesn't:
      - Copy Postgres config to aggregated
   b. If requirement has MongoDB AND aggregated doesn't:
      - Copy MongoDB config to aggregated
   c. If requirement has Redis AND aggregated doesn't:
      - Copy Redis config to aggregated

   # Multi-instance resources (deduplicate by name)
   d. If requirement has SQS:
      - For each queue:
        * If queue.name not in seenQueues:
          - Add to seenQueues
          - Add queue to aggregated.SQS

   e. If requirement has SNS:
      - For each topic:
        * If topic.name not in seenTopics:
          - Add to seenTopics
          - Add topic to aggregated.SNS

   f. If requirement has S3:
      - For each bucket:
        * If bucket.name not in seenBuckets:
          - Add to seenBuckets
          - Add bucket to aggregated.S3

4. Return aggregated
```

**Example:**
```
Service A requires:
  - Postgres (database: "service_a_db")
  - SQS queues: [orders, notifications]

Service B requires:
  - Postgres (database: "service_b_db")  ← Ignored, A's config wins
  - SQS queues: [orders, payments]       ← "orders" deduplicated
  - Redis

Aggregated result:
  - Postgres (database: "service_a_db")
  - Redis
  - SQS queues: [orders, notifications, payments]
```

---

## Environment Variable Resolution

**Location:** `internal/infrastructure/generator/env_resolver.go`

### Overview

Resolves `${placeholder}` syntax in environment variable references to actual values at compose-generation time.

### Supported Placeholders

| Pattern | Example | Resolves To |
|---------|---------|-------------|
| `${postgres.host}` | | `postgres` |
| `${postgres.port}` | | `5432` |
| `${postgres.database}` | | Database name |
| `${postgres.username}` | | `postgres` |
| `${postgres.password}` | | `postgres` |
| `${mongodb.host}` | | `mongodb` |
| `${mongodb.port}` | | `27017` |
| `${redis.host}` | | `redis` |
| `${redis.port}` | | `6379` |
| `${localstack.endpoint}` | | `http://localstack:4566` |
| `${localstack.region}` | | `us-east-1` |
| `${localstack.account_id}` | | `000000000000` |
| `${sqs.<queue>.url}` | `${sqs.orders.url}` | Queue URL |
| `${sqs.<queue>.arn}` | `${sqs.orders.arn}` | Queue ARN |
| `${sqs.<queue>.dlq}` | `${sqs.orders.dlq}` | DLQ URL |
| `${sns.<topic>.arn}` | `${sns.events.arn}` | Topic ARN |
| `${s3.<bucket>.url}` | `${s3.uploads.url}` | Bucket URL |
| `${<service>.host}` | `${user-service.host}` | Service container name |
| `${<service>.port}` | `${user-service.port}` | Service port |
| `${self.host}` | | Current service name |
| `${self.port}` | | Current service port |
| `${self.postgres.database}` | | Service's own DB name |

### Algorithm: Resolve

```
Algorithm: Resolve(envRefs, context)
────────────────────────────────────

Input:
  - envRefs: map of ENV_VAR → value with ${placeholders}
  - context: EnvironmentContext with infrastructure/service details
Output: map of ENV_VAR → resolved value

1. Initialize empty resolved map

2. For each (key, value) in envRefs:
   a. Call resolveValue(value, context)
   b. Store result in resolved[key]

3. Return resolved

───────────────────────────────────────

Algorithm: resolveValue(value, context)
───────────────────────────────────────

1. Find all ${...} patterns in value using regex
2. For each match:
   a. Extract inner path (e.g., "postgres.host")
   b. Call resolvePlaceholder(path, context)
   c. Replace placeholder with resolved value
3. Return fully resolved string

───────────────────────────────────────

Algorithm: resolvePlaceholder(path, context)
────────────────────────────────────────────

1. Split path by "." → parts
2. Get prefix (first part)

3. Switch on prefix:
   - "postgres", "mongodb", "redis":
     → Look up in context.Infrastructure[prefix]
   - "localstack":
     → Look up in context.LocalStack
   - "sqs":
     → Look up queue in context.SQS[parts[1]]
   - "sns":
     → Look up topic in context.SNS[parts[1]]
   - "s3":
     → Look up bucket in context.S3[parts[1]]
   - "self":
     → Look up in context.Self
   - default:
     → Assume it's a service name, look up in context.Services[prefix]

4. Return resolved value or error if not found
```

**Example:**
```yaml
# Input (grund.yaml env_refs)
env_refs:
  DATABASE_URL: "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}"
  ORDERS_QUEUE_URL: "${sqs.orders.url}"
  USER_SERVICE_URL: "http://${user-service.host}:${user-service.port}"

# Output (resolved for user-service with database "users_db")
environment:
  DATABASE_URL: "postgres://postgres:postgres@postgres:5432/users_db"
  ORDERS_QUEUE_URL: "http://localstack:4566/000000000000/orders"
  USER_SERVICE_URL: "http://user-service:8080"
```

---

## Configuration Resolution

**Location:** `internal/config/resolver.go`

### Overview

Finds the services.yaml file through multiple fallback locations.

### Algorithm: ResolveServicesFile

```
Algorithm: ResolveServicesFile()
────────────────────────────────

Output: (servicesPath, orchestrationRoot, error)

1. Check CLI flag (--config):
   - If provided, resolve to absolute path
   - If file exists, return (path, parent_dir)
   - If not exists, return error

2. Check environment variable (GRUND_CONFIG):
   - If set, resolve to absolute path
   - If file exists, return (path, parent_dir)
   - If not exists, return error

3. Search local directories:
   - Start from current working directory
   - For each directory (up to 5 levels up):
     * Check for: services.yaml, grund-services.yaml, grund.services.yaml
     * If found, return (path, containing_dir)
   - If not found, continue to step 4

4. Check global config default:
   - Read default_orchestration_repo from ~/.grund/config.yaml
   - Construct path: {repo}/services.yaml
   - If file exists, return (path, repo_dir)

5. If nothing found:
   - Return error with helpful message
```

**Visual Flow:**

```mermaid
flowchart TD
    A[Start] --> B{--config flag<br/>provided?}
    B -->|Yes| C[Use specified path]
    B -->|No| D{GRUND_CONFIG<br/>env set?}
    D -->|Yes| E[Use env path]
    D -->|No| F{Local file exists?<br/>services.yaml<br/>grund-services.yaml}
    F -->|Yes| G[Use local file]
    F -->|No| H{~/.grund/config.yaml<br/>has services?}
    H -->|Yes| I[Use global config]
    H -->|No| J[❌ Error:<br/>services file not found]

    C --> K[✅ Load services]
    E --> K
    G --> K
    I --> K

    style C fill:#4CAF50,color:#fff
    style E fill:#4CAF50,color:#fff
    style G fill:#4CAF50,color:#fff
    style I fill:#4CAF50,color:#fff
    style J fill:#F44336,color:#fff
    style K fill:#2196F3,color:#fff
```

---

## Algorithm Complexity Summary

| Algorithm | Time Complexity | Space Complexity |
|-----------|-----------------|------------------|
| GetAllDependencies | O(V + E) | O(V) |
| DetectCycle | O(V + E) | O(V) |
| TopologicalSort | O(V + E) | O(V) |
| Aggregate | O(n * m) | O(m) |
| Resolve (env) | O(n * p) | O(n) |
| ResolveServicesFile | O(1) | O(1) |

Where:
- V = number of services (vertices)
- E = number of dependencies (edges)
- n = number of environment variables
- m = number of infrastructure items
- p = number of placeholders per variable

---

## See Also

- [Architecture Overview](./architecture.md)
- [CLI Commands](./cli-commands.md)
- [Configuration Reference](./configuration.md)
