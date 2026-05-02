package aws

import (
	"log/slog"
	"strings"
	"testing"
)

func TestEnrichEC2_LogsMetricErrors(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("enrich ec2: metric fetch failed", "instance", "i-123", "metric", "CPUUtilization", "error", "access denied")
	})
	for _, want := range []string{"enrich ec2", "i-123", "CPUUtilization", "access denied"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log, got: %s", want, logged)
		}
	}
}

func TestEnrichLambda_LogsMetricAndLogErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		keys []string
	}{
		{"metric", "enrich lambda: metric fetch failed", []string{"function", "myFunc", "metric", "Invocations"}},
		{"logs", "enrich lambda: log fetch failed", []string{"function", "myFunc", "logGroup", "/aws/lambda/myFunc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := make([]any, 0, len(tt.keys)+2)
			for _, k := range tt.keys {
				args = append(args, k)
			}
			args = append(args, "error", "timeout")
			logged := captureSlog(func() {
				slog.Debug(tt.msg, args...)
			})
			if !strings.Contains(logged, tt.msg) {
				t.Errorf("expected %q in debug log", tt.msg)
			}
			if !strings.Contains(logged, "timeout") {
				t.Errorf("expected error in debug log")
			}
		})
	}
}

func TestEnrichRDS_LogsMetricErrors(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("enrich rds: metric fetch failed", "db", "mydb", "metric", "CPUUtilization", "error", "throttled")
	})
	for _, want := range []string{"enrich rds", "mydb", "CPUUtilization", "throttled"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log, got: %s", want, logged)
		}
	}
}

func TestEnrichSQS_LogsMetricErrors(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("enrich sqs: metric fetch failed", "queue", "my-queue", "metric", "ApproximateNumberOfMessagesVisible", "error", "denied")
	})
	for _, want := range []string{"enrich sqs", "my-queue", "denied"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log, got: %s", want, logged)
		}
	}
}

func TestEnrichDynamo_LogsMetricErrors(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("enrich dynamo: metric fetch failed", "table", "my-table", "metric", "ThrottledRequests", "error", "timeout")
	})
	for _, want := range []string{"enrich dynamo", "my-table", "ThrottledRequests", "timeout"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log, got: %s", want, logged)
		}
	}
}

func TestEnrichALB_LogsMetricErrors(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("enrich alb: metric fetch failed", "lb", "arn:aws:elasticloadbalancing:us-east-1:123:loadbalancer/app/my-lb/abc", "metric", "RequestCount", "error", "access denied")
	})
	for _, want := range []string{"enrich alb", "my-lb", "RequestCount", "access denied"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log, got: %s", want, logged)
		}
	}
}

func TestGetLambdaDeps_LogsErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{"GetFunction", "GetLambdaDeps: GetFunction failed"},
		{"ListEventSourceMappings", "GetLambdaDeps: ListEventSourceMappings failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logged := captureSlog(func() {
				slog.Debug(tt.msg, "function", "my-func", "error", "not found")
			})
			if !strings.Contains(logged, tt.msg) {
				t.Errorf("expected %q in debug log", tt.msg)
			}
			if !strings.Contains(logged, "my-func") {
				t.Error("expected function name in debug log")
			}
		})
	}
}

func TestListDynamoTables_LogsDescribeTableError(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("ListDynamoTables: DescribeTable failed", "table", "my-table", "error", "access denied")
	})
	for _, want := range []string{"ListDynamoTables", "DescribeTable", "my-table", "access denied"} {
		if !strings.Contains(logged, want) {
			t.Errorf("expected %q in debug log, got: %s", want, logged)
		}
	}
}

func TestBedrock_LogsErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{"marshal", "bedrock: marshal request failed"},
		{"invoke", "bedrock: InvokeModel failed"},
		{"unmarshal", "bedrock: unmarshal response failed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logged := captureSlog(func() {
				slog.Debug(tt.msg, "error", "simulated")
			})
			if !strings.Contains(logged, tt.msg) {
				t.Errorf("expected %q in debug log", tt.msg)
			}
			if !strings.Contains(logged, "simulated") {
				t.Error("expected error in debug log")
			}
		})
	}
}
