# Adding New Infrastructure to Grund

This guide explains how to add support for a new infrastructure type in Grund, using **ScyllaDB** as an example.

## Overview

Adding new infrastructure requires changes across multiple layers following Grund's DDD (Domain-Driven Design) architecture:

```
┌─────────────────────────────────────────────────────────────────┐
│                        CLI Layer                                 │
│   internal/cli/service/add/    - "grund service add" commands   │
│   internal/cli/service/init.go - Interactive init wizard        │
├─────────────────────────────────────────────────────────────────┤
│                    Application Layer                             │
│   internal/application/ports/  - Interface definitions          │
│   internal/application/queries/- Config query handler           │
├─────────────────────────────────────────────────────────────────┤
│                      Domain Layer                                │
│   internal/domain/infrastructure/ - Domain models & aggregation │
│   internal/domain/service/        - Service domain model        │
├─────────────────────────────────────────────────────────────────┤
│                   Infrastructure Layer                           │
│   internal/config/schema.go       - YAML schema definitions     │
│   internal/infrastructure/config/ - Repository & DTO mapping    │
│   internal/infrastructure/docker/ - Provisioner implementation  │
│   internal/infrastructure/generator/ - Compose & env resolver   │
└─────────────────────────────────────────────────────────────────┘
```

## Files to Modify (Checklist)

### 1. Domain Layer (Core Business Logic)

#### `internal/domain/infrastructure/infrastructure.go`

Add the new infrastructure type constant and config struct:

```go
// Add new constant
const (
    InfrastructureTypePostgres   InfrastructureType = "postgres"
    InfrastructureTypeMongoDB    InfrastructureType = "mongodb"
    InfrastructureTypeRedis      InfrastructureType = "redis"
    InfrastructureTypeLocalStack InfrastructureType = "localstack"
    InfrastructureTypeScyllaDB   InfrastructureType = "scylladb"  // NEW
)

// Add to InfrastructureRequirements struct
type InfrastructureRequirements struct {
    Postgres   *PostgresConfig
    MongoDB    *MongoDBConfig
    Redis      *RedisConfig
    ScyllaDB   *ScyllaDBConfig  // NEW
    SQS        *SQSConfig
    SNS        *SNSConfig
    S3         *S3Config
}

// Add new config struct
type ScyllaDBConfig struct {
    Keyspace   string
    Datacenter string
    Seeds      string  // Optional: seed data path
}

// Update Has() method
func (r *InfrastructureRequirements) Has(infraType string) bool {
    switch InfrastructureType(infraType) {
    // ... existing cases ...
    case InfrastructureTypeScyllaDB:
        return r.ScyllaDB != nil
    default:
        return false
    }
}

// Update Aggregate() function to handle ScyllaDB
func Aggregate(requirements ...InfrastructureRequirements) InfrastructureRequirements {
    // ... existing code ...

    // Add ScyllaDB aggregation (single-instance like Postgres)
    if req.ScyllaDB != nil && aggregated.ScyllaDB == nil {
        aggregated.ScyllaDB = req.ScyllaDB
    }

    // ... rest of function ...
}
```

### 2. Config Schema Layer (YAML Parsing)

#### `internal/config/schema.go`

Add the config struct for YAML parsing:

```go
type InfrastructureConfig struct {
    Postgres   *PostgresConfig   `yaml:"postgres,omitempty"`
    MongoDB    *MongoDBConfig    `yaml:"mongodb,omitempty"`
    Redis      interface{}       `yaml:"redis,omitempty"`
    ScyllaDB   *ScyllaDBConfig   `yaml:"scylladb,omitempty"`  // NEW
    SQS        *SQSConfig        `yaml:"sqs,omitempty"`
    SNS        *SNSConfig        `yaml:"sns,omitempty"`
    S3         *S3Config         `yaml:"s3,omitempty"`
}

// NEW: Add ScyllaDB config struct
type ScyllaDBConfig struct {
    Keyspace   string `yaml:"keyspace"`
    Datacenter string `yaml:"datacenter,omitempty"`
    Seeds      string `yaml:"seeds,omitempty"`
}
```

