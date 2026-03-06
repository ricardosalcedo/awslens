package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// ServiceAccess holds the probe result for each service.
type ServiceAccess struct {
	Name    string
	Allowed bool
	Error   string
}

// ProbeAccess checks which services the current credentials can access.
// It runs all probes in parallel for speed.
func (c *Client) ProbeAccess(ctx context.Context) map[string]bool {
	probes := map[string]func() error{
		"EC2": func() error {
			_, err := ec2.NewFromConfig(c.Config).DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
			return err
		},
		"Lambda": func() error {
			_, err := lambda.NewFromConfig(c.Config).ListFunctions(ctx, &lambda.ListFunctionsInput{})
			return err
		},
		"S3": func() error {
			_, err := s3.NewFromConfig(c.Config).ListBuckets(ctx, &s3.ListBucketsInput{})
			return err
		},
		"RDS": func() error {
			_, err := rds.NewFromConfig(c.Config).DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
			return err
		},
		"DynamoDB": func() error {
			_, err := dynamodb.NewFromConfig(c.Config).ListTables(ctx, &dynamodb.ListTablesInput{})
			return err
		},
		"API Gateway": func() error {
			_, err := apigateway.NewFromConfig(c.Config).GetRestApis(ctx, &apigateway.GetRestApisInput{})
			return err
		},
		"ECS": func() error {
			_, err := ecs.NewFromConfig(c.Config).ListClusters(ctx, &ecs.ListClustersInput{})
			return err
		},
		"ECR": func() error {
			_, err := ecr.NewFromConfig(c.Config).DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
			return err
		},
		"Step Functions": func() error {
			_, err := sfn.NewFromConfig(c.Config).ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
			return err
		},
		"Load Balancers": func() error {
			_, err := elasticloadbalancingv2.NewFromConfig(c.Config).DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
			return err
		},
		"Route53": func() error {
			_, err := route53.NewFromConfig(c.Config).ListHostedZones(ctx, &route53.ListHostedZonesInput{})
			return err
		},
		"Secrets Manager": func() error {
			_, err := secretsmanager.NewFromConfig(c.Config).ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
			return err
		},
		"SSM": func() error {
			_, err := ssm.NewFromConfig(c.Config).DescribeParameters(ctx, &ssm.DescribeParametersInput{})
			return err
		},
		"SQS": func() error {
			_, err := sqs.NewFromConfig(c.Config).ListQueues(ctx, &sqs.ListQueuesInput{})
			return err
		},
		"SNS": func() error {
			_, err := sns.NewFromConfig(c.Config).ListTopics(ctx, &sns.ListTopicsInput{})
			return err
		},
		"CloudWatch": func() error {
			_, err := cloudwatch.NewFromConfig(c.Config).DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{})
			return err
		},
		"CloudFormation": func() error {
			_, err := cloudformation.NewFromConfig(c.Config).DescribeStacks(ctx, &cloudformation.DescribeStacksInput{})
			return err
		},
		// Costs always shown — no cheap probe for Cost Explorer
		"Costs": func() error { return nil },
		"ElastiCache": func() error {
			_, err := elasticache.NewFromConfig(c.Config).DescribeCacheClusters(ctx, nil)
			return err
		},
		"OpenSearch": func() error {
			_, err := opensearch.NewFromConfig(c.Config).ListDomainNames(ctx, nil)
			return err
		},
		"MSK": func() error {
			_, err := kafka.NewFromConfig(c.Config).ListClusters(ctx, nil)
			return err
		},
		"Glue": func() error {
			_, err := glue.NewFromConfig(c.Config).GetDatabases(ctx, nil)
			return err
		},
		"Athena": func() error {
			_, err := athena.NewFromConfig(c.Config).ListWorkGroups(ctx, nil)
			return err
		},
		"CodeCommit": func() error {
			_, err := codecommit.NewFromConfig(c.Config).ListRepositories(ctx, nil)
			return err
		},
		"CodePipeline": func() error {
			_, err := codepipeline.NewFromConfig(c.Config).ListPipelines(ctx, nil)
			return err
		},
		"CodeBuild": func() error {
			_, err := codebuild.NewFromConfig(c.Config).ListProjects(ctx, nil)
			return err
		},
		"EventBridge": func() error {
			_, err := eventbridge.NewFromConfig(c.Config).ListRules(ctx, nil)
			return err
		},
		"WAF": func() error {
			_, err := wafv2.NewFromConfig(c.Config).ListWebACLs(ctx, &wafv2.ListWebACLsInput{Scope: waftypes.ScopeRegional})
			return err
		},
	}

	results := make(map[string]bool, len(probes))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for name, probe := range probes {
		name, probe := name, probe
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := probe()
			allowed := err == nil || isNotFoundErr(err)
			mu.Lock()
			results[name] = allowed
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}

// isNotFoundErr returns true for "no resources" responses that aren't auth errors.
func isNotFoundErr(err error) bool {
	if err == nil {
		return true
	}
	s := err.Error()
	// these mean "allowed but empty", not "denied"
	for _, ok := range []string{"NoSuchEntity", "NotFoundException", "ResourceNotFoundException"} {
		if contains(s, ok) {
			return true
		}
	}
	return false
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
