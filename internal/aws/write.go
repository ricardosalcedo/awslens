package aws

import (
	"context"
	"time"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// ── S3 ────────────────────────────────────────────────────────────────────────

func (c *Client) CreateBucket(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := s3.NewFromConfig(c.Config)
	input := &s3.CreateBucketInput{Bucket: aws.String(name)}
	if c.Config.Region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(c.Config.Region),
		}
	}
	_, err := svc.CreateBucket(ctx, input)
	return err
}

func (c *Client) DeleteBucket(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := s3.NewFromConfig(c.Config).DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(name)})
	return err
}

// ── Lambda ────────────────────────────────────────────────────────────────────

func (c *Client) DeleteFunction(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := lambda.NewFromConfig(c.Config).DeleteFunction(ctx, &lambda.DeleteFunctionInput{FunctionName: aws.String(name)})
	return err
}

// ── DynamoDB ──────────────────────────────────────────────────────────────────

func (c *Client) DeleteTable(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := dynamodb.NewFromConfig(c.Config).DeleteTable(ctx, &dynamodb.DeleteTableInput{TableName: aws.String(name)})
	return err
}

func (c *Client) PutDynamoItem(ctx context.Context, table, jsonItem string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	item, err := parseDynamoJSON(jsonItem)
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	_, err = dynamodb.NewFromConfig(c.Config).PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      item,
	})
	return err
}

func parseDynamoJSON(s string) (map[string]dtypes.AttributeValue, error) {
	// simple: treat input as key=value pairs, all as strings
	// e.g. "id=123 name=foo"
	result := map[string]dtypes.AttributeValue{}
	for _, part := range splitFields(s) {
		kv := SplitKV(part)
		if len(kv) == 2 {
			result[kv[0]] = &dtypes.AttributeValueMemberS{Value: kv[1]}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no key=value pairs found")
	}
	return result, nil
}

func splitFields(s string) []string {
	var fields []string
	for _, f := range splitOn(s, ' ') {
		if f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func splitOn(s string, sep rune) []string {
	var parts []string
	cur := ""
	for _, r := range s {
		if r == sep {
			parts = append(parts, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	return append(parts, cur)
}

func SplitKV(s string) []string {
	for i, r := range s {
		if r == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// ── SQS ───────────────────────────────────────────────────────────────────────

func (c *Client) CreateQueue(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := sqs.NewFromConfig(c.Config).CreateQueue(ctx, &sqs.CreateQueueInput{QueueName: aws.String(name)})
	return err
}

func (c *Client) DeleteQueue(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := sqs.NewFromConfig(c.Config).DeleteQueue(ctx, &sqs.DeleteQueueInput{QueueUrl: aws.String(url)})
	return err
}

func (c *Client) PurgeQueue(ctx context.Context, url string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := sqs.NewFromConfig(c.Config).PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: aws.String(url)})
	return err
}

// ── SNS ───────────────────────────────────────────────────────────────────────

func (c *Client) CreateTopic(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := sns.NewFromConfig(c.Config).CreateTopic(ctx, &sns.CreateTopicInput{Name: aws.String(name)})
	return err
}

func (c *Client) DeleteTopic(ctx context.Context, arn string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := sns.NewFromConfig(c.Config).DeleteTopic(ctx, &sns.DeleteTopicInput{TopicArn: aws.String(arn)})
	return err
}

// ── SSM ───────────────────────────────────────────────────────────────────────

func (c *Client) PutSSMParam(ctx context.Context, name, value string, overwrite bool) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := ssm.NewFromConfig(c.Config).PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      ssmtypes.ParameterTypeString,
		Overwrite: aws.Bool(overwrite),
	})
	return err
}

func (c *Client) DeleteSSMParam(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := ssm.NewFromConfig(c.Config).DeleteParameter(ctx, &ssm.DeleteParameterInput{Name: aws.String(name)})
	return err
}

// ── Secrets Manager ───────────────────────────────────────────────────────────

func (c *Client) CreateSecret(ctx context.Context, name, value string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := secretsmanager.NewFromConfig(c.Config).CreateSecret(ctx, &secretsmanager.CreateSecretInput{
		Name:         aws.String(name),
		SecretString: aws.String(value),
	})
	return err
}

func (c *Client) UpdateSecret(ctx context.Context, arn, value string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := secretsmanager.NewFromConfig(c.Config).UpdateSecret(ctx, &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(arn),
		SecretString: aws.String(value),
	})
	return err
}

func (c *Client) DeleteSecret(ctx context.Context, arn string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := secretsmanager.NewFromConfig(c.Config).DeleteSecret(ctx, &secretsmanager.DeleteSecretInput{
		SecretId:                   aws.String(arn),
		ForceDeleteWithoutRecovery: aws.Bool(false), // 7-day recovery window
	})
	return err
}

// ── ECR ───────────────────────────────────────────────────────────────────────

func (c *Client) CreateECRRepo(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := ecr.NewFromConfig(c.Config).CreateRepository(ctx, &ecr.CreateRepositoryInput{RepositoryName: aws.String(name)})
	return err
}

func (c *Client) DeleteECRRepo(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := ecr.NewFromConfig(c.Config).DeleteRepository(ctx, &ecr.DeleteRepositoryInput{
		RepositoryName: aws.String(name),
		Force:          true,
	})
	return err
}

// ── CodeCommit ────────────────────────────────────────────────────────────────

func (c *Client) CreateCodeRepo(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := codecommit.NewFromConfig(c.Config).CreateRepository(ctx, &codecommit.CreateRepositoryInput{RepositoryName: aws.String(name)})
	return err
}

func (c *Client) DeleteCodeRepo(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := codecommit.NewFromConfig(c.Config).DeleteRepository(ctx, &codecommit.DeleteRepositoryInput{RepositoryName: aws.String(name)})
	return err
}

// ── EventBridge ───────────────────────────────────────────────────────────────

func (c *Client) DisableEBRule(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := eventbridge.NewFromConfig(c.Config).DisableRule(ctx, &eventbridge.DisableRuleInput{Name: aws.String(name)})
	return err
}

func (c *Client) EnableEBRule(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_, err := eventbridge.NewFromConfig(c.Config).EnableRule(ctx, &eventbridge.EnableRuleInput{Name: aws.String(name)})
	return err
}

func (c *Client) DeleteEBRule(ctx context.Context, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	// must remove targets first
	svc := eventbridge.NewFromConfig(c.Config)
	tgts, err := svc.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{Rule: aws.String(name)})
	if err == nil && len(tgts.Targets) > 0 {
		ids := make([]string, len(tgts.Targets))
		for i, t := range tgts.Targets {
			ids[i] = aws.ToString(t.Id)
		}
		svc.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{Rule: aws.String(name), Ids: ids})
	}
	_, err = svc.DeleteRule(ctx, &eventbridge.DeleteRuleInput{Name: aws.String(name)})
	return err
}
