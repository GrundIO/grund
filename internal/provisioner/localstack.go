package provisioner

import (
	"context"
	"fmt"

	"github.com/GrundIO/grund/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// AWSResources aggregates all AWS resources needed
type AWSResources struct {
	SQS []config.QueueConfig
	SNS []config.TopicConfig
	S3  []config.BucketConfig
}

// ProvisionAWSResources provisions all AWS resources in LocalStack
func ProvisionAWSResources(ctx context.Context, resources AWSResources, endpoint string) error {
	// Create AWS config pointing to LocalStack
	cfg, err := createLocalStackConfig(endpoint)
	if err != nil {
		return fmt.Errorf("failed to create AWS config: %w", err)
	}

	sqsClient := sqs.NewFromConfig(cfg)
	snsClient := sns.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)

	queueArns := make(map[string]string)

	// Create SQS Queues
	for _, queue := range resources.SQS {
		// Create DLQ first if requested
		if queue.DLQ {
			dlqName := queue.Name + "-dlq"
			_, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
				QueueName: aws.String(dlqName),
			})
			if err != nil {
				return fmt.Errorf("failed to create DLQ %s: %w", dlqName, err)
			}
		}

		result, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
			QueueName: aws.String(queue.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to create queue %s: %w", queue.Name, err)
		}

		// Get queue ARN for SNS subscription
		attrs, _ := sqsClient.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       result.QueueUrl,
			AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameQueueArn},
		})
		if attrs.Attributes != nil {
			queueArns[queue.Name] = attrs.Attributes["QueueArn"]
		}
	}

	// Create SNS Topics
	for _, topic := range resources.SNS {
		result, err := snsClient.CreateTopic(ctx, &sns.CreateTopicInput{
			Name: aws.String(topic.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic.Name, err)
		}

		// Subscribe endpoints to topic
		for _, sub := range topic.Subscriptions {
			// For now, this provisioner expects resolved endpoints (ARNs directly)
			// The main provisioner in infrastructure/aws handles template resolution
			subResult, err := snsClient.Subscribe(ctx, &sns.SubscribeInput{
				TopicArn: result.TopicArn,
				Protocol: aws.String(sub.Protocol),
				Endpoint: aws.String(sub.Endpoint),
			})
			if err != nil {
				return fmt.Errorf("failed to subscribe %s to %s: %w", sub.Endpoint, topic.Name, err)
			}

			// Set subscription attributes (FilterPolicy, FilterPolicyScope, etc.)
			for attrName, attrValue := range sub.Attributes {
				_, err := snsClient.SetSubscriptionAttributes(ctx, &sns.SetSubscriptionAttributesInput{
					SubscriptionArn: subResult.SubscriptionArn,
					AttributeName:   aws.String(attrName),
					AttributeValue:  aws.String(attrValue),
				})
				if err != nil {
					return fmt.Errorf("failed to set attribute %s on subscription: %w", attrName, err)
				}
			}
		}
	}

	// Create S3 Buckets
	for _, bucket := range resources.S3 {
		_, err := s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucket.Name),
		})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucket.Name, err)
		}

		// TODO: Upload seed files if specified
		if bucket.Seed != "" {
			// Implement seed file upload
		}
	}

	return nil
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
