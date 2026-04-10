package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	rdstypes "github.com/aws/aws-sdk-go-v2/service/rds/types"
)

// ── RDS mock ─────────────────────────────────────────────────────────────────

type mockRDS struct {
	out *rds.DescribeDBInstancesOutput
	err error
}

func (m *mockRDS) DescribeDBInstances(ctx context.Context, params *rds.DescribeDBInstancesInput, optFns ...func(*rds.Options)) (*rds.DescribeDBInstancesOutput, error) {
	return m.out, m.err
}

// ── ECS mock ─────────────────────────────────────────────────────────────────

type mockECS struct {
	listOut     *ecs.ListClustersOutput
	listErr     error
	describeOut *ecs.DescribeClustersOutput
	describeErr error
}

func (m *mockECS) ListClusters(ctx context.Context, params *ecs.ListClustersInput, optFns ...func(*ecs.Options)) (*ecs.ListClustersOutput, error) {
	return m.listOut, m.listErr
}

func (m *mockECS) DescribeClusters(ctx context.Context, params *ecs.DescribeClustersInput, optFns ...func(*ecs.Options)) (*ecs.DescribeClustersOutput, error) {
	return m.describeOut, m.describeErr
}

// ── RDS tests ────────────────────────────────────────────────────────────────

func TestListDBInstancesWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockRDS
		want    int
		wantErr bool
		check   func(t *testing.T, dbs []DBInstance)
	}{
		{
			name: "full field mapping",
			mock: &mockRDS{out: &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{{
					DBInstanceIdentifier: aws.String("mydb"),
					Engine:               aws.String("postgres"),
					EngineVersion:        aws.String("15.4"),
					DBInstanceStatus:     aws.String("available"),
					DBInstanceClass:      aws.String("db.t3.micro"),
					MultiAZ:              aws.Bool(true),
					Endpoint:             &rdstypes.Endpoint{Address: aws.String("mydb.example.com"), Port: aws.Int32(5432)},
				}},
			}},
			want: 1,
			check: func(t *testing.T, dbs []DBInstance) {
				d := dbs[0]
				if d.ID != "mydb" {
					t.Errorf("ID = %q, want %q", d.ID, "mydb")
				}
				if d.Engine != "postgres 15.4" {
					t.Errorf("Engine = %q, want %q", d.Engine, "postgres 15.4")
				}
				if d.Status != "available" {
					t.Errorf("Status = %q, want %q", d.Status, "available")
				}
				if d.Class != "db.t3.micro" {
					t.Errorf("Class = %q, want %q", d.Class, "db.t3.micro")
				}
				if !d.MultiAZ {
					t.Error("MultiAZ = false, want true")
				}
				if d.Endpoint != "mydb.example.com:5432" {
					t.Errorf("Endpoint = %q, want %q", d.Endpoint, "mydb.example.com:5432")
				}
			},
		},
		{
			name: "nil endpoint",
			mock: &mockRDS{out: &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{{
					DBInstanceIdentifier: aws.String("nodb"),
					Engine:               aws.String("mysql"),
					EngineVersion:        aws.String("8.0"),
					DBInstanceStatus:     aws.String("creating"),
					DBInstanceClass:      aws.String("db.r5.large"),
				}},
			}},
			want: 1,
			check: func(t *testing.T, dbs []DBInstance) {
				if dbs[0].Endpoint != "" {
					t.Errorf("Endpoint = %q, want empty", dbs[0].Endpoint)
				}
			},
		},
		{
			name: "empty results",
			mock: &mockRDS{out: &rds.DescribeDBInstancesOutput{}},
			want: 0,
		},
		{
			name: "multiple instances",
			mock: &mockRDS{out: &rds.DescribeDBInstancesOutput{
				DBInstances: []rdstypes.DBInstance{
					{DBInstanceIdentifier: aws.String("db1"), Engine: aws.String("postgres"), EngineVersion: aws.String("15")},
					{DBInstanceIdentifier: aws.String("db2"), Engine: aws.String("mysql"), EngineVersion: aws.String("8")},
					{DBInstanceIdentifier: aws.String("db3"), Engine: aws.String("aurora"), EngineVersion: aws.String("3")},
				},
			}},
			want: 3,
		},
		{
			name:    "API error",
			mock:    &mockRDS{err: errors.New("access denied")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbs, err := ListDBInstancesWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(dbs) != tt.want {
				t.Fatalf("got %d instances, want %d", len(dbs), tt.want)
			}
			if tt.check != nil {
				tt.check(t, dbs)
			}
		})
	}
}

// ── ECS tests ────────────────────────────────────────────────────────────────

func TestListClustersWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockECS
		want    int
		wantErr bool
		check   func(t *testing.T, clusters []Cluster)
	}{
		{
			name: "full field mapping",
			mock: &mockECS{
				listOut: &ecs.ListClustersOutput{ClusterArns: []string{"arn:aws:ecs:us-east-1:123:cluster/web"}},
				describeOut: &ecs.DescribeClustersOutput{Clusters: []ecstypes.Cluster{{
					ClusterName:         aws.String("web"),
					Status:              aws.String("ACTIVE"),
					RunningTasksCount:   5,
					PendingTasksCount:   2,
					ActiveServicesCount: 3,
				}}},
			},
			want: 1,
			check: func(t *testing.T, clusters []Cluster) {
				c := clusters[0]
				if c.Name != "web" {
					t.Errorf("Name = %q, want %q", c.Name, "web")
				}
				if c.Status != "ACTIVE" {
					t.Errorf("Status = %q, want %q", c.Status, "ACTIVE")
				}
				if c.RunningTasks != 5 {
					t.Errorf("RunningTasks = %d, want 5", c.RunningTasks)
				}
				if c.PendingTasks != 2 {
					t.Errorf("PendingTasks = %d, want 2", c.PendingTasks)
				}
				if c.ActiveServices != 3 {
					t.Errorf("ActiveServices = %d, want 3", c.ActiveServices)
				}
			},
		},
		{
			name: "empty cluster list returns nil",
			mock: &mockECS{
				listOut: &ecs.ListClustersOutput{ClusterArns: []string{}},
			},
			want: 0,
		},
		{
			name: "multiple clusters",
			mock: &mockECS{
				listOut: &ecs.ListClustersOutput{ClusterArns: []string{"arn1", "arn2"}},
				describeOut: &ecs.DescribeClustersOutput{Clusters: []ecstypes.Cluster{
					{ClusterName: aws.String("c1"), Status: aws.String("ACTIVE")},
					{ClusterName: aws.String("c2"), Status: aws.String("INACTIVE")},
				}},
			},
			want: 2,
		},
		{
			name:    "ListClusters API error",
			mock:    &mockECS{listErr: errors.New("network error")},
			wantErr: true,
		},
		{
			name: "DescribeClusters API error",
			mock: &mockECS{
				listOut:     &ecs.ListClustersOutput{ClusterArns: []string{"arn1"}},
				describeErr: errors.New("throttled"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusters, err := ListClustersWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(clusters) != tt.want {
				t.Fatalf("got %d clusters, want %d", len(clusters), tt.want)
			}
			if tt.check != nil {
				tt.check(t, clusters)
			}
		})
	}
}
