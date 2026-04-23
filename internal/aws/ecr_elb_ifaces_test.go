package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
)

// ── ECR mock ─────────────────────────────────────────────────────────────────

type mockECR struct {
	out *ecr.DescribeRepositoriesOutput
	err error
}

func (m *mockECR) DescribeRepositories(ctx context.Context, params *ecr.DescribeRepositoriesInput, optFns ...func(*ecr.Options)) (*ecr.DescribeRepositoriesOutput, error) {
	return m.out, m.err
}

// ── ELB mock ─────────────────────────────────────────────────────────────────

type mockELB struct {
	out *elasticloadbalancingv2.DescribeLoadBalancersOutput
	err error
}

func (m *mockELB) DescribeLoadBalancers(ctx context.Context, params *elasticloadbalancingv2.DescribeLoadBalancersInput, optFns ...func(*elasticloadbalancingv2.Options)) (*elasticloadbalancingv2.DescribeLoadBalancersOutput, error) {
	return m.out, m.err
}

// ── ECR tests ────────────────────────────────────────────────────────────────

func TestListECRReposWithAPI(t *testing.T) {
	ts := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockECR
		region  string
		want    int
		wantErr bool
		check   func(t *testing.T, repos []ECRRepo)
	}{
		{
			name:   "full field mapping",
			region: "us-east-1",
			mock: &mockECR{out: &ecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{{
					RepositoryName: aws.String("my-app"),
					RepositoryUri:  aws.String("123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app"),
					CreatedAt:      &ts,
					ImageScanningConfiguration: &ecrtypes.ImageScanningConfiguration{ScanOnPush: true},
				}},
			}},
			want: 1,
			check: func(t *testing.T, repos []ECRRepo) {
				r := repos[0]
				if r.Name != "my-app" {
					t.Errorf("Name = %q, want %q", r.Name, "my-app")
				}
				if r.URI != "123456789012.dkr.ecr.us-east-1.amazonaws.com/my-app" {
					t.Errorf("URI = %q", r.URI)
				}
				if !r.ScanOnPush {
					t.Error("ScanOnPush = false, want true")
				}
				if r.Created != "2025-03-15" {
					t.Errorf("Created = %q, want %q", r.Created, "2025-03-15")
				}
				if r.Region != "us-east-1" {
					t.Errorf("Region = %q, want %q", r.Region, "us-east-1")
				}
			},
		},
		{
			name:   "nil ImageScanningConfiguration",
			region: "eu-west-1",
			mock: &mockECR{out: &ecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{{
					RepositoryName: aws.String("no-scan"),
				}},
			}},
			want: 1,
			check: func(t *testing.T, repos []ECRRepo) {
				if repos[0].ScanOnPush {
					t.Error("ScanOnPush = true, want false")
				}
			},
		},
		{
			name: "multiple repos",
			mock: &mockECR{out: &ecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{
					{RepositoryName: aws.String("r1")},
					{RepositoryName: aws.String("r2")},
					{RepositoryName: aws.String("r3")},
				},
			}},
			want: 3,
		},
		{
			name: "empty results",
			mock: &mockECR{out: &ecr.DescribeRepositoriesOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockECR{err: errors.New("access denied")},
			wantErr: true,
		},
		{
			name:   "nil CreatedAt",
			region: "ap-southeast-1",
			mock: &mockECR{out: &ecr.DescribeRepositoriesOutput{
				Repositories: []ecrtypes.Repository{{
					RepositoryName: aws.String("no-date"),
				}},
			}},
			want: 1,
			check: func(t *testing.T, repos []ECRRepo) {
				if repos[0].Created != "" {
					t.Errorf("Created = %q, want empty", repos[0].Created)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos, err := ListECRReposWithAPI(context.Background(), tt.mock, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(repos) != tt.want {
				t.Fatalf("got %d repos, want %d", len(repos), tt.want)
			}
			if tt.check != nil {
				tt.check(t, repos)
			}
		})
	}
}

// ── ELB tests ────────────────────────────────────────────────────────────────

func TestListLoadBalancersWithAPI(t *testing.T) {
	ts := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockELB
		want    int
		wantErr bool
		check   func(t *testing.T, lbs []LoadBalancer)
	}{
		{
			name: "full field mapping",
			mock: &mockELB{out: &elasticloadbalancingv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{{
					LoadBalancerName: aws.String("my-alb"),
					Type:             elbtypes.LoadBalancerTypeEnumApplication,
					Scheme:           elbtypes.LoadBalancerSchemeEnumInternetFacing,
					State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
					DNSName:          aws.String("my-alb-123.us-east-1.elb.amazonaws.com"),
					VpcId:            aws.String("vpc-abc123"),
					CreatedTime:      &ts,
				}},
			}},
			want: 1,
			check: func(t *testing.T, lbs []LoadBalancer) {
				lb := lbs[0]
				if lb.Name != "my-alb" {
					t.Errorf("Name = %q, want %q", lb.Name, "my-alb")
				}
				if lb.Type != "application" {
					t.Errorf("Type = %q, want %q", lb.Type, "application")
				}
				if lb.Scheme != "internet-facing" {
					t.Errorf("Scheme = %q, want %q", lb.Scheme, "internet-facing")
				}
				if lb.State != "active" {
					t.Errorf("State = %q, want %q", lb.State, "active")
				}
				if lb.DNS != "my-alb-123.us-east-1.elb.amazonaws.com" {
					t.Errorf("DNS = %q", lb.DNS)
				}
				if lb.VPC != "vpc-abc123" {
					t.Errorf("VPC = %q", lb.VPC)
				}
				if lb.Created != "2025-06-01" {
					t.Errorf("Created = %q, want %q", lb.Created, "2025-06-01")
				}
			},
		},
		{
			name: "nil State handled gracefully",
			mock: &mockELB{out: &elasticloadbalancingv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{{
					LoadBalancerName: aws.String("no-state"),
				}},
			}},
			want: 1,
			check: func(t *testing.T, lbs []LoadBalancer) {
				if lbs[0].State != "" {
					t.Errorf("State = %q, want empty", lbs[0].State)
				}
			},
		},
		{
			name: "nil CreatedTime",
			mock: &mockELB{out: &elasticloadbalancingv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{{
					LoadBalancerName: aws.String("no-time"),
					State:            &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive},
				}},
			}},
			want: 1,
			check: func(t *testing.T, lbs []LoadBalancer) {
				if lbs[0].Created != "" {
					t.Errorf("Created = %q, want empty", lbs[0].Created)
				}
			},
		},
		{
			name: "multiple load balancers",
			mock: &mockELB{out: &elasticloadbalancingv2.DescribeLoadBalancersOutput{
				LoadBalancers: []elbtypes.LoadBalancer{
					{LoadBalancerName: aws.String("lb1"), State: &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive}},
					{LoadBalancerName: aws.String("lb2"), State: &elbtypes.LoadBalancerState{Code: elbtypes.LoadBalancerStateEnumActive}},
				},
			}},
			want: 2,
		},
		{
			name: "empty results",
			mock: &mockELB{out: &elasticloadbalancingv2.DescribeLoadBalancersOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockELB{err: errors.New("throttled")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lbs, err := ListLoadBalancersWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(lbs) != tt.want {
				t.Fatalf("got %d LBs, want %d", len(lbs), tt.want)
			}
			if tt.check != nil {
				tt.check(t, lbs)
			}
		})
	}
}
