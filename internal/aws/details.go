package aws

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// ── Lambda detail ─────────────────────────────────────────────────────────────

type FunctionDetail struct {
	Function
	Description string
	EnvVars     map[string]string
	Triggers    []string
	LogGroup    string
	CodeSize    int64
	LastUpdate  string
}

func (c *Client) GetFunctionDetail(ctx context.Context, name string) (*FunctionDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := lambda.NewFromConfig(c.Config)
	out, err := svc.GetFunction(ctx, &lambda.GetFunctionInput{FunctionName: aws.String(name)})
	if err != nil {
		return nil, err
	}
	cfg := out.Configuration
	d := &FunctionDetail{
		Function: Function{
			Name:    aws.ToString(cfg.FunctionName),
			Runtime: string(cfg.Runtime),
			Memory:  aws.ToInt32(cfg.MemorySize),
			Timeout: aws.ToInt32(cfg.Timeout),
			Handler: aws.ToString(cfg.Handler),
			Role:    aws.ToString(cfg.Role),
		},
		Description: aws.ToString(cfg.Description),
		LogGroup:    aws.ToString(cfg.LoggingConfig.LogGroup),
		CodeSize:    cfg.CodeSize,
		LastUpdate:  parseLambdaTime(aws.ToString(cfg.LastModified)),
	}
	if cfg.Environment != nil {
		d.EnvVars = cfg.Environment.Variables
	}

	// list event source mappings as triggers
	mappings, err := svc.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
		FunctionName: aws.String(name),
	})
	if err == nil {
		for _, m := range mappings.EventSourceMappings {
			d.Triggers = append(d.Triggers, aws.ToString(m.EventSourceArn))
		}
	}
	return d, nil
}

func (c *Client) GetFunctionLogs(ctx context.Context, functionName string, limit int32) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := cloudwatchlogs.NewFromConfig(c.Config)
	logGroup := fmt.Sprintf("/aws/lambda/%s", functionName)

	// get latest 5 log streams
	streams, err := svc.DescribeLogStreams(ctx, &cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: aws.String(logGroup),
		OrderBy:      cwltypes.OrderByLastEventTime,
		Descending:   aws.Bool(true),
		Limit:        aws.Int32(5),
	})
	if err != nil || len(streams.LogStreams) == 0 {
		return nil, err
	}

	type logEntry struct {
		ts  int64
		msg string
	}
	var all []logEntry
	perStream := limit / int32(len(streams.LogStreams))
	if perStream < 5 {
		perStream = 5
	}

	for _, s := range streams.LogStreams {
		events, err := svc.GetLogEvents(ctx, &cloudwatchlogs.GetLogEventsInput{
			LogGroupName:  aws.String(logGroup),
			LogStreamName: s.LogStreamName,
			Limit:         aws.Int32(perStream),
			StartFromHead: aws.Bool(false),
		})
		if err != nil {
			continue
		}
		for _, e := range events.Events {
			all = append(all, logEntry{aws.ToInt64(e.Timestamp), strings.TrimRight(aws.ToString(e.Message), "\n")})
		}
	}

	sort.Slice(all, func(i, j int) bool { return all[i].ts < all[j].ts })

	// cap to limit
	if int32(len(all)) > limit {
		all = all[len(all)-int(limit):]
	}

	var lines []string
	for _, e := range all {
		ts := time.UnixMilli(e.ts).Format("15:04:05")
		lines = append(lines, fmt.Sprintf("%s  %s", ts, e.msg))
	}
	return lines, nil
}

// ── S3 detail ─────────────────────────────────────────────────────────────────

type S3Object struct {
	Key          string
	Size         int64
	LastModified string
	StorageClass string
}

func (c *Client) ListObjects(ctx context.Context, bucket string, prefix string) ([]S3Object, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := s3.NewFromConfig(c.Config)
	out, err := svc.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
		MaxKeys:   aws.Int32(200),
	})
	if err != nil {
		return nil, err
	}
	var objects []S3Object
	// common prefixes (folders)
	for _, p := range out.CommonPrefixes {
		objects = append(objects, S3Object{Key: aws.ToString(p.Prefix), StorageClass: "PREFIX"})
	}
	for _, o := range out.Contents {
		mod := ""
		if o.LastModified != nil {
			mod = o.LastModified.Format("2006-01-02 15:04")
		}
		objects = append(objects, S3Object{
			Key:          aws.ToString(o.Key),
			Size:         aws.ToInt64(o.Size),
			LastModified: mod,
			StorageClass: string(o.StorageClass),
		})
	}
	return objects, nil
}

