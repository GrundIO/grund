package aws

import (
	"context"
	"fmt"

	"github.com/GrundIO/grund/internal/application/ports"
	"github.com/GrundIO/grund/internal/domain/infrastructure"
	"github.com/GrundIO/grund/internal/infrastructure/generator"
	"github.com/GrundIO/grund/internal/ui"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// LocalStackProvisioner implements InfrastructureProvisioner for LocalStack
type LocalStackProvisioner struct {
	endpoint string
}

// NewLocalStackProvisioner creates a new LocalStack provisioner
func NewLocalStackProvisioner(endpoint string) ports.InfrastructureProvisioner {
	return &LocalStackProvisioner{
		endpoint: endpoint,
	}
}

// ProvisionPostgres provisions PostgreSQL (not applicable for LocalStack)
func (p *LocalStackProvisioner) ProvisionPostgres(ctx context.Context, config *infrastructure.PostgresConfig) error {
	return fmt.Errorf("postgres provisioning not supported by LocalStack provisioner")
}

// ProvisionMongoDB provisions MongoDB (not applicable for LocalStack)
func (p *LocalStackProvisioner) ProvisionMongoDB(ctx context.Context, config *infrastructure.MongoDBConfig) error {
	return fmt.Errorf("mongodb provisioning not supported by LocalStack provisioner")
}

// ProvisionRedis provisions Redis (not applicable for LocalStack)
func (p *LocalStackProvisioner) ProvisionRedis(ctx context.Context, config *infrastructure.RedisConfig) error {
	return fmt.Errorf("redis provisioning not supported by LocalStack provisioner")
}

// ProvisionLocalStack provisions AWS resources in LocalStack
func (p *LocalStackProvisioner) ProvisionLocalStack(ctx context.Context, req infrastructure.InfrastructureRequirements) (*ports.ProvisionedAWSResources, error) {
	ui.Debug("Connecting to LocalStack at %s", p.endpoint)

	cfg, err := createLocalStackConfig(p.endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS config: %w", err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	snsClient := sns.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true // Required for LocalStack and bucket names with dots
	})

	// Initialize result with provisioned resource details
	result := &ports.ProvisionedAWSResources{
		SQS: make(map[string]ports.ProvisionedQueue),
		SNS: make(map[string]ports.ProvisionedTopic),
		S3:  make(map[string]ports.ProvisionedBucket),
	}

	queueArns := make(map[string]string)

	// Create SQS Queues
	if req.SQS != nil {
		for _, queue := range req.SQS.Queues {
			var dlqURL, dlqARN string

			if queue.DLQ {
				dlqName := queue.Name + "-dlq"
				if existingURL, exists := getExistingQueueURL(ctx, sqsClient, dlqName); exists {
					ui.Infof("SQS DLQ already exists: %s", dlqName)
					dlqURL = existingURL
				} else {
					ui.SubStep("Creating SQS DLQ: %s", dlqName)
					dlqResult, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
						QueueName: aws.String(dlqName),
					})
					if err != nil {
						return nil, fmt.Errorf("failed to create DLQ %s: %w", dlqName, err)
					}
					dlqURL = *dlqResult.QueueUrl
					ui.Successf("Created SQS DLQ: %s", dlqName)
				}
				// Get DLQ ARN
				dlqAttrs, _ := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
					QueueUrl:       aws.String(dlqURL),
					AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
				})
				if dlqAttrs != nil && dlqAttrs.Attributes != nil {
					dlqARN = dlqAttrs.Attributes["QueueArn"]
				}
			}

			var queueURL string
			if existingURL, exists := getExistingQueueURL(ctx, sqsClient, queue.Name); exists {
				ui.Infof("SQS queue already exists: %s", queue.Name)
				queueURL = existingURL
			} else {
				ui.SubStep("Creating SQS queue: %s", queue.Name)
				createResult, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
					QueueName: aws.String(queue.Name),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to create queue %s: %w", queue.Name, err)
				}
				queueURL = *createResult.QueueUrl
				ui.Successf("Created SQS queue: %s", queue.Name)
			}

			// Get queue ARN
			var queueARN string
			attrs, _ := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
				QueueUrl:       aws.String(queueURL),
				AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameQueueArn},
			})
			if attrs != nil && attrs.Attributes != nil {
				queueARN = attrs.Attributes["QueueArn"]
				queueArns[queue.Name] = queueARN
			}

			// Store provisioned queue details
			result.SQS[queue.Name] = ports.ProvisionedQueue{
				Name:   queue.Name,
				URL:    queueURL,
				ARN:    queueARN,
				DLQURL: dlqURL,
				DLQARN: dlqARN,
			}
		}
	}

	// Create SNS Topics
	if req.SNS != nil {
		// Build environment context for template resolution
		envContext := ports.EnvironmentContext{
			LocalStack: ports.LocalStackContext{
				Region:    "us-east-1",
				AccountID: "000000000000",
				Endpoint:  p.endpoint,
			},
			SQS: make(map[string]ports.QueueContext),
		}
		for queueName, arn := range queueArns {
			envContext.SQS[queueName] = ports.QueueContext{
				Name: queueName,
				ARN:  arn,
			}
		}

		resolver := generator.NewEnvironmentResolver()

		for _, topic := range req.SNS.Topics {
			// CreateTopic is idempotent - returns existing topic ARN if exists
			ui.SubStep("Ensuring SNS topic: %s", topic.Name)
			topicResult, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
				Name: aws.String(topic.Name),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create topic %s: %w", topic.Name, err)
			}
			ui.Successf("SNS topic ready: %s", topic.Name)

			// Store provisioned topic details
			result.SNS[topic.Name] = ports.ProvisionedTopic{
				Name: topic.Name,
				ARN:  *topicResult.TopicArn,
			}

			// Subscribe endpoints to topic
			for _, sub := range topic.Subscriptions {
				// Resolve endpoint template (e.g., "${sqs.queue-name.arn}" -> actual ARN)
				resolved, err := resolver.Resolve(
					map[string]string{"endpoint": sub.Endpoint},
					envContext,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve endpoint %s: %w", sub.Endpoint, err)
				}
				endpoint := resolved["endpoint"]

				ui.SubStep("Subscribing %s to topic %s", endpoint, topic.Name)
				subResult, err := snsClient.Subscribe(ctx, &sns.SubscribeInput{
					TopicArn: topicResult.TopicArn,
					Protocol: aws.String(sub.Protocol),
					Endpoint: aws.String(endpoint),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to subscribe %s to %s: %w", endpoint, topic.Name, err)
				}

				// Set subscription attributes (FilterPolicy, FilterPolicyScope, etc.)
				for attrName, attrValue := range sub.Attributes {
					ui.Infof("Setting subscription attribute: %s", attrName)
					_, err := snsClient.SetSubscriptionAttributes(ctx, &sns.SetSubscriptionAttributesInput{
						SubscriptionArn: subResult.SubscriptionArn,
						AttributeName:   aws.String(attrName),
						AttributeValue:  aws.String(attrValue),
					})
					if err != nil {
						return nil, fmt.Errorf("failed to set attribute %s on subscription: %w", attrName, err)
					}
				}
			}
		}
	}

	// Create S3 Buckets
	if req.S3 != nil {
		for _, bucket := range req.S3.Buckets {
			if getExistingBucket(ctx, s3Client, bucket.Name) {
				ui.Infof("S3 bucket already exists: %s", bucket.Name)
			} else {
				ui.SubStep("Creating S3 bucket: %s", bucket.Name)
				_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
					Bucket: aws.String(bucket.Name),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to create bucket %s: %w", bucket.Name, err)
				}
				ui.Successf("Created S3 bucket: %s", bucket.Name)
			}

			if bucket.CORS {
				ui.SubStep("Configuring CORS for bucket: %s", bucket.Name)
				_, err := s3Client.PutBucketCors(ctx, &s3.PutBucketCorsInput{
					Bucket: aws.String(bucket.Name),
					CORSConfiguration: &s3types.CORSConfiguration{
						CORSRules: []s3types.CORSRule{
							{
								AllowedOrigins: []string{"*"},
								AllowedMethods: []string{"GET", "PUT", "POST", "DELETE", "HEAD"},
								AllowedHeaders: []string{"*"},
							},
						},
					},
				})
				if err != nil {
					return nil, fmt.Errorf("failed to configure CORS for bucket %s: %w", bucket.Name, err)
				}
				ui.Successf("Configured CORS for bucket: %s", bucket.Name)
			}

			// Store provisioned bucket details (use path-style URL for LocalStack)
			result.S3[bucket.Name] = ports.ProvisionedBucket{
				Name: bucket.Name,
				URL:  fmt.Sprintf("%s/%s", p.endpoint, bucket.Name),
			}
		}
	}

	return result, nil
}

// getExistingQueueURL checks if a queue already exists and returns its URL
func getExistingQueueURL(ctx context.Context, client *sqs.Client, queueName string) (string, bool) {
	result, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", false
	}
	return *result.QueueUrl, true
}

// getExistingBucket checks if a bucket already exists
func getExistingBucket(ctx context.Context, client *s3.Client, bucketName string) bool {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})
	return err == nil
}

func createLocalStackConfig(endpoint string) (aws.Config, error) {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(
			func(service, region string, options ...interface{}) (aws.Endpoint, error) {
				return aws.Endpoint{
					URL:           endpoint,
					SigningRegion: "us-east-1",
				}, nil
			})),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test", "test", "")),
	)
	return cfg, err
}
