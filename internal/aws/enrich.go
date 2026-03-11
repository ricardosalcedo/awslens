package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

// MetricSummary holds 7-day stats for a single metric.
type MetricSummary struct {
	Name string
	Avg  float64
	Max  float64
	Sum  float64
}

// GetMetricSummary returns 7-day avg/max/sum for a metric.
func (c *Client) GetMetricSummary(ctx context.Context, namespace, metric, dimName, dimValue string) (*MetricSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	svc := cloudwatch.NewFromConfig(c.Config)
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)

	out, err := svc.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metric),
		Dimensions: []cwtypes.Dimension{{Name: aws.String(dimName), Value: aws.String(dimValue)}},
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(604800), // single 7-day bucket
		Statistics: []cwtypes.Statistic{cwtypes.StatisticAverage, cwtypes.StatisticMaximum, cwtypes.StatisticSum},
	})
	if err != nil || len(out.Datapoints) == 0 {
		return nil, err
	}
	dp := out.Datapoints[0]
	return &MetricSummary{
		Name: metric,
		Avg:  aws.ToFloat64(dp.Average),
		Max:  aws.ToFloat64(dp.Maximum),
		Sum:  aws.ToFloat64(dp.Sum),
	}, nil
}

// GetRecentErrors searches a log group for ERROR/Exception lines in the last 24h.
func (c *Client) GetRecentErrors(ctx context.Context, logGroup string, limit int32) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	svc := cloudwatchlogs.NewFromConfig(c.Config)
	start := time.Now().Add(-24 * time.Hour).UnixMilli()

	out, err := svc.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(logGroup),
		FilterPattern: aws.String("?ERROR ?Exception ?FATAL ?Error"),
		StartTime:     aws.Int64(start),
		Limit:         aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, e := range out.Events {
		msg := strings.TrimSpace(aws.ToString(e.Message))
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		lines = append(lines, msg)
	}
	return lines, nil
}

// EnrichContext gathers operational data for a resource to feed into AI insight.
type EnrichContext struct {
	Metrics []string
	Errors  []string
	Deps    []string
	Extra   []string
}

func (e *EnrichContext) String() string {
	var b strings.Builder
	if len(e.Metrics) > 0 {
		b.WriteString("\n[7-DAY METRICS]\n")
		for _, m := range e.Metrics {
			b.WriteString("  " + m + "\n")
		}
	}
	if len(e.Errors) > 0 {
		b.WriteString("\n[RECENT ERRORS (24h)]\n")
		for _, e := range e.Errors {
			b.WriteString("  " + e + "\n")
		}
	}
	if len(e.Deps) > 0 {
		b.WriteString("\n[DEPENDENCIES]\n")
		for _, d := range e.Deps {
			b.WriteString("  " + d + "\n")
		}
	}
	if len(e.Extra) > 0 {
		b.WriteString("\n[ADDITIONAL CONTEXT]\n")
		for _, x := range e.Extra {
			b.WriteString("  " + x + "\n")
		}
	}
	return b.String()
}

func fmtMetric(m *MetricSummary, unit string) string {
	if m == nil {
		return ""
	}
	return fmt.Sprintf("%s: avg=%.2f%s max=%.2f%s total=%.0f%s", m.Name, m.Avg, unit, m.Max, unit, m.Sum, unit)
}

// EnrichEC2 gathers CPU, network, and status check metrics.
func (c *Client) EnrichEC2(ctx context.Context, instanceID string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/EC2", "InstanceId"
	if m, _ := c.GetMetricSummary(ctx, ns, "CPUUtilization", dim, instanceID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "%"))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "NetworkIn", dim, instanceID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, " bytes"))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "StatusCheckFailed", dim, instanceID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// EnrichLambda gathers invocations, errors, duration, and dependencies.
func (c *Client) EnrichLambda(ctx context.Context, funcName string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/Lambda", "FunctionName"
	if m, _ := c.GetMetricSummary(ctx, ns, "Invocations", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
		if m.Sum == 0 {
			e.Extra = append(e.Extra, "Function has NOT been invoked in the last 7 days")
		}
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "Errors", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "Duration", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "ms"))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "Throttles", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	// recent errors from logs
	logGroup := fmt.Sprintf("/aws/lambda/%s", funcName)
	if errs, _ := c.GetRecentErrors(ctx, logGroup, 5); len(errs) > 0 {
		e.Errors = errs
	}
	// dependencies
	if deps := c.GetLambdaDeps(ctx, funcName); len(deps) > 0 {
		for _, d := range deps {
			e.Deps = append(e.Deps, fmt.Sprintf("%s --%s--> %s", d.From, d.Relation, d.To))
		}
	}
	return e
}

// EnrichRDS gathers CPU, connections, free storage.
func (c *Client) EnrichRDS(ctx context.Context, dbID string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/RDS", "DBInstanceIdentifier"
	if m, _ := c.GetMetricSummary(ctx, ns, "CPUUtilization", dim, dbID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "%"))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "DatabaseConnections", dim, dbID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "FreeStorageSpace", dim, dbID); m != nil {
		e.Metrics = append(e.Metrics, fmt.Sprintf("FreeStorageSpace: %.0f GB", m.Avg/1073741824))
	}
	return e
}

// EnrichSQS gathers queue depth and age metrics.
func (c *Client) EnrichSQS(ctx context.Context, queueName string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/SQS", "QueueName"
	if m, _ := c.GetMetricSummary(ctx, ns, "ApproximateNumberOfMessagesVisible", dim, queueName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "ApproximateAgeOfOldestMessage", dim, queueName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "s"))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "NumberOfMessagesSent", dim, queueName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// EnrichDynamo gathers read/write capacity and throttle metrics.
func (c *Client) EnrichDynamo(ctx context.Context, tableName string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/DynamoDB", "TableName"
	if m, _ := c.GetMetricSummary(ctx, ns, "ConsumedReadCapacityUnits", dim, tableName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "ConsumedWriteCapacityUnits", dim, tableName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "ThrottledRequests", dim, tableName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// EnrichALB gathers request count, latency, 5xx errors.
func (c *Client) EnrichALB(ctx context.Context, lbArn string) *EnrichContext {
	e := &EnrichContext{}
	// ALB dimension uses the ARN suffix after "app/"
	dimVal := lbArn
	if idx := strings.Index(lbArn, "app/"); idx >= 0 {
		dimVal = lbArn[idx:]
	}
	ns, dim := "AWS/ApplicationELB", "LoadBalancer"
	if m, _ := c.GetMetricSummary(ctx, ns, "RequestCount", dim, dimVal); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "TargetResponseTime", dim, dimVal); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "s"))
	}
	if m, _ := c.GetMetricSummary(ctx, ns, "HTTPCode_ELB_5XX_Count", dim, dimVal); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}
