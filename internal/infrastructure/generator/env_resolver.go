package generator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Saturn-Fintech/grund/internal/application/ports"
)

// EnvironmentResolverImpl implements EnvironmentResolver
type EnvironmentResolverImpl struct{}

// NewEnvironmentResolver creates a new environment resolver
func NewEnvironmentResolver() ports.EnvironmentResolver {
	return &EnvironmentResolverImpl{}
}

// Resolve resolves environment variable references
// Supports placeholders like:
//   - ${postgres.host}, ${postgres.port}, ${postgres.database}
//   - ${redis.host}, ${redis.port}
//   - ${mongodb.host}, ${mongodb.port}
//   - ${localstack.endpoint}, ${localstack.region}, ${localstack.account_id}
//   - ${sqs.<queue-name>.url}, ${sqs.<queue-name>.arn}, ${sqs.<queue-name>.dlq}
//   - ${sns.<topic-name>.arn}
//   - ${s3.<bucket-name>.name}, ${s3.<bucket-name>.url}
//   - ${<service-name>.host}, ${<service-name>.port}
//   - ${self.host}, ${self.port}, ${self.postgres.database}
//   - ${tunnel.<name>.url}, ${tunnel.<name>.host}
func (r *EnvironmentResolverImpl) Resolve(envRefs map[string]string, context ports.EnvironmentContext) (map[string]string, error) {
	resolved := make(map[string]string)

	for key, value := range envRefs {
		resolvedValue, err := r.resolveValue(value, context)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve %s: %w", key, err)
		}
		resolved[key] = resolvedValue
	}

	return resolved, nil
}

func (r *EnvironmentResolverImpl) resolveValue(value string, context ports.EnvironmentContext) (string, error) {
	result := value

	// Find all ${...} placeholders
	placeholderRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := placeholderRegex.FindAllStringSubmatch(value, -1)

	for _, match := range matches {
		placeholder := match[0] // Full match like ${postgres.host}
		path := match[1]        // Inner part like postgres.host

		resolved, err := r.resolvePlaceholder(path, context)
		if err != nil {
			return "", fmt.Errorf("cannot resolve %s: %w", placeholder, err)
		}

		result = strings.Replace(result, placeholder, resolved, 1)
	}

	return result, nil
}

func (r *EnvironmentResolverImpl) resolvePlaceholder(path string, context ports.EnvironmentContext) (string, error) {
	parts := strings.Split(path, ".")

	if len(parts) < 2 {
		return "", fmt.Errorf("invalid placeholder format: %s", path)
	}

	prefix := parts[0]

	switch prefix {
	case "postgres":
		return r.resolveInfrastructure("postgres", parts[1:], context)
	case "redis":
		return r.resolveInfrastructure("redis", parts[1:], context)
	case "mongodb":
		return r.resolveInfrastructure("mongodb", parts[1:], context)
	case "localstack":
		return r.resolveLocalStack(parts[1:], context)
	case "sqs":
		return r.resolveSQS(parts[1:], context)
	case "sns":
		return r.resolveSNS(parts[1:], context)
	case "s3":
		return r.resolveS3(parts[1:], context)
	case "self":
		return r.resolveSelf(parts[1:], context)
	case "tunnel":
		return r.resolveTunnel(parts[1:], context)
	default:
		// Try to resolve as a service reference
		return r.resolveService(prefix, parts[1:], context)
	}
}

func (r *EnvironmentResolverImpl) resolveInfrastructure(infraType string, parts []string, context ports.EnvironmentContext) (string, error) {
	infra, ok := context.Infrastructure[infraType]
	if !ok {
		return "", fmt.Errorf("%s not configured", infraType)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("missing property for %s", infraType)
	}

	switch parts[0] {
	case "host":
		return infra.Host, nil
	case "port":
		return fmt.Sprintf("%d", infra.Port), nil
	case "database":
		return infra.Database, nil
	case "username":
		if infra.Username == "" {
			return "postgres", nil // Default
		}
		return infra.Username, nil
	case "password":
		if infra.Password == "" {
			return "postgres", nil // Default
		}
		return infra.Password, nil
	default:
		return "", fmt.Errorf("unknown property %s for %s", parts[0], infraType)
	}
}

func (r *EnvironmentResolverImpl) resolveLocalStack(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("missing property for localstack")
	}

	switch parts[0] {
	case "endpoint":
		return context.LocalStack.Endpoint, nil
	case "host":
		// Extract host from endpoint
		endpoint := context.LocalStack.Endpoint
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		hostParts := strings.Split(endpoint, ":")
		return hostParts[0], nil
	case "port":
		return "4566", nil
	case "region":
		return context.LocalStack.Region, nil
	case "access_key_id", "accessKeyId":
		return context.LocalStack.AccessKeyID, nil
	case "secret_access_key", "secretAccessKey":
		return context.LocalStack.SecretAccessKey, nil
	case "account_id", "accountId":
		return context.LocalStack.AccountID, nil
	default:
		return "", fmt.Errorf("unknown property %s for localstack", parts[0])
	}
}