func (c *Client) GetBucketPolicy(ctx context.Context, bucket string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := s3.NewFromConfig(c.Config)
	out, err := svc.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: aws.String(bucket)})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Policy), nil
}

// ── CloudFormation detail ─────────────────────────────────────────────────────

type StackEvent struct {
	Time         string
	Resource     string
	Type         string
	Status       string
	Reason       string
}

type StackOutput struct {
	Key         string
	Value       string
	Description string
}

type StackResource struct {
	LogicalID  string
	Type       string
	Status     string
	PhysicalID string
}

func (c *Client) GetStackEvents(ctx context.Context, stackName string) ([]StackEvent, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := cloudformation.NewFromConfig(c.Config)
	out, err := svc.DescribeStackEvents(ctx, &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}
	var events []StackEvent
	for _, e := range out.StackEvents {
		ts := ""
		if e.Timestamp != nil {
			ts = e.Timestamp.Format("01-02 15:04:05")
		}
		events = append(events, StackEvent{
			Time:     ts,
			Resource: aws.ToString(e.LogicalResourceId),
			Type:     aws.ToString(e.ResourceType),
			Status:   string(e.ResourceStatus),
			Reason:   aws.ToString(e.ResourceStatusReason),
		})
	}
	return events, nil
}

func (c *Client) GetStackResources(ctx context.Context, stackName string) ([]StackResource, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := cloudformation.NewFromConfig(c.Config)
	out, err := svc.ListStackResources(ctx, &cloudformation.ListStackResourcesInput{
		StackName: aws.String(stackName),
	})
	if err != nil {
		return nil, err
	}
	var resources []StackResource
	for _, r := range out.StackResourceSummaries {
		resources = append(resources, StackResource{
			LogicalID:  aws.ToString(r.LogicalResourceId),
			Type:       aws.ToString(r.ResourceType),
			Status:     string(r.ResourceStatus),
			PhysicalID: aws.ToString(r.PhysicalResourceId),
		})
	}
	return resources, nil
}

func (c *Client) GetStackOutputs(ctx context.Context, stackName string) ([]StackOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := cloudformation.NewFromConfig(c.Config)
	out, err := svc.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	})
	if err != nil || len(out.Stacks) == 0 {
		return nil, err
	}
	var outputs []StackOutput
	for _, o := range out.Stacks[0].Outputs {
		outputs = append(outputs, StackOutput{
			Key:         aws.ToString(o.OutputKey),
			Value:       aws.ToString(o.OutputValue),
			Description: aws.ToString(o.Description),
		})
	}
	return outputs, nil
}

// ── SNS detail ────────────────────────────────────────────────────────────────

type Subscription struct {
	Protocol string
	Endpoint string
	Status   string
}

func (c *Client) GetTopicSubscriptions(ctx context.Context, topicARN string) ([]Subscription, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := sns.NewFromConfig(c.Config)
	out, err := svc.ListSubscriptionsByTopic(ctx, &sns.ListSubscriptionsByTopicInput{
		TopicArn: aws.String(topicARN),
	})
	if err != nil {
		return nil, err
	}
	var subs []Subscription
	for _, s := range out.Subscriptions {
		subs = append(subs, Subscription{
			Protocol: aws.ToString(s.Protocol),
			Endpoint: aws.ToString(s.Endpoint),
			Status:   aws.ToString(s.SubscriptionArn),
		})
	}
	return subs, nil
}

// ── Costs detail ──────────────────────────────────────────────────────────────

func SortCostsByAmount(entries []CostEntry) []CostEntry {
	sorted := make([]CostEntry, len(entries))
	copy(sorted, entries)
	sort.Slice(sorted, func(i, j int) bool {
		var a, b float64
		fmt.Sscanf(sorted[i].Amount, "%f", &a)
		fmt.Sscanf(sorted[j].Amount, "%f", &b)
		return a > b
	})
	return sorted
}

func CostBar(amount, max float64, width int) string {
	if max == 0 {
		return ""
	}
	filled := int(amount / max * float64(width))
	if filled < 1 && amount > 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// parseLambdaTime parses Lambda's LastModified string format.
func parseLambdaTime(s string) string {
	if s == "" {
		return ""
	}
	formats := []string{
		"2006-01-02T15:04:05.000+0000",
		"2006-01-02T15:04:05.999999999Z07:00",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.Format("2006-01-02 15:04")
		}
	}
	return s
}