### 3. Infrastructure Config Layer (DTO Mapping)

#### `internal/infrastructure/config/service_repository.go`

Add DTO struct and mapping functions:

```go
// Add DTO struct
type ScyllaDBConfigDTO struct {
    Keyspace   string `yaml:"keyspace"`
    Datacenter string `yaml:"datacenter,omitempty"`
    Seeds      string `yaml:"seeds,omitempty"`
}

// Add to InfrastructureConfigDTO
type InfrastructureConfigDTO struct {
    // ... existing fields ...
    ScyllaDB   *ScyllaDBConfigDTO   `yaml:"scylladb,omitempty"`  // NEW
}

// Update toInfrastructureRequirements()
func (r *ServiceRepositoryImpl) toInfrastructureRequirements(dto InfrastructureConfigDTO) infrastructure.InfrastructureRequirements {
    var req infrastructure.InfrastructureRequirements

    // ... existing code ...

    // NEW: Handle ScyllaDB
    if dto.ScyllaDB != nil {
        req.ScyllaDB = &infrastructure.ScyllaDBConfig{
            Keyspace:   dto.ScyllaDB.Keyspace,
            Datacenter: dto.ScyllaDB.Datacenter,
            Seeds:      dto.ScyllaDB.Seeds,
        }
    }

    return req
}

// Update toConfigDTO() for reverse mapping
func (r *ServiceRepositoryImpl) toConfigDTO(svc *service.Service) ServiceConfigDTO {
    // ... existing code ...

    // NEW: Handle ScyllaDB
    if svc.Dependencies.Infrastructure.ScyllaDB != nil {
        infraDTO.ScyllaDB = &ScyllaDBConfigDTO{
            Keyspace:   svc.Dependencies.Infrastructure.ScyllaDB.Keyspace,
            Datacenter: svc.Dependencies.Infrastructure.ScyllaDB.Datacenter,
            Seeds:      svc.Dependencies.Infrastructure.ScyllaDB.Seeds,
        }
    }

    return dto
}
```

### 4. Application Ports Layer (Interfaces)

#### `internal/application/ports/orchestrator.go`

Add provisioning method to the interface:

```go
type InfrastructureProvisioner interface {
    ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error
    ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error
    ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error
    ProvisionScyllaDB(ctx context.Context, config *infrastructure.ScyllaDBConfig) error  // NEW
    ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) error
}
```

#### `internal/application/ports/compose.go`

Add context for environment variable resolution:

```go
// Add to EnvironmentContext if ScyllaDB needs special env vars
type EnvironmentContext struct {
    Infrastructure map[string]InfrastructureContext
    // ... existing fields ...
}

// ScyllaDB uses InfrastructureContext like Postgres/MongoDB
// No new struct needed - it's added to the Infrastructure map
```

### 5. Provisioner Implementation

#### `internal/infrastructure/docker/scylladb_provisioner.go` (NEW FILE)

Create a new provisioner:

```go
package docker

import (
    "context"
    "fmt"

    "github.com/Saturn-Fintech/grund/internal/application/ports"
    "github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
)

// ScyllaDBProvisioner implements infrastructure provisioning for ScyllaDB
type ScyllaDBProvisioner struct{}

// NewScyllaDBProvisioner creates a new ScyllaDB provisioner
func NewScyllaDBProvisioner() ports.InfrastructureProvisioner {
    return &ScyllaDBProvisioner{}
}

// ProvisionScyllaDB provisions ScyllaDB (creates keyspace, runs migrations)
func (s *ScyllaDBProvisioner) ProvisionScyllaDB(ctx context.Context, config *infrastructure.ScyllaDBConfig) error {
    // ScyllaDB is started via docker-compose
    // Provisioning involves:
    // 1. Wait for cluster to be ready
    // 2. Create keyspace if not exists
    // 3. Run schema migrations if specified
    // TODO: Implement keyspace creation and migration logic
    return nil
}

// ProvisionPostgres not applicable
func (s *ScyllaDBProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
    return fmt.Errorf("postgres provisioning not supported by scylladb provisioner")
}

// ProvisionMongoDB not applicable
func (s *ScyllaDBProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
    return fmt.Errorf("mongodb provisioning not supported by scylladb provisioner")
}

// ProvisionRedis not applicable
func (s *ScyllaDBProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
    return fmt.Errorf("redis provisioning not supported by scylladb provisioner")
}

// ProvisionLocalStack not applicable
func (s *ScyllaDBProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) error {
    return fmt.Errorf("localstack provisioning not supported by scylladb provisioner")
}
```

