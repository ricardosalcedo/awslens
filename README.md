# ⬡ awslens

A modern TUI for AWS — see everything, navigate fast.

Built with Go, [Bubble Tea](https://github.com/charmbracelet/bubbletea), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **28 AWS services** in one terminal dashboard
- **Multi-region scanning** across 12 regions in parallel
- **CRUD operations** — create, delete, edit resources inline
- **AI insights** — press `i` for Bedrock-powered analysis of any resource (Claude 3 Haiku)
- **Profile picker** with IAM role assumption and SSO support
- **Sensitive value masking** — SSM params and Lambda env vars masked by default
- **Search/filter** — press `/` to filter any resource list
- **Scrollable views** with keyboard navigation

## Install

```bash
go install github.com/awslens/awslens@latest
```

Or build from source:

```bash
git clone https://github.com/ricardosalcedo/awslens.git
cd awslens
go build -o awslens .
```

## Usage

```bash
awslens                        # interactive profile picker
awslens --profile admin        # skip picker, use specific profile
awslens --region us-west-2     # override region
```

## Key Bindings

| Key | Action |
|-----|--------|
| `↑/↓` or `j/k` | Navigate |
| `enter` | Drill down / select |
| `esc` or `q` | Go back |
| `ctrl+c` | Quit |
| `i` | AI insight (Bedrock) |
| `r` | Refresh |
| `/` | Filter list |
| `s` | Toggle sensitive value mask |
| `n` | New resource |
| `d` | Delete resource |
| `e` | Edit resource |

## Supported Services

EC2 · Lambda · S3 · RDS · DynamoDB · API Gateway · ECS · ECR · Step Functions · Load Balancers · Route53 · Secrets Manager · SSM Parameter Store · SQS · SNS · CloudWatch · CloudFormation · Cost Explorer · ElastiCache · OpenSearch · MSK · Glue · Athena · CodeCommit · CodePipeline · CodeBuild · EventBridge · WAF

## Requirements

- Go 1.21+
- AWS credentials configured (`~/.aws/config` and `~/.aws/credentials`)
- For AI insights: Bedrock model access enabled for Claude 3 Haiku

## License

MIT
