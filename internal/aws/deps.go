package aws

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type Dependency struct {
	From     string
	To       string
	Relation string // "triggers", "reads", "writes", "invokes"
}

// GetLambdaDeps returns dependencies for a Lambda function based on its config.
func (c *Client) GetLambdaDeps(ctx context.Context, name string) []Dependency {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	svc := lambda.NewFromConfig(c.Config)
	out, err := svc.GetFunction(ctx, &lambda.GetFunctionInput{FunctionName: aws.String(name)})
	if err != nil {
		return nil
	}

	var deps []Dependency
	cfg := out.Configuration
	fnName := aws.ToString(cfg.FunctionName)

	// env vars often reference other resources
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

	// event source mappings
	mappings, err := svc.ListEventSourceMappings(ctx, &lambda.ListEventSourceMappingsInput{
		FunctionName: aws.String(name),
	})
	if err == nil {
		for _, m := range mappings.EventSourceMappings {
			src := aws.ToString(m.EventSourceArn)
			deps = append(deps, Dependency{extractName(src), fnName, "triggers"})
		}
	}

	// role
	if cfg.Role != nil {
		deps = append(deps, Dependency{fnName, extractName(aws.ToString(cfg.Role)), "assumes role"})
	}

	return deps
}

func extractName(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) > 0 {
		last := parts[len(parts)-1]
		if strings.Contains(last, "/") {
			segs := strings.Split(last, "/")
			return segs[len(segs)-1]
		}
		return last
	}
	return arn
}

// FormatDeps renders dependencies as a text tree.
func FormatDeps(deps []Dependency) string {
	if len(deps) == 0 {
		return "  No dependencies detected"
	}
	var b strings.Builder
	for _, d := range deps {
		b.WriteString(fmt.Sprintf("  %s ─── %s ───▶ %s\n", d.From, d.Relation, d.To))
	}
	return b.String()
}
