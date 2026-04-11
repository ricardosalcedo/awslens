package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cftypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// ── SNS mock ─────────────────────────────────────────────────────────────────

type mockSNS struct {
	out *sns.ListTopicsOutput
	err error
}

func (m *mockSNS) ListTopics(ctx context.Context, params *sns.ListTopicsInput, optFns ...func(*sns.Options)) (*sns.ListTopicsOutput, error) {
	return m.out, m.err
}

// ── CloudFormation mock ──────────────────────────────────────────────────────

type mockCF struct {
	out *cloudformation.DescribeStacksOutput
	err error
}

func (m *mockCF) DescribeStacks(ctx context.Context, params *cloudformation.DescribeStacksInput, optFns ...func(*cloudformation.Options)) (*cloudformation.DescribeStacksOutput, error) {
	return m.out, m.err
}

// ── SNS tests ────────────────────────────────────────────────────────────────

func TestListTopicsWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockSNS
		want    int
		wantErr bool
		check   func(t *testing.T, topics []Topic)
	}{
		{
			name: "single topic with ARN parsing",
			mock: &mockSNS{out: &sns.ListTopicsOutput{
				Topics: []snstypes.Topic{{TopicArn: aws.String("arn:aws:sns:us-east-1:123456789012:my-topic")}},
			}},
			want: 1,
			check: func(t *testing.T, topics []Topic) {
				if topics[0].ARN != "arn:aws:sns:us-east-1:123456789012:my-topic" {
					t.Errorf("ARN = %q", topics[0].ARN)
				}
				if topics[0].Name != "my-topic" {
					t.Errorf("Name = %q, want %q", topics[0].Name, "my-topic")
				}
			},
		},
		{
			name: "multiple topics",
			mock: &mockSNS{out: &sns.ListTopicsOutput{
				Topics: []snstypes.Topic{
					{TopicArn: aws.String("arn:aws:sns:us-east-1:123:t1")},
					{TopicArn: aws.String("arn:aws:sns:us-east-1:123:t2")},
					{TopicArn: aws.String("arn:aws:sns:us-east-1:123:t3")},
				},
			}},
			want: 3,
		},
		{
			name: "empty results",
			mock: &mockSNS{out: &sns.ListTopicsOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockSNS{err: errors.New("access denied")},
			wantErr: true,
		},
		{
			name: "nil TopicArn handled gracefully",
			mock: &mockSNS{out: &sns.ListTopicsOutput{
				Topics: []snstypes.Topic{{TopicArn: nil}},
			}},
			want: 1,
			check: func(t *testing.T, topics []Topic) {
				if topics[0].ARN != "" {
					t.Errorf("ARN = %q, want empty", topics[0].ARN)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topics, err := ListTopicsWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(topics) != tt.want {
				t.Fatalf("got %d topics, want %d", len(topics), tt.want)
			}
			if tt.check != nil {
				tt.check(t, topics)
			}
		})
	}
}

// ── CloudFormation tests ─────────────────────────────────────────────────────

func TestListStacksWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockCF
		region  string
		want    int
		wantErr bool
		check   func(t *testing.T, stacks []Stack)
	}{
		{
			name:   "full field mapping",
			region: "us-west-2",
			mock: &mockCF{out: &cloudformation.DescribeStacksOutput{
				Stacks: []cftypes.Stack{{
					StackName:   aws.String("my-stack"),
					StackStatus: cftypes.StackStatusCreateComplete,
					DriftInformation: &cftypes.StackDriftInformation{
						StackDriftStatus: cftypes.StackDriftStatusInSync,
					},
				}},
			}},
			want: 1,
			check: func(t *testing.T, stacks []Stack) {
				s := stacks[0]
				if s.Name != "my-stack" {
					t.Errorf("Name = %q, want %q", s.Name, "my-stack")
				}
				if s.Status != "CREATE_COMPLETE" {
					t.Errorf("Status = %q, want %q", s.Status, "CREATE_COMPLETE")
				}
				if s.Drift != "IN_SYNC" {
					t.Errorf("Drift = %q, want %q", s.Drift, "IN_SYNC")
				}
				if s.Region != "us-west-2" {
					t.Errorf("Region = %q, want %q", s.Region, "us-west-2")
				}
			},
		},
		{
			name:   "nil DriftInformation",
			region: "eu-west-1",
			mock: &mockCF{out: &cloudformation.DescribeStacksOutput{
				Stacks: []cftypes.Stack{{
					StackName:   aws.String("no-drift"),
					StackStatus: cftypes.StackStatusUpdateComplete,
				}},
			}},
			want: 1,
			check: func(t *testing.T, stacks []Stack) {
				if stacks[0].Drift != "" {
					t.Errorf("Drift = %q, want empty", stacks[0].Drift)
				}
			},
		},
		{
			name: "multiple stacks",
			mock: &mockCF{out: &cloudformation.DescribeStacksOutput{
				Stacks: []cftypes.Stack{
					{StackName: aws.String("s1"), StackStatus: cftypes.StackStatusCreateComplete},
					{StackName: aws.String("s2"), StackStatus: cftypes.StackStatusDeleteComplete},
				},
			}},
			want: 2,
		},
		{
			name: "empty results",
			mock: &mockCF{out: &cloudformation.DescribeStacksOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockCF{err: errors.New("throttled")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stacks, err := ListStacksWithAPI(context.Background(), tt.mock, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(stacks) != tt.want {
				t.Fatalf("got %d stacks, want %d", len(stacks), tt.want)
			}
			if tt.check != nil {
				tt.check(t, stacks)
			}
		})
	}
}
