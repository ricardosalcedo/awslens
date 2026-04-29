package aws

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	costypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"time"
)

// ── EC2 ──────────────────────────────────────────────────────────────────────

type Instance struct {
	ID         string
	Name       string
	State      string
	Type       string
	AZ         string
	PublicIP   string
	LaunchTime string
	Region     string
}

func (c *Client) ListInstances(ctx context.Context) ([]Instance, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := ec2.NewFromConfig(c.Config)
	out, err := svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, err
	}
	var instances []Instance
	for _, r := range out.Reservations {
		for _, i := range r.Instances {
			inst := Instance{
				ID:     aws.ToString(i.InstanceId),
				State:  string(i.State.Name),
				Type:   string(i.InstanceType),
				AZ:     aws.ToString(i.Placement.AvailabilityZone),
				Region: c.Region,
			}
			if i.PublicIpAddress != nil {
				inst.PublicIP = aws.ToString(i.PublicIpAddress)
			}
			if i.LaunchTime != nil {
				inst.LaunchTime = i.LaunchTime.Format("2006-01-02")
			}
			for _, tag := range i.Tags {
				if aws.ToString(tag.Key) == "Name" {
					inst.Name = aws.ToString(tag.Value)
				}
			}
			instances = append(instances, inst)
		}
	}
	return instances, nil
}

func (c *Client) StartInstance(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := ec2.NewFromConfig(c.Config)
	_, err := svc.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{id}})
	return err
}

func (c *Client) StopInstance(ctx context.Context, id string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := ec2.NewFromConfig(c.Config)
	_, err := svc.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{id}})
	return err
}

func (c *Client) ListSecurityGroups(ctx context.Context) ([]ec2types.SecurityGroup, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := ec2.NewFromConfig(c.Config)
	out, err := svc.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{})
	if err != nil {
		return nil, err
	}
	return out.SecurityGroups, nil
}

// ── LAMBDA ───────────────────────────────────────────────────────────────────

type Function struct {
	Name         string
	Runtime      string
	Memory       int32
	Timeout      int32
	LastModified string
	Handler      string
	Role         string
	Region       string
}

func (c *Client) ListFunctions(ctx context.Context) ([]Function, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := lambda.NewFromConfig(c.Config)
	var fns []Function
	paginator := lambda.NewListFunctionsPaginator(svc, &lambda.ListFunctionsInput{})
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, f := range out.Functions {
			fns = append(fns, Function{
				Name:         aws.ToString(f.FunctionName),
				Runtime:      string(f.Runtime),
				Memory:       aws.ToInt32(f.MemorySize),
				Timeout:      aws.ToInt32(f.Timeout),
				LastModified: parseLambdaTime(aws.ToString(f.LastModified)),
				Handler:      aws.ToString(f.Handler),
				Role:         aws.ToString(f.Role),
				Region:       c.Region,
			})
		}
	}
	return fns, nil
}

func (c *Client) InvokeFunction(ctx context.Context, name string, payload []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := lambda.NewFromConfig(c.Config)
	out, err := svc.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: aws.String(name),
		Payload:      payload,
	})
	if err != nil {
		return nil, err
	}
	return out.Payload, nil
}

// ── S3 ───────────────────────────────────────────────────────────────────────

type Bucket struct {
	Name         string
	CreationDate string
	Region       string
}

func (c *Client) ListBuckets(ctx context.Context) ([]Bucket, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := s3.NewFromConfig(c.Config)
	out, err := svc.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var buckets []Bucket
	for _, b := range out.Buckets {
		bucket := Bucket{Name: aws.ToString(b.Name)}
		if b.CreationDate != nil {
			bucket.CreationDate = b.CreationDate.Format("2006-01-02")
		}
		// get bucket region
		loc, err := svc.GetBucketLocation(ctx, &s3.GetBucketLocationInput{Bucket: b.Name})
		if err == nil {
			bucket.Region = string(loc.LocationConstraint)
			if bucket.Region == "" {
				bucket.Region = "us-east-1"
			}
		}
		buckets = append(buckets, bucket)
	}
	return buckets, nil
}

// ── RDS ──────────────────────────────────────────────────────────────────────

type DBInstance struct {
	ID       string
	Engine   string
	Status   string
	Class    string
	Endpoint string
	MultiAZ  bool
}

