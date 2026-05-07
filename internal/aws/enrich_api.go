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
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

// GetMetricSummaryWithAPI returns 7-day avg/max/sum for a metric using the provided API.
func GetMetricSummaryWithAPI(ctx context.Context, api CloudWatchAPI, namespace, metric, dimName, dimValue string) (*MetricSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour)

	out, err := api.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		MetricName: aws.String(metric),
		Dimensions: []cwtypes.Dimension{{Name: aws.String(dimName), Value: aws.String(dimValue)}},
		StartTime:  aws.Time(start),
		EndTime:    aws.Time(end),
		Period:     aws.Int32(604800),
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

// GetRecentErrorsWithAPI searches a log group for ERROR/Exception lines using the provided API.
func GetRecentErrorsWithAPI(ctx context.Context, api CloudWatchLogsAPI, logGroup string, limit int32) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now().Add(-24 * time.Hour).UnixMilli()

	out, err := api.FilterLogEvents(ctx, &cloudwatchlogs.FilterLogEventsInput{
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

// EnrichEC2WithAPI gathers CPU, network, and status check metrics using the provided API.
func EnrichEC2WithAPI(ctx context.Context, api CloudWatchAPI, instanceID string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/EC2", "InstanceId"
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "CPUUtilization", dim, instanceID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "%"))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "NetworkIn", dim, instanceID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, " bytes"))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "StatusCheckFailed", dim, instanceID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// EnrichLambdaWithAPI gathers invocations, errors, duration, dependencies, and log errors.
func EnrichLambdaWithAPI(ctx context.Context, cwAPI CloudWatchAPI, logsAPI CloudWatchLogsAPI, lambdaAPI LambdaGetAPI, funcName string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/Lambda", "FunctionName"
	if m, _ := GetMetricSummaryWithAPI(ctx, cwAPI, ns, "Invocations", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
		if m.Sum == 0 {
			e.Extra = append(e.Extra, "Function has NOT been invoked in the last 7 days")
		}
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, cwAPI, ns, "Errors", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, cwAPI, ns, "Duration", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "ms"))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, cwAPI, ns, "Throttles", dim, funcName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	// recent errors from logs
	logGroup := fmt.Sprintf("/aws/lambda/%s", funcName)
	if errs, _ := GetRecentErrorsWithAPI(ctx, logsAPI, logGroup, 5); len(errs) > 0 {
		e.Errors = errs
	}
	// dependencies
	if deps := GetLambdaDepsWithAPI(ctx, lambdaAPI, funcName); len(deps) > 0 {
		for _, d := range deps {
			e.Deps = append(e.Deps, fmt.Sprintf("%s --%s--> %s", d.From, d.Relation, d.To))
		}
	}
	return e
}

// EnrichRDSWithAPI gathers CPU, connections, free storage using the provided API.
func EnrichRDSWithAPI(ctx context.Context, api CloudWatchAPI, dbID string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/RDS", "DBInstanceIdentifier"
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "CPUUtilization", dim, dbID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "%"))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "DatabaseConnections", dim, dbID); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "FreeStorageSpace", dim, dbID); m != nil {
		e.Metrics = append(e.Metrics, fmt.Sprintf("FreeStorageSpace: %.0f GB", m.Avg/1073741824))
	}
	return e
}

// EnrichSQSWithAPI gathers queue depth and age metrics using the provided API.
func EnrichSQSWithAPI(ctx context.Context, api CloudWatchAPI, queueName string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/SQS", "QueueName"
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "ApproximateNumberOfMessagesVisible", dim, queueName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "ApproximateAgeOfOldestMessage", dim, queueName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "s"))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "NumberOfMessagesSent", dim, queueName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// EnrichDynamoWithAPI gathers read/write capacity and throttle metrics using the provided API.
func EnrichDynamoWithAPI(ctx context.Context, api CloudWatchAPI, tableName string) *EnrichContext {
	e := &EnrichContext{}
	ns, dim := "AWS/DynamoDB", "TableName"
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "ConsumedReadCapacityUnits", dim, tableName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "ConsumedWriteCapacityUnits", dim, tableName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "ThrottledRequests", dim, tableName); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// EnrichALBWithAPI gathers request count, latency, 5xx errors using the provided API.
func EnrichALBWithAPI(ctx context.Context, api CloudWatchAPI, lbArn string) *EnrichContext {
	e := &EnrichContext{}
	dimVal := lbArn
	if idx := strings.Index(lbArn, "app/"); idx >= 0 {
		dimVal = lbArn[idx:]
	}
	ns, dim := "AWS/ApplicationELB", "LoadBalancer"
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "RequestCount", dim, dimVal); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "TargetResponseTime", dim, dimVal); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, "s"))
	}
	if m, _ := GetMetricSummaryWithAPI(ctx, api, ns, "HTTPCode_ELB_5XX_Count", dim, dimVal); m != nil {
		e.Metrics = append(e.Metrics, fmtMetric(m, ""))
	}
	return e
}

// GetLambdaDepsWithAPI returns dependencies for a Lambda function using the provided API.
func GetLambdaDepsWithAPI(ctx context.Context, api LambdaGetAPI, name string) []Dependency {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out, err := api.GetFunction(ctx, &lambda.GetFunctionInput{FunctionName: aws.String(name)})
	if err != nil {
		return nil
	}

	var deps []Dependency
	cfg := out.Configuration
	fnName := aws.ToString(cfg.FunctionName)

	if cfg.Environment != nil {
		for k, v := range cfg.Environment.Variables {
			if strings.Contains(v, "arn:aws:sqs:") {
				deps = append(deps, Dependency{fnName, extractName(v), "writes to SQS"})
			} else if strings.Contains(v, "arn:aws:sns:") {
				deps = append(deps, Dependency{fnName, extractName(v), "publishes to SNS"})
			} else if strings.Contains(v, "arn:aws:dynamodb:") {
				deps = append(deps, Dependency{fnName, extractName(v), "reads/writes DynamoDB"})
			} else if strings.Contains(v, "arn:aws:s3:") || strings.HasPrefix(v, "s3://") {
				deps = append(deps, Dependency{fnName, v, "accesses S3"})
			} else if strings.Contains(strings.ToUpper(k), "TABLE") && !strings.Contains(v, "arn:") {
				deps = append(deps, Dependency{fnName, v, "DynamoDB table"})
			} else if strings.Contains(strings.ToUpper(k), "BUCKET") {
				deps = append(deps, Dependency{fnName, v, "S3 bucket"})
			} else if strings.Contains(strings.ToUpper(k), "QUEUE") {
				deps = append(deps, Dependency{fnName, v, "SQS queue"})
			}
		}
	}

	mappings, err := api.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
		FunctionName: aws.String(name),
	})
	if err == nil {
		for _, m := range mappings.EventSourceMappings {
			src := aws.ToString(m.EventSourceArn)
			deps = append(deps, Dependency{extractName(src), fnName, "triggers"})
		}
	}

	if cfg.Role != nil {
		deps = append(deps, Dependency{fnName, extractName(aws.ToString(cfg.Role)), "assumes role"})
	}

	return deps
}
