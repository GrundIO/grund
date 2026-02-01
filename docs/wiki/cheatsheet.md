# Grund Command Cheatsheet

Quick reference for all Grund CLI commands.

## Starting Services

| Task | Command |
|------|---------|
| Start service with dependencies | `grund up <service>` |
| Start multiple services | `grund up service1 service2` |
| Start without dependencies | `grund up <service> --no-deps` |
| Start infrastructure only | `grund up <service> --infra-only` |
| Force rebuild containers | `grund up <service> --build` |
| Verbose output | `grund up <service> -v` |

## Stopping Services

| Task | Command |
|------|---------|
| Stop all services | `grund down` |
| Stop and keep data | `grund reset` |
| Stop and delete data | `grund reset -v` |
| Full cleanup | `grund reset -v --images` |

## Monitoring

| Task | Command |
|------|---------|
| View all service status | `grund status` |
| View all logs | `grund logs` |
| View specific service logs | `grund logs <service>` |
| Follow logs (live) | `grund logs -f` |
| Follow specific service | `grund logs <service> -f` |
| Last N lines | `grund logs --tail 50` |

## Service Management

| Task | Command |
|------|---------|
| Restart a service | `grund restart <service>` |

## Configuration

| Task | Command |
|------|---------|
| Interactive setup | `grund init` |
| Initialize global config | `grund config init` |
| Show all config | `grund config show` |
| Show service config | `grund config show <service>` |
| Initialize service | `grund service init` |
| Validate service config | `grund service validate` |

## Adding Infrastructure

| Task | Command |
|------|---------|
| Add PostgreSQL | `grund service add postgres <db_name>` |
| Add MongoDB | `grund service add mongodb <db_name>` |
| Add Redis | `grund service add redis` |
| Add SQS queue | `grund service add queue <name>` |
| Add SNS topic | `grund service add topic <name>` |
| Add S3 bucket | `grund service add bucket <name>` |
| Add tunnel | `grund service add tunnel <name>` |
| Add dependency | `grund service add dependency <service>` |

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--config` | `-c` | Path to services registry |
| `--verbose` | `-v` | Enable debug output |
| `--help` | `-h` | Show help |

## Common Workflows

### First-time setup
```bash
grund init                    # Interactive wizard
```

### Daily development
```bash
grund up my-service           # Start working
grund logs my-service -f      # Watch logs
grund restart my-service      # After code changes
grund down                    # End of day
```

### Fresh start (reset everything)
```bash
grund reset -v                # Stop and delete data
grund up my-service           # Start fresh
```

### Debug issues
```bash
grund status                  # Check what's running
grund logs <service>          # Check logs
grund -v up <service>         # Verbose startup
```

---

See also:
- [CLI Commands Reference](./cli-commands.md) - Full documentation
- [Configuration Reference](./configuration.md) - Config file details