#### `internal/infrastructure/docker/infrastructure_provisioner.go`

Update composite provisioner:

```go
// Update ProvisionScyllaDB in CompositeInfrastructureProvisioner
func (c *CompositeInfrastructureProvisioner) ProvisionScyllaDB(ctx context.Context, config *infrastructure.ScyllaDBConfig) error {
    var lastErr error
    for _, p := range c.provisioners {
        if err := p.ProvisionScyllaDB(ctx, config); err == nil {
            return nil
        } else {
            lastErr = err
        }
    }
    return lastErr
}
```

### 6. Docker Compose Generator

#### `internal/infrastructure/generator/compose_generator.go`

Add ScyllaDB to compose file generation:

```go
func (g *ComposeGeneratorImpl) addInfrastructureServices(compose *ComposeFile, infra infrastructure.InfrastructureRequirements) {
    // ... existing services ...

    // NEW: Add ScyllaDB
    if infra.ScyllaDB != nil {
        compose.Services["scylladb"] = ComposeService{
            Image:         "scylladb/scylla:5.2",
            ContainerName: "grund-scylladb",
            Ports:         []string{"9042:9042", "9160:9160"},
            Command: []string{
                "--smp", "1",
                "--memory", "512M",
                "--overprovisioned", "1",
            },
            Environment: map[string]string{
                "SCYLLA_CLUSTER_NAME": "grund-cluster",
            },
            Volumes:  []string{"scylladb-data:/var/lib/scylla"},
            Networks: []string{"grund-network"},
            Healthcheck: &ComposeHealth{
                Test:        []string{"CMD-SHELL", "cqlsh -e 'describe cluster'"},
                Interval:    "10s",
                Timeout:     "5s",
                Retries:     10,
                StartPeriod: "30s",
            },
        }
        compose.Volumes["scylladb-data"] = ComposeVolume{}
    }
}

// Update buildEnvironmentContext() to add ScyllaDB context
func (g *ComposeGeneratorImpl) buildEnvironmentContext(services []*service.Service, infra infrastructure.InfrastructureRequirements) ports.EnvironmentContext {
    ctx := ports.NewDefaultEnvironmentContext()

    // ... existing infrastructure contexts ...

    // NEW: Add ScyllaDB context
    if infra.ScyllaDB != nil {
        ctx.Infrastructure["scylladb"] = ports.InfrastructureContext{
            Host:     "scylladb",
            Port:     9042,
            Database: infra.ScyllaDB.Keyspace,
        }
    }

    return ctx
}

// Update buildDependsOn() for service dependencies
func (g *ComposeGeneratorImpl) buildDependsOn(svc *service.Service) map[string]DependsOnCondition {
    dependsOn := make(map[string]DependsOnCondition)

    // ... existing infrastructure dependencies ...

    // NEW: Add ScyllaDB dependency
    if svc.RequiresInfrastructure("scylladb") {
        dependsOn["scylladb"] = DependsOnCondition{Condition: "service_healthy"}
    }

    return dependsOn
}
```

### 7. Environment Variable Resolver

#### `internal/infrastructure/generator/env_resolver.go`

Add ScyllaDB placeholder resolution:

