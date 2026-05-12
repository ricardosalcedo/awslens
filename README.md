# ⬡ awslens

[![CI](https://github.com/ricardosalcedo/awslens/actions/workflows/ci.yml/badge.svg)](https://github.com/ricardosalcedo/awslens/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ricardosalcedo/awslens)](https://goreportcard.com/report/github.com/ricardosalcedo/awslens)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

A modern TUI for AWS — see everything, navigate fast.

Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

![awslens demo](docs/demo.gif)

## Features

- **28 AWS services** in one terminal dashboard
- **Multi-region scanning** across 12 regions in parallel
- **CRUD operations** — create, delete, edit resources inline
- **AI insights** — press `i` for Bedrock-powered analysis of any resource (Claude 3 Haiku)
- **Security audit** — scans security groups, S3 buckets, IAM users, and root account
- **Profile picker** with IAM role assumption and SSO support
- **Sensitive value masking** — SSM params and Lambda env vars masked by default
- **Search/filter** — press `/` to filter any resource list
- **Scrollable views** with keyboard navigation

## Supported Services

EC2 · Lambda · S3 · RDS · DynamoDB · API Gateway · ECS · ECR · Step Functions · Load Balancers · Route 53 · Secrets Manager · SSM Parameter Store · SQS · SNS · CloudWatch · CloudFormation · Cost Explorer · ElastiCache · OpenSearch · MSK · Glue · Athena · CodeCommit · CodePipeline · CodeBuild · EventBridge · WAF

## Install

### Homebrew (macOS/Linux)

```bash
brew install ricardosalcedo/tap/awslens
```

### Go

```bash
go install github.com/ricardosalcedo/awslens@latest
```

### From source

```bash
git clone https://github.com/ricardosalcedo/awslens.git
cd awslens
go build -o awslens .
```

### Binary releases

Download pre-built binaries for Linux, macOS, and Windows from the [Releases](https://github.com/ricardosalcedo/awslens/releases) page.

## Usage

```bash
awslens                        # interactive profile picker
awslens --profile admin        # skip picker, use specific profile
awslens --region us-west-2     # override region
awslens --months-back 3        # start costs view 3 months back
```

### Headless Export

Export service data to stdout without launching the TUI — useful for scripting, CI pipelines, and automation:

```bash
awslens --profile admin --output json                  # export all services as JSON
awslens --profile admin --output csv                   # export all services as CSV
awslens --profile admin --output json --service ec2    # export only EC2 instances
awslens --profile admin --output csv --service lambda  # export only Lambda functions
```

Available services: `ec2`, `lambda`, `s3`, `rds`, `dynamodb`, `apigateway`, `ecs`, `ecr`, `stepfunctions`, `elb`, `route53`, `secretsmanager`, `ssm`, `sqs`, `sns`, `cloudwatch`, `cloudformation`, `costs`, `elasticache`, `opensearch`, `msk`, `glue`, `athena`, `codecommit`, `codepipeline`, `codebuild`, `eventbridge`, `waf`.

## Key Bindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `Enter` | Drill down / select |
| `Esc` or `q` | Go back |
| `Ctrl+C` | Quit |
| `i` | AI insight (Bedrock) |
| `r` | Refresh |
| `/` | Filter list |
| `s` | Toggle sensitive value mask |
| `n` | New resource |
| `d` | Delete resource |
| `e` | Edit resource |
| `[` / `]` | Previous / next month (Costs view) |

## Architecture

```
awslens
├── cmd/            # CLI entry point (cobra)
├── internal/
│   ├── aws/        # AWS SDK clients, service fetchers, security audit, AI insights
│   └── tui/        # Bubble Tea views, CRUD forms, profile picker, export
├── main.go
└── .goreleaser.yml # Cross-platform release config
```

## Requirements

- Go 1.21+
- AWS credentials configured (`~/.aws/config` and `~/.aws/credentials`)
- For AI insights: Bedrock model access enabled for Claude 3 Haiku in your region

## Contributing

Contributions are welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
