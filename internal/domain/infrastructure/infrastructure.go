package infrastructure

// InfrastructureType represents the type of infrastructure
type InfrastructureType string

const (
	InfrastructureTypePostgres   InfrastructureType = "postgres"
	InfrastructureTypeMongoDB    InfrastructureType = "mongodb"
	InfrastructureTypeRedis      InfrastructureType = "redis"
	InfrastructureTypeLocalStack InfrastructureType = "localstack"
)

// InfrastructureRequirements aggregates all infrastructure needs
type InfrastructureRequirements struct {
	Postgres *PostgresConfig
	MongoDB  *MongoDBConfig
	Redis    *RedisConfig
	SQS      *SQSConfig
	SNS      *SNSConfig
	S3       *S3Config
	Tunnel   *TunnelRequirement
}

// Has checks if a specific infrastructure type is required
func (r *InfrastructureRequirements) Has(infraType string) bool {
	switch InfrastructureType(infraType) {
	case InfrastructureTypePostgres:
		return r.Postgres != nil
	case InfrastructureTypeMongoDB:
		return r.MongoDB != nil
	case InfrastructureTypeRedis:
		return r.Redis != nil
	case InfrastructureTypeLocalStack:
		return r.SQS != nil || r.SNS != nil || r.S3 != nil
	default:
		return false
	}
}

// PostgresConfig represents PostgreSQL configuration
type PostgresConfig struct {
	Database   string             // Primary database (created by Docker via POSTGRES_DB)
	Migrations string             // Migrations path for primary database
	Seed       string             // Seed path for primary database
	Databases  []PostgresDatabase // Additional databases from other services
}

// PostgresDatabase represents a single PostgreSQL database
type PostgresDatabase struct {
	Name       string
	Migrations string
	Seed       string
}

// MongoDBConfig represents MongoDB configuration
type MongoDBConfig struct {
	Database string
	Seed     string
}

// RedisConfig represents Redis configuration
type RedisConfig struct {
	// Future: specific redis config if needed
}

// SQSConfig represents SQS queue configuration
type SQSConfig struct {
	Queues []QueueConfig
}

// QueueConfig represents a single SQS queue
type QueueConfig struct {
	Name string
	DLQ  bool
}

// SNSConfig represents SNS topic configuration
type SNSConfig struct {
	Topics []TopicConfig
}

// TopicConfig represents a single SNS topic
type TopicConfig struct {
	Name          string
	Subscriptions []SubscriptionConfig
}

// SubscriptionConfig represents an SNS subscription
type SubscriptionConfig struct {
	Protocol   string
	Endpoint   string            // Template: "${sqs.queue-name.arn}"
	Attributes map[string]string // AWS subscription attributes (FilterPolicy, FilterPolicyScope, etc.)
}

// S3Config represents S3 bucket configuration
type S3Config struct {
	Buckets []BucketConfig
}

// BucketConfig represents a single S3 bucket
type BucketConfig struct {
	Name string
	Seed string
	CORS bool
}

// TunnelRequirement represents tunnel infrastructure needs
type TunnelRequirement struct {
	Provider string
	Targets  []TunnelTargetRequirement
}

// TunnelTargetRequirement represents a single tunnel target
type TunnelTargetRequirement struct {
	Name string
	Host string
	Port string
}

// Aggregate aggregates infrastructure requirements from multiple services
// - Single-instance resources (Postgres, MongoDB, Redis): Creates one shared instance
// - Multi-instance resources (SQS, SNS, S3): Deduplicates by name
func Aggregate(requirements ...InfrastructureRequirements) InfrastructureRequirements {
	aggregated := InfrastructureRequirements{}

	// Track seen resources to avoid duplicates
	seenQueues := make(map[string]bool)
	seenTopics := make(map[string]bool)
	seenBuckets := make(map[string]bool)
	seenDatabases := make(map[string]bool)

	for _, req := range requirements {
		// PostgreSQL: aggregate all databases from all services
		if req.Postgres != nil {
			if aggregated.Postgres == nil {
				// First Postgres config becomes the primary
				aggregated.Postgres = &PostgresConfig{
					Database:   req.Postgres.Database,
					Migrations: req.Postgres.Migrations,
					Seed:       req.Postgres.Seed,
					Databases:  []PostgresDatabase{},
				}
				seenDatabases[req.Postgres.Database] = true
			} else if !seenDatabases[req.Postgres.Database] {
				// Additional databases get added to the list
				aggregated.Postgres.Databases = append(aggregated.Postgres.Databases, PostgresDatabase{
					Name:       req.Postgres.Database,
					Migrations: req.Postgres.Migrations,
					Seed:       req.Postgres.Seed,
				})
				seenDatabases[req.Postgres.Database] = true
			}
		}
		if req.MongoDB != nil && aggregated.MongoDB == nil {
			aggregated.MongoDB = req.MongoDB
		}
		if req.Redis != nil && aggregated.Redis == nil {
			aggregated.Redis = req.Redis
		}

		// Aggregate SQS queues (deduplicate by name)
		if req.SQS != nil {
			if aggregated.SQS == nil {
				aggregated.SQS = &SQSConfig{Queues: []QueueConfig{}}
			}
			for _, queue := range req.SQS.Queues {
				if !seenQueues[queue.Name] {
					seenQueues[queue.Name] = true
					aggregated.SQS.Queues = append(aggregated.SQS.Queues, queue)
				}
			}
		}

		// Aggregate SNS topics (deduplicate by name)
		if req.SNS != nil {
			if aggregated.SNS == nil {
				aggregated.SNS = &SNSConfig{Topics: []TopicConfig{}}
			}
			for _, topic := range req.SNS.Topics {
				if !seenTopics[topic.Name] {
					seenTopics[topic.Name] = true
					aggregated.SNS.Topics = append(aggregated.SNS.Topics, topic)
				}
			}
		}

		// Aggregate S3 buckets (deduplicate by name)
		if req.S3 != nil {
			if aggregated.S3 == nil {
				aggregated.S3 = &S3Config{Buckets: []BucketConfig{}}
			}
			for _, bucket := range req.S3.Buckets {
				if !seenBuckets[bucket.Name] {
					seenBuckets[bucket.Name] = true
					aggregated.S3.Buckets = append(aggregated.S3.Buckets, bucket)
				}
			}
		}

		// Aggregate tunnel targets (deduplicate by name)
		if req.Tunnel != nil {
			if aggregated.Tunnel == nil {
				aggregated.Tunnel = &TunnelRequirement{
					Provider: req.Tunnel.Provider,
					Targets:  make([]TunnelTargetRequirement, 0),
				}
			}
			// Add targets, avoiding duplicates by name
			existingNames := make(map[string]bool)
			for _, t := range aggregated.Tunnel.Targets {
				existingNames[t.Name] = true
			}
			for _, t := range req.Tunnel.Targets {
				if !existingNames[t.Name] {
					aggregated.Tunnel.Targets = append(aggregated.Tunnel.Targets, t)
					existingNames[t.Name] = true
				}
			}
		}
	}

	return aggregated
}
