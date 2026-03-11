module github.com/awslens/awslens

go 1.25.0

require (
	github.com/aws/aws-sdk-go-v2 v1.41.3
	github.com/aws/aws-sdk-go-v2/config v1.32.11
	github.com/aws/aws-sdk-go-v2/credentials v1.19.11
	github.com/aws/aws-sdk-go-v2/service/apigateway v1.38.6
	github.com/aws/aws-sdk-go-v2/service/athena v1.57.2
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.50.1
	github.com/aws/aws-sdk-go-v2/service/cloudformation v1.71.7
	github.com/aws/aws-sdk-go-v2/service/cloudwatch v1.55.1
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.64.0
	github.com/aws/aws-sdk-go-v2/service/codebuild v1.68.11
	github.com/aws/aws-sdk-go-v2/service/codecommit v1.33.10
	github.com/aws/aws-sdk-go-v2/service/codepipeline v1.46.19
	github.com/aws/aws-sdk-go-v2/service/costexplorer v1.63.4
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.56.1
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.294.0
	github.com/aws/aws-sdk-go-v2/service/ecr v1.55.4
	github.com/aws/aws-sdk-go-v2/service/ecs v1.73.1
	github.com/aws/aws-sdk-go-v2/service/elasticache v1.51.11
	github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2 v1.54.8
	github.com/aws/aws-sdk-go-v2/service/eventbridge v1.45.21
	github.com/aws/aws-sdk-go-v2/service/glue v1.137.2
	github.com/aws/aws-sdk-go-v2/service/iam v1.53.5
	github.com/aws/aws-sdk-go-v2/service/kafka v1.48.2
	github.com/aws/aws-sdk-go-v2/service/lambda v1.88.2
	github.com/aws/aws-sdk-go-v2/service/opensearch v1.59.0
	github.com/aws/aws-sdk-go-v2/service/rds v1.116.2
	github.com/aws/aws-sdk-go-v2/service/route53 v1.62.3
	github.com/aws/aws-sdk-go-v2/service/s3 v1.96.4
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.41.3
	github.com/aws/aws-sdk-go-v2/service/sfn v1.40.8
	github.com/aws/aws-sdk-go-v2/service/sns v1.39.13
	github.com/aws/aws-sdk-go-v2/service/sqs v1.42.23
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.2
	github.com/aws/aws-sdk-go-v2/service/wafv2 v1.71.1
	github.com/charmbracelet/bubbles v1.0.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/spf13/cobra v1.10.2
)

require (
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.6 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.5 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/checksum v1.9.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery v1.11.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/s3shared v1.19.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.8 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.5.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.3.8 // indirect
)
