package aws

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type SecurityFinding struct {
	Severity string // HIGH, MEDIUM, LOW
	Service  string
	Resource string
	Issue    string
}

func (c *Client) RunSecurityAudit(ctx context.Context) []SecurityFinding {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var findings []SecurityFinding
	ch := make(chan []SecurityFinding, 4)

	go func() { ch <- c.auditSecurityGroups(ctx) }()
	go func() { ch <- c.auditS3Buckets(ctx) }()
	go func() { ch <- c.auditIAMUsers(ctx) }()
	go func() { ch <- c.auditRootAccount(ctx) }()

	for i := 0; i < 4; i++ {
		findings = append(findings, <-ch...)
	}
	return findings
}

func (c *Client) auditSecurityGroups(ctx context.Context) []SecurityFinding {
	var findings []SecurityFinding
	svc := ec2.NewFromConfig(c.Config)
	out, err := svc.DescribeSecurityGroups(ctx, nil)
	if err != nil {
		slog.Debug("security audit: security groups", "error", err)
		return nil
	}
	for _, sg := range out.SecurityGroups {
		name := aws.ToString(sg.GroupName)
		for _, perm := range sg.IpPermissions {
			for _, r := range perm.IpRanges {
				cidr := aws.ToString(r.CidrIp)
				if cidr == "0.0.0.0/0" {
					port := "all"
					if perm.FromPort != nil {
						port = fmt.Sprintf("%d", *perm.FromPort)
					}
					sev := "MEDIUM"
					if port == "22" || port == "3389" || port == "all" {
						sev = "HIGH"
					}
					findings = append(findings, SecurityFinding{
						Severity: sev,
						Service:  "EC2",
						Resource: name,
						Issue:    fmt.Sprintf("Port %s open to 0.0.0.0/0", port),
					})
				}
			}
		}
	}
	return findings
}

func (c *Client) auditS3Buckets(ctx context.Context) []SecurityFinding {
	var findings []SecurityFinding
	svc := s3.NewFromConfig(c.Config)
	buckets, err := svc.ListBuckets(ctx, nil)
	if err != nil {
		slog.Debug("security audit: s3 buckets", "error", err)
		return nil
	}
	for _, b := range buckets.Buckets {
		name := aws.ToString(b.Name)
		// check public access block
		pub, err := svc.GetPublicAccessBlock(ctx, &s3.GetPublicAccessBlockInput{Bucket: b.Name})
		if err != nil {
			findings = append(findings, SecurityFinding{
				Severity: "HIGH",
				Service:  "S3",
				Resource: name,
				Issue:    "No public access block configured",
			})
			continue
		}
		cfg := pub.PublicAccessBlockConfiguration
		if !aws.ToBool(cfg.BlockPublicAcls) || !aws.ToBool(cfg.BlockPublicPolicy) {
			findings = append(findings, SecurityFinding{
				Severity: "HIGH",
				Service:  "S3",
				Resource: name,
				Issue:    "Public access not fully blocked",
			})
		}
		// check encryption
		_, err = svc.GetBucketEncryption(ctx, &s3.GetBucketEncryptionInput{Bucket: b.Name})
		if err != nil {
			findings = append(findings, SecurityFinding{
				Severity: "MEDIUM",
				Service:  "S3",
				Resource: name,
				Issue:    "No default encryption configured",
			})
		}
		// check versioning
		ver, err := svc.GetBucketVersioning(ctx, &s3.GetBucketVersioningInput{Bucket: b.Name})
		if err == nil && ver.Status != s3types.BucketVersioningStatusEnabled {
			findings = append(findings, SecurityFinding{
				Severity: "LOW",
				Service:  "S3",
				Resource: name,
				Issue:    "Versioning not enabled",
			})
		}
	}
	return findings
}

func (c *Client) auditIAMUsers(ctx context.Context) []SecurityFinding {
	var findings []SecurityFinding
	svc := iam.NewFromConfig(c.Config)
	users, err := svc.ListUsers(ctx, nil)
	if err != nil {
		slog.Debug("security audit: iam users", "error", err)
		return nil
	}
	for _, u := range users.Users {
		name := aws.ToString(u.UserName)
		// check MFA
		mfa, err := svc.ListMFADevices(ctx, &iam.ListMFADevicesInput{UserName: u.UserName})
		if err == nil && len(mfa.MFADevices) == 0 {
			findings = append(findings, SecurityFinding{
				Severity: "HIGH",
				Service:  "IAM",
				Resource: name,
				Issue:    "No MFA device configured",
			})
		}
		// check old access keys
		keys, err := svc.ListAccessKeys(ctx, &iam.ListAccessKeysInput{UserName: u.UserName})
		if err == nil {
			for _, k := range keys.AccessKeyMetadata {
				if k.CreateDate != nil && time.Since(*k.CreateDate) > 90*24*time.Hour {
					findings = append(findings, SecurityFinding{
						Severity: "MEDIUM",
						Service:  "IAM",
						Resource: name,
						Issue:    fmt.Sprintf("Access key %s older than 90 days", aws.ToString(k.AccessKeyId)),
					})
				}
			}
		}
	}
	return findings
}

func (c *Client) auditRootAccount(ctx context.Context) []SecurityFinding {
	var findings []SecurityFinding
	svc := iam.NewFromConfig(c.Config)
	summary, err := svc.GetAccountSummary(ctx, nil)
	if err != nil {
		slog.Debug("security audit: root account", "error", err)
		return nil
	}
	if v, ok := summary.SummaryMap["AccountMFAEnabled"]; ok && v == 0 {
		findings = append(findings, SecurityFinding{
			Severity: "HIGH",
			Service:  "IAM",
			Resource: "root",
			Issue:    "Root account MFA not enabled",
		})
	}
	return findings
}