func (r *EnvironmentResolverImpl) resolveSQS(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("SQS reference must be ${sqs.<queue-name>.<property>}")
	}

	queueName := parts[0]
	property := parts[1]

	queue, ok := context.SQS[queueName]
	if !ok {
		// Generate URL based on naming convention if not explicitly set
		accountID := context.LocalStack.AccountID
		if accountID == "" {
			accountID = "000000000000"
		}
		queue = ports.QueueContext{
			Name: queueName,
			URL:  fmt.Sprintf("%s/%s/%s", context.LocalStack.Endpoint, accountID, queueName),
			ARN:  fmt.Sprintf("arn:aws:sqs:%s:%s:%s", context.LocalStack.Region, accountID, queueName),
			DLQ:  fmt.Sprintf("%s/%s/%s-dlq", context.LocalStack.Endpoint, accountID, queueName),
		}
	}

	switch property {
	case "url":
		return queue.URL, nil
	case "arn":
		return queue.ARN, nil
	case "dlq":
		return queue.DLQ, nil
	case "name":
		return queue.Name, nil
	default:
		return "", fmt.Errorf("unknown SQS property %s", property)
	}
}

func (r *EnvironmentResolverImpl) resolveSNS(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("SNS reference must be ${sns.<topic-name>.<property>}")
	}

	topicName := parts[0]
	property := parts[1]

	topic, ok := context.SNS[topicName]
	if !ok {
		// Generate ARN based on naming convention if not explicitly set
		accountID := context.LocalStack.AccountID
		if accountID == "" {
			accountID = "000000000000"
		}
		topic = ports.TopicContext{
			Name: topicName,
			ARN:  fmt.Sprintf("arn:aws:sns:%s:%s:%s", context.LocalStack.Region, accountID, topicName),
		}
	}

	switch property {
	case "arn":
		return topic.ARN, nil
	case "name":
		return topic.Name, nil
	default:
		return "", fmt.Errorf("unknown SNS property %s", property)
	}
}

func (r *EnvironmentResolverImpl) resolveS3(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("S3 reference must be ${s3.<bucket-name>.<property>}")
	}

	bucketName := parts[0]
	property := parts[1]

	bucket, ok := context.S3[bucketName]
	if !ok {
		// Generate values based on naming convention if not explicitly set
		bucket = ports.BucketContext{
			Name: bucketName,
			URL:  fmt.Sprintf("%s/%s", context.LocalStack.Endpoint, bucketName),
		}
	}

	switch property {
	case "name":
		return bucket.Name, nil
	case "url":
		return bucket.URL, nil
	default:
		return "", fmt.Errorf("unknown S3 property %s", property)
	}
}

func (r *EnvironmentResolverImpl) resolveService(serviceName string, parts []string, context ports.EnvironmentContext) (string, error) {
	svc, ok := context.Services[serviceName]
	if !ok {
		return "", fmt.Errorf("service %s not found", serviceName)
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("missing property for service %s", serviceName)
	}

	switch parts[0] {
	case "host":
		return svc.Host, nil
	case "port":
		return fmt.Sprintf("%d", svc.Port), nil
	default:
		// Check in config map
		if val, ok := svc.Config[parts[0]]; ok {
			return fmt.Sprintf("%v", val), nil
		}
		return "", fmt.Errorf("unknown property %s for service %s", parts[0], serviceName)
	}
}

func (r *EnvironmentResolverImpl) resolveSelf(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) == 0 {
		return "", fmt.Errorf("missing property for self")
	}

	switch parts[0] {
	case "host":
		return context.Self.Host, nil
	case "port":
		return fmt.Sprintf("%d", context.Self.Port), nil
	case "postgres":
		// self.postgres.database -> the database name for this service
		if len(parts) < 2 {
			return "", fmt.Errorf("missing property for self.postgres")
		}
		if val, ok := context.Self.Config["postgres."+parts[1]]; ok {
			return fmt.Sprintf("%v", val), nil
		}
		return "", fmt.Errorf("self.postgres.%s not found", parts[1])
	case "mongodb":
		if len(parts) < 2 {
			return "", fmt.Errorf("missing property for self.mongodb")
		}
		if val, ok := context.Self.Config["mongodb."+parts[1]]; ok {
			return fmt.Sprintf("%v", val), nil
		}
		return "", fmt.Errorf("self.mongodb.%s not found", parts[1])
	default:
		// Check in config map
		if val, ok := context.Self.Config[parts[0]]; ok {
			return fmt.Sprintf("%v", val), nil
		}
		return "", fmt.Errorf("unknown property %s for self", parts[0])
	}
}

func (r *EnvironmentResolverImpl) resolveTunnel(parts []string, context ports.EnvironmentContext) (string, error) {
	if len(parts) < 2 {
		return "", fmt.Errorf("tunnel reference must be ${tunnel.<name>.<property>}")
	}

	tunnelName := parts[0]
	property := parts[1]

	tunnel, ok := context.Tunnel[tunnelName]
	if !ok {
		// Tunnel not available yet (e.g., during early validation before tunnels start)
		// Return a placeholder that will be resolved later during compose generation
		return fmt.Sprintf("TUNNEL_PENDING_%s_%s", strings.ToUpper(tunnelName), strings.ToUpper(property)), nil
	}

	switch property {
	case "url":
		return tunnel.PublicURL, nil
	case "host":
		// Extract host from URL
		url := tunnel.PublicURL
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		// Remove any path
		if idx := strings.Index(url, "/"); idx != -1 {
			url = url[:idx]
		}
		return url, nil
	default:
		return "", fmt.Errorf("unknown tunnel property %s", property)
	}
}
