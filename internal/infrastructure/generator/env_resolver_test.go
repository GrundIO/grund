package generator

import (
	"testing"

	"github.com/GrundIO/grund/internal/application/ports"
)

func TestEnvironmentResolver_ResolvePostgres(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Infrastructure["postgres"] = ports.InfrastructureContext{
		Host:     "postgres",
		Port:     5432,
		Database: "mydb",
		Username: "myuser",
		Password: "mypass",
	}

	envRefs := map[string]string{
		"DATABASE_URL": "postgres://${postgres.username}:${postgres.password}@${postgres.host}:${postgres.port}/${postgres.database}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "postgres://myuser:mypass@postgres:5432/mydb"
	if resolved["DATABASE_URL"] != expected {
		t.Errorf("DATABASE_URL = %q, want %q", resolved["DATABASE_URL"], expected)
	}
}

func TestEnvironmentResolver_ResolveRedis(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Infrastructure["redis"] = ports.InfrastructureContext{
		Host: "redis",
		Port: 6379,
	}

	envRefs := map[string]string{
		"REDIS_URL": "redis://${redis.host}:${redis.port}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "redis://redis:6379"
	if resolved["REDIS_URL"] != expected {
		t.Errorf("REDIS_URL = %q, want %q", resolved["REDIS_URL"], expected)
	}
}

func TestEnvironmentResolver_ResolveSQS(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.SQS["order-queue"] = ports.QueueContext{
		Name: "order-queue",
		URL:  "http://localstack:4566/000000000000/order-queue",
		ARN:  "arn:aws:sqs:us-east-1:000000000000:order-queue",
		DLQ:  "http://localstack:4566/000000000000/order-queue-dlq",
	}

	envRefs := map[string]string{
		"ORDER_QUEUE_URL": "${sqs.order-queue.url}",
		"ORDER_QUEUE_ARN": "${sqs.order-queue.arn}",
		"ORDER_DLQ_URL":   "${sqs.order-queue.dlq}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved["ORDER_QUEUE_URL"] != "http://localstack:4566/000000000000/order-queue" {
		t.Errorf("ORDER_QUEUE_URL = %q", resolved["ORDER_QUEUE_URL"])
	}
	if resolved["ORDER_QUEUE_ARN"] != "arn:aws:sqs:us-east-1:000000000000:order-queue" {
		t.Errorf("ORDER_QUEUE_ARN = %q", resolved["ORDER_QUEUE_ARN"])
	}
	if resolved["ORDER_DLQ_URL"] != "http://localstack:4566/000000000000/order-queue-dlq" {
		t.Errorf("ORDER_DLQ_URL = %q", resolved["ORDER_DLQ_URL"])
	}
}

func TestEnvironmentResolver_ResolveSQS_AutoGenerate(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	// SQS queue not explicitly set - should auto-generate based on naming convention

	envRefs := map[string]string{
		"PAYMENT_QUEUE_URL": "${sqs.payment-queue.url}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "http://localstack:4566/000000000000/payment-queue"
	if resolved["PAYMENT_QUEUE_URL"] != expected {
		t.Errorf("PAYMENT_QUEUE_URL = %q, want %q", resolved["PAYMENT_QUEUE_URL"], expected)
	}
}

func TestEnvironmentResolver_ResolveSNS(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.SNS["order-events"] = ports.TopicContext{
		Name: "order-events",
		ARN:  "arn:aws:sns:us-east-1:000000000000:order-events",
	}

	envRefs := map[string]string{
		"ORDER_TOPIC_ARN": "${sns.order-events.arn}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "arn:aws:sns:us-east-1:000000000000:order-events"
	if resolved["ORDER_TOPIC_ARN"] != expected {
		t.Errorf("ORDER_TOPIC_ARN = %q, want %q", resolved["ORDER_TOPIC_ARN"], expected)
	}
}

func TestEnvironmentResolver_ResolveS3(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.S3["user-uploads"] = ports.BucketContext{
		Name: "user-uploads",
		URL:  "http://localstack:4566/user-uploads",
	}

	envRefs := map[string]string{
		"S3_BUCKET":     "${s3.user-uploads.name}",
		"S3_BUCKET_URL": "${s3.user-uploads.url}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved["S3_BUCKET"] != "user-uploads" {
		t.Errorf("S3_BUCKET = %q, want 'user-uploads'", resolved["S3_BUCKET"])
	}
	if resolved["S3_BUCKET_URL"] != "http://localstack:4566/user-uploads" {
		t.Errorf("S3_BUCKET_URL = %q", resolved["S3_BUCKET_URL"])
	}
}

func TestEnvironmentResolver_ResolveLocalStack(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()

	envRefs := map[string]string{
		"AWS_ENDPOINT": "${localstack.endpoint}",
		"AWS_REGION":   "${localstack.region}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved["AWS_ENDPOINT"] != "http://localstack:4566" {
		t.Errorf("AWS_ENDPOINT = %q, want 'http://localstack:4566'", resolved["AWS_ENDPOINT"])
	}
	if resolved["AWS_REGION"] != "us-east-1" {
		t.Errorf("AWS_REGION = %q, want 'us-east-1'", resolved["AWS_REGION"])
	}
}

func TestEnvironmentResolver_ResolveService(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Services["service-b"] = ports.ServiceContext{
		Host: "service-b",
		Port: 8081,
	}

	envRefs := map[string]string{
		"SERVICE_B_URL": "http://${service-b.host}:${service-b.port}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "http://service-b:8081"
	if resolved["SERVICE_B_URL"] != expected {
		t.Errorf("SERVICE_B_URL = %q, want %q", resolved["SERVICE_B_URL"], expected)
	}
}

func TestEnvironmentResolver_ResolveSelf(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Self = ports.ServiceContext{
		Host: "my-service",
		Port: 8080,
		Config: map[string]interface{}{
			"postgres.database": "my_service_db",
		},
	}

	envRefs := map[string]string{
		"SELF_HOST":     "${self.host}",
		"SELF_PORT":     "${self.port}",
		"SELF_DATABASE": "${self.postgres.database}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved["SELF_HOST"] != "my-service" {
		t.Errorf("SELF_HOST = %q, want 'my-service'", resolved["SELF_HOST"])
	}
	if resolved["SELF_PORT"] != "8080" {
		t.Errorf("SELF_PORT = %q, want '8080'", resolved["SELF_PORT"])
	}
	if resolved["SELF_DATABASE"] != "my_service_db" {
		t.Errorf("SELF_DATABASE = %q, want 'my_service_db'", resolved["SELF_DATABASE"])
	}
}

func TestEnvironmentResolver_ComplexURL(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Infrastructure["postgres"] = ports.InfrastructureContext{
		Host:     "postgres",
		Port:     5432,
		Database: "mydb",
	}
	ctx.Self = ports.ServiceContext{
		Host: "my-service",
		Port: 8080,
		Config: map[string]interface{}{
			"postgres.database": "service_a_db",
		},
	}

	envRefs := map[string]string{
		"DATABASE_URL": "postgres://postgres:postgres@${postgres.host}:${postgres.port}/${self.postgres.database}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "postgres://postgres:postgres@postgres:5432/service_a_db"
	if resolved["DATABASE_URL"] != expected {
		t.Errorf("DATABASE_URL = %q, want %q", resolved["DATABASE_URL"], expected)
	}
}

func TestEnvironmentResolver_MultiplePlaceholders(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Infrastructure["postgres"] = ports.InfrastructureContext{
		Host: "postgres",
		Port: 5432,
	}
	ctx.Infrastructure["redis"] = ports.InfrastructureContext{
		Host: "redis",
		Port: 6379,
	}
	ctx.SQS["orders"] = ports.QueueContext{
		URL: "http://localstack:4566/000000000000/orders",
	}

	envRefs := map[string]string{
		"CONFIG": "pg=${postgres.host}:${postgres.port},redis=${redis.host}:${redis.port},sqs=${sqs.orders.url}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	expected := "pg=postgres:5432,redis=redis:6379,sqs=http://localstack:4566/000000000000/orders"
	if resolved["CONFIG"] != expected {
		t.Errorf("CONFIG = %q, want %q", resolved["CONFIG"], expected)
	}
}

func TestEnvironmentResolver_UnknownPlaceholder(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()

	envRefs := map[string]string{
		"UNKNOWN": "${unknown.value}",
	}

	_, err := resolver.Resolve(envRefs, ctx)
	if err == nil {
		t.Error("Expected error for unknown placeholder, got nil")
	}
}

func TestEnvironmentResolver_NoPlaceholders(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()

	envRefs := map[string]string{
		"STATIC_VALUE": "just-a-string",
		"NUMBER":       "12345",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if resolved["STATIC_VALUE"] != "just-a-string" {
		t.Errorf("STATIC_VALUE = %q, want 'just-a-string'", resolved["STATIC_VALUE"])
	}
	if resolved["NUMBER"] != "12345" {
		t.Errorf("NUMBER = %q, want '12345'", resolved["NUMBER"])
	}
}

func TestResolveTunnelPlaceholders(t *testing.T) {
	resolver := NewEnvironmentResolver()

	ctx := ports.NewDefaultEnvironmentContext()
	ctx.Tunnel["localstack"] = ports.TunnelContext{
		Name:      "localstack",
		PublicURL: "https://abc-xyz.trycloudflare.com",
	}
	ctx.Tunnel["api"] = ports.TunnelContext{
		Name:      "api",
		PublicURL: "https://def-123.trycloudflare.com",
	}

	envRefs := map[string]string{
		"PUBLIC_S3_ENDPOINT": "${tunnel.localstack.url}",
		"PUBLIC_API_URL":     "${tunnel.api.url}",
		"PUBLIC_S3_HOST":     "${tunnel.localstack.host}",
	}

	resolved, err := resolver.Resolve(envRefs, ctx)
	if err != nil {
		t.Fatalf("failed to resolve: %v", err)
	}

	if resolved["PUBLIC_S3_ENDPOINT"] != "https://abc-xyz.trycloudflare.com" {
		t.Errorf("expected https://abc-xyz.trycloudflare.com, got %s", resolved["PUBLIC_S3_ENDPOINT"])
	}
	if resolved["PUBLIC_API_URL"] != "https://def-123.trycloudflare.com" {
		t.Errorf("expected https://def-123.trycloudflare.com, got %s", resolved["PUBLIC_API_URL"])
	}
	if resolved["PUBLIC_S3_HOST"] != "abc-xyz.trycloudflare.com" {
		t.Errorf("expected abc-xyz.trycloudflare.com, got %s", resolved["PUBLIC_S3_HOST"])
	}
}
