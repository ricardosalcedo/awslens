package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
)

// --- Mocks ---

type mockCloudWatchAPI struct {
	output *cloudwatch.GetMetricStatisticsOutput
	err    error
}

func (m *mockCloudWatchAPI) GetMetricStatistics(ctx context.Context, params *cloudwatch.GetMetricStatisticsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricStatisticsOutput, error) {
	return m.output, m.err
}

type mockCloudWatchLogsAPI struct {
	output *cloudwatchlogs.FilterLogEventsOutput
	err    error
}

func (m *mockCloudWatchLogsAPI) FilterLogEvents(ctx context.Context, params *cloudwatchlogs.FilterLogEventsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.FilterLogEventsOutput, error) {
	return m.output, m.err
}

type mockLambdaGetAPI struct {
	getFuncOutput *lambda.GetFunctionOutput
	getFuncErr    error
	listESMOutput *lambda.ListEventSourceMappingsOutput
	listESMErr    error
}

func (m *mockLambdaGetAPI) GetFunction(ctx context.Context, params *lambda.GetFunctionInput, optFns ...func(*lambda.Options)) (*lambda.GetFunctionOutput, error) {
	return m.getFuncOutput, m.getFuncErr
}

func (m *mockLambdaGetAPI) ListEventSourceMappings(ctx context.Context, params *lambda.ListEventSourceMappingsInput, optFns ...func(*lambda.Options)) (*lambda.ListEventSourceMappingsOutput, error) {
	return m.listESMOutput, m.listESMErr
}

// --- Tests ---

func TestGetMetricSummaryWithAPI_Success(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(45.5), Maximum: aws.Float64(92.1), Sum: aws.Float64(1000.0)},
			},
		},
	}
	m, err := GetMetricSummaryWithAPI(context.Background(), mock, "AWS/EC2", "CPUUtilization", "InstanceId", "i-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Name != "CPUUtilization" {
		t.Errorf("Name = %q, want CPUUtilization", m.Name)
	}
	if m.Avg != 45.5 {
		t.Errorf("Avg = %f, want 45.5", m.Avg)
	}
	if m.Max != 92.1 {
		t.Errorf("Max = %f, want 92.1", m.Max)
	}
	if m.Sum != 1000.0 {
		t.Errorf("Sum = %f, want 1000.0", m.Sum)
	}
}

func TestGetMetricSummaryWithAPI_NoDatapoints(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{Datapoints: []cwtypes.Datapoint{}},
	}
	m, err := GetMetricSummaryWithAPI(context.Background(), mock, "AWS/EC2", "CPUUtilization", "InstanceId", "i-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m != nil {
		t.Errorf("expected nil MetricSummary for empty datapoints, got %+v", m)
	}
}

func TestGetMetricSummaryWithAPI_Error(t *testing.T) {
	mock := &mockCloudWatchAPI{err: errors.New("throttled")}
	m, err := GetMetricSummaryWithAPI(context.Background(), mock, "AWS/EC2", "CPUUtilization", "InstanceId", "i-123")
	if err == nil {
		t.Fatal("expected error")
	}
	if m != nil {
		t.Errorf("expected nil MetricSummary on error, got %+v", m)
	}
}

func TestGetRecentErrorsWithAPI_Success(t *testing.T) {
	mock := &mockCloudWatchLogsAPI{
		output: &cloudwatchlogs.FilterLogEventsOutput{
			Events: []cwltypes.FilteredLogEvent{
				{Message: aws.String("ERROR: something failed")},
				{Message: aws.String("Exception in handler")},
			},
		},
	}
	lines, err := GetRecentErrorsWithAPI(context.Background(), mock, "/aws/lambda/myFunc", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "ERROR: something failed" {
		t.Errorf("lines[0] = %q", lines[0])
	}
}

func TestGetRecentErrorsWithAPI_Truncation(t *testing.T) {
	longMsg := ""
	for i := 0; i < 210; i++ {
		longMsg += "x"
	}
	mock := &mockCloudWatchLogsAPI{
		output: &cloudwatchlogs.FilterLogEventsOutput{
			Events: []cwltypes.FilteredLogEvent{{Message: aws.String(longMsg)}},
		},
	}
	lines, err := GetRecentErrorsWithAPI(context.Background(), mock, "/aws/lambda/myFunc", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines[0]) != 203 { // 200 + "..."
		t.Errorf("expected truncated to 203 chars, got %d", len(lines[0]))
	}
}

func TestGetRecentErrorsWithAPI_Error(t *testing.T) {
	mock := &mockCloudWatchLogsAPI{err: errors.New("access denied")}
	lines, err := GetRecentErrorsWithAPI(context.Background(), mock, "/aws/lambda/myFunc", 5)
	if err == nil {
		t.Fatal("expected error")
	}
	if lines != nil {
		t.Errorf("expected nil lines on error")
	}
}

func TestEnrichEC2WithAPI(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(25.0), Maximum: aws.Float64(80.0), Sum: aws.Float64(500.0)},
			},
		},
	}
	e := EnrichEC2WithAPI(context.Background(), mock, "i-abc123")
	if len(e.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(e.Metrics))
	}
}

func TestEnrichEC2WithAPI_NoData(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{Datapoints: []cwtypes.Datapoint{}},
	}
	e := EnrichEC2WithAPI(context.Background(), mock, "i-abc123")
	if len(e.Metrics) != 0 {
		t.Errorf("expected 0 metrics for no data, got %d", len(e.Metrics))
	}
}

func TestEnrichRDSWithAPI(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(10.0), Maximum: aws.Float64(50.0), Sum: aws.Float64(200.0)},
			},
		},
	}
	e := EnrichRDSWithAPI(context.Background(), mock, "mydb")
	if len(e.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(e.Metrics))
	}
}