func (c *Client) ListDBInstances(ctx context.Context) ([]DBInstance, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := rds.NewFromConfig(c.Config)
	out, err := svc.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, err
	}
	var dbs []DBInstance
	for _, db := range out.DBInstances {
		d := DBInstance{
			ID:      aws.ToString(db.DBInstanceIdentifier),
			Engine:  aws.ToString(db.Engine) + " " + aws.ToString(db.EngineVersion),
			Status:  aws.ToString(db.DBInstanceStatus),
			Class:   aws.ToString(db.DBInstanceClass),
			MultiAZ: aws.ToBool(db.MultiAZ),
		}
		if db.Endpoint != nil {
			d.Endpoint = fmt.Sprintf("%s:%d", aws.ToString(db.Endpoint.Address), aws.ToInt32(db.Endpoint.Port))
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

// ── ECS ──────────────────────────────────────────────────────────────────────

type Cluster struct {
	Name             string
	Status           string
	RunningTasks     int32
	PendingTasks     int32
	ActiveServices   int32
}

func (c *Client) ListClusters(ctx context.Context) ([]Cluster, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := ecs.NewFromConfig(c.Config)
	arns, err := svc.ListClusters(ctx, &ecs.ListClustersInput{})
	if err != nil {
		return nil, err
	}
	if len(arns.ClusterArns) == 0 {
		return nil, nil
	}
	out, err := svc.DescribeClusters(ctx, &ecs.DescribeClustersInput{Clusters: arns.ClusterArns})
	if err != nil {
		return nil, err
	}
	var clusters []Cluster
	for _, cl := range out.Clusters {
		clusters = append(clusters, Cluster{
			Name:           aws.ToString(cl.ClusterName),
			Status:         aws.ToString(cl.Status),
			RunningTasks:   cl.RunningTasksCount,
			PendingTasks:   cl.PendingTasksCount,
			ActiveServices: cl.ActiveServicesCount,
		})
	}
	return clusters, nil
}

// ── SQS ──────────────────────────────────────────────────────────────────────

type Queue struct {
	URL              string
	Name             string
	Messages         string
	MessagesInFlight string
}

func (c *Client) ListQueues(ctx context.Context) ([]Queue, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := sqs.NewFromConfig(c.Config)
	out, err := svc.ListQueues(ctx, &sqs.ListQueuesInput{})
	if err != nil {
		return nil, err
	}
	var queues []Queue
	for _, url := range out.QueueUrls {
		q := Queue{URL: url}
		// extract name from URL
		parts := splitLast(url, "/")
		q.Name = parts
		// get attributes
		attrs, err := svc.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(url),
			AttributeNames: []sqstypes.QueueAttributeName{"ApproximateNumberOfMessages", "ApproximateNumberOfMessagesNotVisible"},
		})
		if err == nil {
			q.Messages = attrs.Attributes["ApproximateNumberOfMessages"]
			q.MessagesInFlight = attrs.Attributes["ApproximateNumberOfMessagesNotVisible"]
		}
		queues = append(queues, q)
	}
	return queues, nil
}

// ── SNS ──────────────────────────────────────────────────────────────────────

type Topic struct {
	ARN  string
	Name string
}

func (c *Client) ListTopics(ctx context.Context) ([]Topic, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := sns.NewFromConfig(c.Config)
	out, err := svc.ListTopics(ctx, &sns.ListTopicsInput{})
	if err != nil {
		return nil, err
	}
	var topics []Topic
	for _, t := range out.Topics {
		arn := aws.ToString(t.TopicArn)
		topics = append(topics, Topic{ARN: arn, Name: splitLast(arn, ":")})
	}
	return topics, nil
}

// ── CLOUDFORMATION ───────────────────────────────────────────────────────────

type Stack struct {
	Name   string
	Status string
	Drift  string
	Region string
}

func (c *Client) ListStacks(ctx context.Context) ([]Stack, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := cloudformation.NewFromConfig(c.Config)
	out, err := svc.DescribeStacks(ctx, &cloudformation.DescribeStacksInput{})
	if err != nil {
		return nil, err
	}
	var stacks []Stack
	for _, s := range out.Stacks {
		stacks = append(stacks, Stack{
			Name:   aws.ToString(s.StackName),
			Status: string(s.StackStatus),
			Drift:  string(s.DriftInformation.StackDriftStatus),
			Region: c.Region,
		})
	}
	return stacks, nil
}

// ── COSTS ────────────────────────────────────────────────────────────────────

type CostEntry struct {
	Service   string
	Amount    string
	Unit      string
	PrevMonth string // last month's amount for comparison
}