```go
func (r *EnvironmentResolverImpl) resolvePlaceholder(path string, context ports.EnvironmentContext) (string, error) {
    parts := strings.Split(path, ".")
    prefix := parts[0]

    switch prefix {
    // ... existing cases ...
    case "scylladb":  // NEW
        return r.resolveInfrastructure("scylladb", parts[1:], context)
    default:
        return r.resolveService(prefix, parts[1:], context)
    }
}
```

### 8. Docker Orchestrator

#### `internal/infrastructure/docker/orchestrator.go`

Add ScyllaDB to infrastructure services list:

```go
// Update the list of infrastructure services
var infrastructureServices = []string{"postgres", "mongodb", "redis", "scylladb", "localstack"}
```

### 9. CLI - Add Command

#### `internal/cli/add.go`

Add ScyllaDB support to the `grund add` command:

```go
var (
    addDatabase   string
    addKeyspace   string  // NEW
    addDatacenter string  // NEW
    // ... existing flags ...
)

var addCmd = &cobra.Command{
    Long: `Add infrastructure or service dependencies to an existing grund.yaml file.

Resources:
  postgres    Add PostgreSQL database
  mongodb     Add MongoDB database
  redis       Add Redis cache
  scylladb    Add ScyllaDB cluster  // NEW
  sqs         Add SQS queue
  // ...
`,
}

func init() {
    // ... existing flags ...
    addCmd.Flags().StringVar(&addKeyspace, "keyspace", "", "Keyspace name (for scylladb)")
    addCmd.Flags().StringVar(&addDatacenter, "datacenter", "datacenter1", "Datacenter name (for scylladb)")
}

func runAdd(cmd *cobra.Command, args []string) error {
    resource := strings.ToLower(args[0])

    switch resource {
    // ... existing cases ...
    case "scylladb":  // NEW
        return addScyllaDB(config, infrastructure, configPath)
    }
}

// NEW: Add ScyllaDB function
func addScyllaDB(config map[string]any, infra map[string]any, configPath string) error {
    if _, exists := infra["scylladb"]; exists {
        return fmt.Errorf("scylladb already configured in grund.yaml")
    }

    keyspace := addKeyspace
    if keyspace == "" {
        if svc, ok := config["service"].(map[string]any); ok {
            if name, ok := svc["name"].(string); ok {
                keyspace = name + "_keyspace"
            }
        }
        if keyspace == "" {
            keyspace = "app_keyspace"
        }
    }

    scylla := map[string]any{"keyspace": keyspace}
    if addDatacenter != "" && addDatacenter != "datacenter1" {
        scylla["datacenter"] = addDatacenter
    }
    infra["scylladb"] = scylla

    // Add env for ScyllaDB connection
    addEnvRef(config, "SCYLLADB_HOSTS", "${scylladb.host}:${scylladb.port}")
    addEnvRef(config, "SCYLLADB_KEYSPACE", "${self.scylladb.keyspace}")

    if err := writeConfig(config, configPath); err != nil {
        return err
    }

    ui.Successf("Added ScyllaDB (keyspace: %s)", keyspace)
    return nil
}
```

### 10. CLI - Init Command

#### `internal/cli/init.go`

Add ScyllaDB to interactive wizard:

```go
type initConfig struct {
    // ... existing fields ...
    ScyllaDB    *scyllaInit  // NEW
}

type scyllaInit struct {
    Keyspace   string
    Datacenter string
}

func runInit(cmd *cobra.Command, args []string) error {
    // ... existing code ...

    // Infrastructure section
    fmt.Println("\n--- Infrastructure Requirements ---")

    // ... existing prompts ...

    // NEW: ScyllaDB prompt
    if promptYesNo(reader, "Need ScyllaDB?", false) {
        keyspace := prompt(reader, "  Keyspace name", cfg.Name+"_keyspace")
        datacenter := prompt(reader, "  Datacenter", "datacenter1")
        cfg.ScyllaDB = &scyllaInit{Keyspace: keyspace, Datacenter: datacenter}
    }
}

// Update generateGrundYAML() to include ScyllaDB
func generateGrundYAML(cfg *initConfig) string {
    // ... existing code ...

    // NEW: Add ScyllaDB to infrastructure
    if cfg.ScyllaDB != nil {
        scylla := map[string]any{
            "keyspace": cfg.ScyllaDB.Keyspace,
        }
        if cfg.ScyllaDB.Datacenter != "" && cfg.ScyllaDB.Datacenter != "datacenter1" {
            scylla["datacenter"] = cfg.ScyllaDB.Datacenter
        }
        infrastructure["scylladb"] = scylla
    }

    // ... env section ...
    if cfg.ScyllaDB != nil {
        env["SCYLLADB_HOSTS"] = "${scylladb.host}:${scylladb.port}"
        env["SCYLLADB_KEYSPACE"] = "${self.scylladb.keyspace}"
    }
}
```