func TestEnrichSQSWithAPI(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(100.0), Maximum: aws.Float64(500.0), Sum: aws.Float64(10000.0)},
			},
		},
	}
	e := EnrichSQSWithAPI(context.Background(), mock, "my-queue")
	if len(e.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(e.Metrics))
	}
}

func TestEnrichDynamoWithAPI(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(5.0), Maximum: aws.Float64(20.0), Sum: aws.Float64(100.0)},
			},
		},
	}
	e := EnrichDynamoWithAPI(context.Background(), mock, "my-table")
	if len(e.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(e.Metrics))
	}
}

func TestEnrichALBWithAPI(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(0.5), Maximum: aws.Float64(2.0), Sum: aws.Float64(1000.0)},
			},
		},
	}
	e := EnrichALBWithAPI(context.Background(), mock, "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-alb/abc123")
	if len(e.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(e.Metrics))
	}
}

func TestEnrichALBWithAPI_NoAppPrefix(t *testing.T) {
	mock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(1.0), Maximum: aws.Float64(3.0), Sum: aws.Float64(50.0)},
			},
		},
	}
	// ARN without "app/" prefix — dimVal should be the full ARN
	e := EnrichALBWithAPI(context.Background(), mock, "some-lb-id")
	if len(e.Metrics) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(e.Metrics))
	}
}

func TestEnrichLambdaWithAPI_Full(t *testing.T) {
	cwMock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(10.0), Maximum: aws.Float64(50.0), Sum: aws.Float64(100.0)},
			},
		},
	}
	logsMock := &mockCloudWatchLogsAPI{
		output: &cloudwatchlogs.FilterLogEventsOutput{
			Events: []cwltypes.FilteredLogEvent{
				{Message: aws.String("ERROR: timeout")},
			},
		},
	}
	lambdaMock := &mockLambdaGetAPI{
		getFuncOutput: &lambda.GetFunctionOutput{
			Configuration: &lambdatypes.FunctionConfiguration{
				FunctionName: aws.String("myFunc"),
				Role:         aws.String("arn:aws:iam::123:role/MyRole"),
				Environment: &lambdatypes.EnvironmentResponse{
					Variables: map[string]string{
						"TABLE_NAME": "my-table",
						"QUEUE_URL":  "arn:aws:sqs:us-east-1:123:my-queue",
					},
				},
			},
		},
		listESMOutput: &lambda.ListEventSourceMappingsOutput{
			EventSourceMappings: []lambdatypes.EventSourceMappingConfiguration{
				{EventSourceArn: aws.String("arn:aws:sqs:us-east-1:123:trigger-queue")},
			},
		},
	}

	e := EnrichLambdaWithAPI(context.Background(), cwMock, logsMock, lambdaMock, "myFunc")
	if len(e.Metrics) != 4 {
		t.Errorf("expected 4 metrics, got %d", len(e.Metrics))
	}
	if len(e.Errors) != 1 {
		t.Errorf("expected 1 error line, got %d", len(e.Errors))
	}
	if len(e.Deps) < 3 {
		t.Errorf("expected at least 3 deps (table, queue, role, trigger), got %d", len(e.Deps))
	}
}

func TestEnrichLambdaWithAPI_ZeroInvocations(t *testing.T) {
	cwMock := &mockCloudWatchAPI{
		output: &cloudwatch.GetMetricStatisticsOutput{
			Datapoints: []cwtypes.Datapoint{
				{Average: aws.Float64(0.0), Maximum: aws.Float64(0.0), Sum: aws.Float64(0.0)},
			},
		},
	}
	logsMock := &mockCloudWatchLogsAPI{
		output: &cloudwatchlogs.FilterLogEventsOutput{},
	}
	lambdaMock := &mockLambdaGetAPI{
		getFuncOutput: &lambda.GetFunctionOutput{
			Configuration: &lambdatypes.FunctionConfiguration{
				FunctionName: aws.String("deadFunc"),
			},
		},
		listESMOutput: &lambda.ListEventSourceMappingsOutput{},
	}

	e := EnrichLambdaWithAPI(context.Background(), cwMock, logsMock, lambdaMock, "deadFunc")
	if len(e.Extra) == 0 {
		t.Error("expected Extra note about zero invocations")
	}
}

func TestGetLambdaDepsWithAPI_Error(t *testing.T) {
	mock := &mockLambdaGetAPI{getFuncErr: errors.New("not found")}
	deps := GetLambdaDepsWithAPI(context.Background(), mock, "missing")
	if deps != nil {
		t.Errorf("expected nil deps on error, got %v", deps)
	}
}

func TestGetLambdaDepsWithAPI_EnvVars(t *testing.T) {
	mock := &mockLambdaGetAPI{
		getFuncOutput: &lambda.GetFunctionOutput{
			Configuration: &lambdatypes.FunctionConfiguration{
				FunctionName: aws.String("fn"),
				Role:         aws.String("arn:aws:iam::123:role/Exec"),
				Environment: &lambdatypes.EnvironmentResponse{
					Variables: map[string]string{
						"SNS_TOPIC":   "arn:aws:sns:us-east-1:123:my-topic",
						"DYNAMO_ARN":  "arn:aws:dynamodb:us-east-1:123:table/orders",
						"BUCKET_NAME": "my-bucket",
						"S3_PATH":     "s3://data-bucket/prefix",
					},
				},
			},
		},
		listESMOutput: &lambda.ListEventSourceMappingsOutput{},
	}
	deps := GetLambdaDepsWithAPI(context.Background(), mock, "fn")
	if len(deps) < 4 { // SNS, DynamoDB, S3 bucket, S3 path, role
		t.Errorf("expected at least 4 deps, got %d: %+v", len(deps), deps)
	}
}