func (c *Client) GetMonthlyCosts(ctx context.Context, monthsBack int) ([]CostEntry, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := costexplorer.NewFromConfig(c.Config)
	now := time.Now()

	// Shift the window back by monthsBack months
	ref := now.AddDate(0, -monthsBack, 0)
	var end string
	if monthsBack == 0 {
		end = now.Format("2006-01-02")
	} else {
		next := ref.AddDate(0, 1, 0)
		end = fmt.Sprintf("%d-%02d-01", next.Year(), next.Month())
	}

	// also get previous month for anomaly comparison
	prevStart := ref.AddDate(0, -1, 0)
	prevStartStr := fmt.Sprintf("%d-%02d-01", prevStart.Year(), prevStart.Month())

	out, err := svc.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &costypes.DateInterval{
			Start: aws.String(prevStartStr),
			End:   aws.String(end),
		},
		Granularity: costypes.GranularityMonthly,
		GroupBy: []costypes.GroupDefinition{{
			Type: costypes.GroupDefinitionTypeDimension,
			Key:  aws.String("SERVICE"),
		}},
		Metrics: []string{"UnblendedCost"},
	})
	if err != nil {
		return nil, "", err
	}

	// collect per-service costs by period
	prevCosts := map[string]string{}
	var entries []CostEntry
	total := 0.0
	for i, result := range out.ResultsByTime {
		for _, group := range result.Groups {
			amt := aws.ToString(group.Metrics["UnblendedCost"].Amount)
			unit := aws.ToString(group.Metrics["UnblendedCost"].Unit)
			if i == 0 {
				// last month
				prevCosts[group.Keys[0]] = amt
			} else {
				// current month
				entries = append(entries, CostEntry{
					Service:   group.Keys[0],
					Amount:    amt,
					Unit:      unit,
					PrevMonth: prevCosts[group.Keys[0]],
				})
				var f float64
				if v, err := strconv.ParseFloat(amt, 64); err == nil {
					f = v
				}
				total += f
			}
		}
	}
	return entries, fmt.Sprintf("%.4f", total), nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func splitLast(s, sep string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return s[i+1:]
		}
	}
	return s
}

// ── Multi-region aggregation ──────────────────────────────────────────────────

// RegionResult tags any resource with its source region.
type RegionResult[T any] struct {
	Region string
	Items  []T
	Err    error
}

func AllRegionsInstances(ctx context.Context, c *Client) []Instance {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Instance, error) {
		return rc.ListInstances(ctx)
	})
}

func AllRegionsFunctions(ctx context.Context, c *Client) []Function {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Function, error) {
		return rc.ListFunctions(ctx)
	})
}

func AllRegionsDynamoTables(ctx context.Context, c *Client) []DynamoTable {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]DynamoTable, error) {
		return rc.ListDynamoTables(ctx)
	})
}

func AllRegionsRestAPIs(ctx context.Context, c *Client) []RestAPI {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]RestAPI, error) {
		return rc.ListRestAPIs(ctx)
	})
}

func AllRegionsStacks(ctx context.Context, c *Client) []Stack {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Stack, error) {
		return rc.ListStacks(ctx)
	})
}

func AllRegionsAlarms(ctx context.Context, c *Client) []CWAlarm {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]CWAlarm, error) {
		return rc.ListAlarms(ctx)
	})
}

func AllRegionsStateMachines(ctx context.Context, c *Client) []StateMachine {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]StateMachine, error) {
		return rc.ListStateMachines(ctx)
	})
}

func AllRegionsECRRepos(ctx context.Context, c *Client) []ECRRepo {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]ECRRepo, error) {
		return rc.ListECRRepos(ctx)
	})
}

func AllRegionsLoadBalancers(ctx context.Context, c *Client) []LoadBalancer {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]LoadBalancer, error) {
		return rc.ListLoadBalancers(ctx)
	})
}

func AllRegionsQueues(ctx context.Context, c *Client) []Queue {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Queue, error) {
		return rc.ListQueues(ctx)
	})
}

func AllRegionsTopics(ctx context.Context, c *Client) []Topic {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Topic, error) {
		return rc.ListTopics(ctx)
	})
}

func AllRegionsDBInstances(ctx context.Context, c *Client) []DBInstance {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]DBInstance, error) {
		return rc.ListDBInstances(ctx)
	})
}

func AllRegionsClusters(ctx context.Context, c *Client) []Cluster {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Cluster, error) {
		return rc.ListClusters(ctx)
	})
}

func AllRegionsSSMParams(ctx context.Context, c *Client) []SSMParam {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]SSMParam, error) {
		return rc.ListSSMParams(ctx)
	})
}

func AllRegionsSecrets(ctx context.Context, c *Client) []Secret {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Secret, error) {
		return rc.ListSecrets(ctx)
	})
}

// aggregateRegions fans out a fetch function across all common regions in parallel
// and merges results. Regions that return errors are logged at debug level via
// log/slog so they can be diagnosed with --debug.
func aggregateRegions[T any](ctx context.Context, c *Client, fn func(context.Context, *Client) ([]T, error)) []T {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	type result struct {
		region string
		items  []T
		err    error
	}
	ch := make(chan result, len(CommonRegions))
	for _, r := range CommonRegions {
		r := r
		go func() {
			rc := c.NewForRegion(r)
			items, err := fn(ctx, rc)
			ch <- result{region: r, items: items, err: err}
		}()
	}
	var all []T
	for range CommonRegions {
		r := <-ch
		if r.err != nil {
			slog.Debug("region fetch failed", "region", r.region, "error", r.err)
			continue
		}
		all = append(all, r.items...)
	}
	return all
}