### 11. Application Layer - Up Command Handler

#### `internal/application/commands/up_command.go`

Add ScyllaDB provisioning:

```go
func (h *UpCommandHandler) provisionInfrastructure(ctx context.Context, req infrastructure.InfrastructureRequirements) error {
    // ... existing provisioning ...

    // NEW: Provision ScyllaDB
    if req.ScyllaDB != nil {
        if err := h.provisioner.ProvisionScyllaDB(ctx, req.ScyllaDB); err != nil {
            return fmt.Errorf("scylladb: %w", err)
        }
    }

    return nil
}
```

### 12. Application Layer - Config Query Handler

#### `internal/application/queries/config_query.go`

Add ScyllaDB to environment context:

```go
func (h *ConfigQueryHandler) buildEnvironmentContext(svc *service.Service) ports.EnvironmentContext {
    ctx := ports.NewDefaultEnvironmentContext()

    // ... existing infrastructure ...

    // NEW: Add ScyllaDB if required
    if svc.Dependencies.Infrastructure.ScyllaDB != nil {
        ctx.Infrastructure["scylladb"] = ports.InfrastructureContext{
            Host:     "scylladb",
            Port:     9042,
            Database: svc.Dependencies.Infrastructure.ScyllaDB.Keyspace,
        }
        ctx.Self.Config["scylladb.keyspace"] = svc.Dependencies.Infrastructure.ScyllaDB.Keyspace
    }

    return ctx
}
```

### 13. Wiring (Dependency Injection)

#### `internal/application/wiring/wiring.go`

Register the new provisioner:

```go
func NewContainerWithConfig(orchestrationRoot, servicesPath string, configResolver *appconfig.ConfigResolver) (*Container, error) {
    // ... existing code ...

    // Initialize provisioners
    postgresProvisioner := docker.NewPostgresProvisioner()
    mongodbProvisioner := docker.NewMongoDBProvisioner()
    redisProvisioner := docker.NewRedisProvisioner()
    scylladbProvisioner := docker.NewScyllaDBProvisioner()  // NEW
    localstackProvisioner := aws.NewLocalStackProvisioner(localstackEndpoint)

    provisioner := docker.NewCompositeInfrastructureProvisioner(
        postgresProvisioner,
        mongodbProvisioner,
        redisProvisioner,
        scylladbProvisioner,  // NEW
        localstackProvisioner,
    )

    // ... rest of function ...
}
```

### 14. Update README

#### `README.md`

Add ScyllaDB documentation:

```markdown
#### ScyllaDB
```bash
grund service add scylladb myapp_keyspace
```

### Environment Variable Interpolation

| Placeholder | Description |
|-------------|-------------|
| `${scylladb.host}` | ScyllaDB hostname |
| `${scylladb.port}` | ScyllaDB CQL port (9042) |
| `${self.scylladb.keyspace}` | This service's keyspace name |
```

## Testing the Implementation

### 1. Unit Tests

Add tests for the new infrastructure type:

