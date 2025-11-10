package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"

	"github.com/comfortablynumb/pmp-cloud-inspector/pkg/resource"
)

// collectSNSTopics collects SNS topics from a region
func (p *Provider) collectSNSTopics(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting SNS topics in %s...\n", region)

	client := sns.NewFromConfig(cfg)

	// List all topics
	paginator := sns.NewListTopicsPaginator(client, &sns.ListTopicsInput{})

	topicCount := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list SNS topics: %w", err)
		}

		for _, topic := range output.Topics {
			if topic.TopicArn == nil {
				continue
			}

			// Get topic attributes
			attrs, err := client.GetTopicAttributes(ctx, &sns.GetTopicAttributesInput{
				TopicArn: topic.TopicArn,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: failed to get attributes for topic %s: %v\n", *topic.TopicArn, err)
				continue
			}

			properties := make(map[string]interface{})
			for key, value := range attrs.Attributes {
				properties[key] = value
			}

			res := &resource.Resource{
				ID:         *topic.TopicArn,
				Type:       resource.TypeAWSSNSTopic,
				Name:       extractTopicName(*topic.TopicArn),
				Provider:   "aws",
				Region:     region,
				ARN:        *topic.TopicArn,
				Properties: properties,
				RawData:    topic,
			}

			collection.Add(res)
			topicCount++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d SNS topics\n", topicCount)
	return nil
}

// collectSQSQueues collects SQS queues from a region
func (p *Provider) collectSQSQueues(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting SQS queues in %s...\n", region)

	client := sqs.NewFromConfig(cfg)

	// List all queues
	output, err := client.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return fmt.Errorf("failed to list SQS queues: %w", err)
	}

	queueCount := 0
	for _, queueURL := range output.QueueUrls {
		// Get queue attributes
		attrs, err := client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(queueURL),
			AttributeNames: []sqstypes.QueueAttributeName{sqstypes.QueueAttributeNameAll},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "    Warning: failed to get attributes for queue %s: %v\n", queueURL, err)
			continue
		}

		properties := make(map[string]interface{})
		for key, value := range attrs.Attributes {
			properties[key] = value
		}

		// Extract queue ARN
		queueARN := attrs.Attributes["QueueArn"]

		res := &resource.Resource{
			ID:         queueURL,
			Type:       resource.TypeAWSSQSQueue,
			Name:       extractQueueName(queueURL),
			Provider:   "aws",
			Region:     region,
			ARN:        queueARN,
			Properties: properties,
			RawData:    queueURL,
		}

		collection.Add(res)
		queueCount++
	}

	fmt.Fprintf(os.Stderr, "    Found %d SQS queues\n", queueCount)
	return nil
}

// collectDynamoDBTables collects DynamoDB tables from a region
func (p *Provider) collectDynamoDBTables(ctx context.Context, collection *resource.Collection, region string, cfg aws.Config) error {
	fmt.Fprintf(os.Stderr, "  Collecting DynamoDB tables in %s...\n", region)

	client := dynamodb.NewFromConfig(cfg)

	// List all tables
	paginator := dynamodb.NewListTablesPaginator(client, &dynamodb.ListTablesInput{})

	tableCount := 0
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list DynamoDB tables: %w", err)
		}

		for _, tableName := range output.TableNames {
			// Describe table
			desc, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(tableName),
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "    Warning: failed to describe table %s: %v\n", tableName, err)
				continue
			}

			table := desc.Table
			if table == nil {
				continue
			}

			properties := map[string]interface{}{
				"TableStatus":            table.TableStatus,
				"ItemCount":              table.ItemCount,
				"TableSizeBytes":         table.TableSizeBytes,
				"BillingModeSummary":     table.BillingModeSummary,
				"ProvisionedThroughput":  table.ProvisionedThroughput,
				"KeySchema":              table.KeySchema,
				"AttributeDefinitions":   table.AttributeDefinitions,
				"GlobalSecondaryIndexes": table.GlobalSecondaryIndexes,
				"LocalSecondaryIndexes":  table.LocalSecondaryIndexes,
			}

			if table.StreamSpecification != nil {
				properties["StreamEnabled"] = table.StreamSpecification.StreamEnabled
				properties["StreamViewType"] = table.StreamSpecification.StreamViewType
			}

			res := &resource.Resource{
				ID:         *table.TableName,
				Type:       resource.TypeAWSDynamoDBTable,
				Name:       *table.TableName,
				Provider:   "aws",
				Region:     region,
				ARN:        aws.ToString(table.TableArn),
				Properties: properties,
				RawData:    table,
			}

			collection.Add(res)
			tableCount++
		}
	}

	fmt.Fprintf(os.Stderr, "    Found %d DynamoDB tables\n", tableCount)
	return nil
}

// Helper functions to extract names from ARNs and URLs

func extractTopicName(arn string) string {
	// ARN format: arn:aws:sns:region:account-id:topic-name
	parts := parseARN(arn)
	if len(parts) > 5 {
		return parts[5]
	}
	return arn
}

func extractQueueName(url string) string {
	// URL format: https://sqs.region.amazonaws.com/account-id/queue-name
	// Extract the last part after the final /
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			return url[i+1:]
		}
	}
	return url
}

func parseARN(arn string) []string {
	parts := []string{}
	current := ""
	for _, ch := range arn {
		if ch == ':' {
			parts = append(parts, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