```go
// internal/domain/infrastructure/infrastructure_test.go
func TestInfrastructureRequirements_Has_ScyllaDB(t *testing.T) {
    req := InfrastructureRequirements{
        ScyllaDB: &ScyllaDBConfig{Keyspace: "test"},
    }

    if !req.Has("scylladb") {
        t.Error("Expected Has('scylladb') to return true")
    }
}

func TestAggregate_ScyllaDB(t *testing.T) {
    req1 := InfrastructureRequirements{
        ScyllaDB: &ScyllaDBConfig{Keyspace: "keyspace1"},
    }
    req2 := InfrastructureRequirements{
        ScyllaDB: &ScyllaDBConfig{Keyspace: "keyspace2"},
    }

    result := Aggregate(req1, req2)

    // First config should win (single-instance behavior)
    if result.ScyllaDB.Keyspace != "keyspace1" {
        t.Errorf("Expected keyspace1, got %s", result.ScyllaDB.Keyspace)
    }
}
```

### 2. Integration Test

Create a test service with ScyllaDB:

```yaml
# test/fixtures/services/scylladb-service/grund.yaml
version: "1"

service:
  name: scylladb-service
  type: go
  port: 8080

requires:
  infrastructure:
    scylladb:
      keyspace: test_keyspace

env:
  SCYLLADB_HOSTS: "${scylladb.host}:${scylladb.port}"
  SCYLLADB_KEYSPACE: "${self.scylladb.keyspace}"
```

### 3. Manual Testing

```bash
# Initialize a new service with ScyllaDB
cd my-service
grund service init
# Select ScyllaDB when prompted

# Or add to existing service
grund service add scylladb myapp_ks

# Start the service
grund up my-service

# Check status
grund status

# View config
grund config show my-service
```

## Summary Checklist

| Layer | File | Changes |
|-------|------|---------|
| Domain | `internal/domain/infrastructure/infrastructure.go` | Add type, config, Has(), Aggregate() |
| Schema | `internal/config/schema.go` | Add YAML config struct |
| Config | `internal/infrastructure/config/service_repository.go` | Add DTO and mapping |
| Ports | `internal/application/ports/orchestrator.go` | Add interface method |
| Provisioner | `internal/infrastructure/docker/scylladb_provisioner.go` | New file |
| Provisioner | `internal/infrastructure/docker/infrastructure_provisioner.go` | Add composite method |
| Generator | `internal/infrastructure/generator/compose_generator.go` | Add compose service |
| Resolver | `internal/infrastructure/generator/env_resolver.go` | Add placeholder resolution |
| Orchestrator | `internal/infrastructure/docker/orchestrator.go` | Add to infra services list |
| CLI | `internal/cli/add.go` | Add command support |
| CLI | `internal/cli/init.go` | Add wizard support |
| Command | `internal/application/commands/up_command.go` | Add provisioning call |
| Query | `internal/application/queries/config_query.go` | Add to env context |
| Wiring | `internal/application/wiring/wiring.go` | Register provisioner |
| Docs | `README.md` | Document the new infrastructure |

## Architecture Notes

1. **Single-Instance vs Multi-Instance**: ScyllaDB follows the single-instance pattern (like Postgres) - all services share one cluster. Use the Aggregate() function to merge requirements.

2. **Provisioning**: The provisioner runs AFTER the container is healthy. Use it to create keyspaces, run migrations, or seed data.

3. **Health Checks**: Choose an appropriate health check for the Docker container. For ScyllaDB, checking CQL connectivity is recommended.

4. **Environment Variables**: Follow the `${infra.property}` pattern for consistency. Use `${self.infra.property}` for service-specific config like keyspace names.

---

## Adding a New Tunnel Provider

Tunnel providers (like cloudflared, ngrok) expose local infrastructure to the internet. This is useful for webhooks, presigned URLs, or external callbacks.

### Files to Modify

#### 1. Domain Layer - Tunnel Types

**`internal/domain/infrastructure/tunnel.go`**

The domain already supports multiple providers via the `Provider` field:

```go
type TunnelRequirement struct {
    Provider string                      // "cloudflared" or "ngrok"
    Targets  []TunnelTargetRequirement
}
```

No changes needed unless adding provider-specific config.

#### 2. Tunnel Manager Interface

**`internal/application/ports/tunnel.go`**

The interface is provider-agnostic:

```go
type TunnelManager interface {
    Start(ctx context.Context, req *infrastructure.TunnelRequirement) (map[string]TunnelContext, error)
    Stop(ctx context.Context) error
}
```

#### 3. Implement the New Provider

**`internal/infrastructure/tunnel/<provider>_tunnel.go`** (NEW FILE)

Example for ngrok:

```go
package tunnel

import (
    "context"
    "fmt"
    "os/exec"
    "regexp"

    "github.com/Saturn-Fintech/grund/internal/application/ports"
    "github.com/Saturn-Fintech/grund/internal/domain/infrastructure"
)

type NgrokTunnelManager struct {
    cmd *exec.Cmd
}

func NewNgrokTunnelManager() ports.TunnelManager {
    return &NgrokTunnelManager{}
}

func (m *NgrokTunnelManager) Start(ctx context.Context, req *infrastructure.TunnelRequirement) (map[string]ports.TunnelContext, error) {
    tunnels := make(map[string]ports.TunnelContext)

    for _, target := range req.Targets {
        localAddr := fmt.Sprintf("%s:%s", target.Host, target.Port)

        // Start ngrok tunnel
        cmd := exec.CommandContext(ctx, "ngrok", "http", localAddr, "--log", "stdout")
        // ... capture output and parse public URL

        // Parse URL from ngrok output (format varies)
        // Example: "url=https://abc123.ngrok.io"
        publicURL := parseNgrokURL(output)

        tunnels[target.Name] = ports.TunnelContext{
            Name:      target.Name,
            LocalAddr: localAddr,
            PublicURL: publicURL,
        }
    }

    return tunnels, nil
}

func (m *NgrokTunnelManager) Stop(ctx context.Context) error {
    if m.cmd != nil && m.cmd.Process != nil {
        return m.cmd.Process.Kill()
    }
    return nil
}
```

#### 4. Provider Factory

**`internal/infrastructure/tunnel/factory.go`**

Create a factory to select the right provider:

```go
package tunnel

import (
    "fmt"

    "github.com/Saturn-Fintech/grund/internal/application/ports"
)

func NewTunnelManager(provider string) (ports.TunnelManager, error) {
    switch provider {
    case "cloudflared":
        return NewCloudflaredTunnelManager(), nil
    case "ngrok":
        return NewNgrokTunnelManager(), nil
    default:
        return nil, fmt.Errorf("unsupported tunnel provider: %s", provider)
    }
}
```

#### 5. Update Wiring

**`internal/application/wiring/wiring.go`**

Use the factory or hardcode default:

```go
// Create tunnel manager based on provider in config
tunnelManager, _ := tunnel.NewTunnelManager("cloudflared")
```

### Provider-Specific Considerations

| Provider | Auth Required | Multi-Tunnel | Output Format |
|----------|---------------|--------------|---------------|
| cloudflared | No (quick tunnels) | Sequential | `INF +url=https://...` |
| ngrok | Yes (free tier limited) | Via config file | JSON or `url=https://...` |
| localtunnel | No | Sequential | `your url is: https://...` |

### Testing New Providers

1. **Unit test** the URL parsing logic
2. **Integration test** with actual tunnel binary
3. **E2E test** with a service using `${tunnel.<name>.url}`

```yaml
# test/fixtures/services/tunnel-test/grund.yaml
requires:
  infrastructure:
    tunnel:
      provider: ngrok  # or your new provider
      targets:
        - name: localstack
          host: localhost
          port: "4566"

env:
  PUBLIC_ENDPOINT: "${tunnel.localstack.url}"
```

### Environment Variable Resolution

Tunnel URLs are resolved via the `${tunnel.<name>.<property>}` pattern:

| Placeholder | Description |
|-------------|-------------|
| `${tunnel.<name>.url}` | Full public URL (https://...) |
| `${tunnel.<name>.host}` | Just the hostname |

The resolver in `env_resolver.go` handles this automatically once `TunnelContext` is populated.
