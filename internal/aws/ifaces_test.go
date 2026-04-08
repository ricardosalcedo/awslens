package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// ── EC2 mock ─────────────────────────────────────────────────────────────────

type mockEC2 struct {
	out *ec2.DescribeInstancesOutput
	err error
}

func (m *mockEC2) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	return m.out, m.err
}

// ── S3 mock ──────────────────────────────────────────────────────────────────

type mockS3 struct {
	listOut *s3.ListBucketsOutput
	listErr error
	locOut  map[string]*s3.GetBucketLocationOutput // keyed by bucket name
	locErr  error
}

func (m *mockS3) ListBuckets(ctx context.Context, params *s3.ListBucketsInput, optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error) {
	return m.listOut, m.listErr
}

func (m *mockS3) GetBucketLocation(ctx context.Context, params *s3.GetBucketLocationInput, optFns ...func(*s3.Options)) (*s3.GetBucketLocationOutput, error) {
	if m.locErr != nil {
		return nil, m.locErr
	}
	if out, ok := m.locOut[aws.ToString(params.Bucket)]; ok {
		return out, nil
	}
	return &s3.GetBucketLocationOutput{}, nil
}

// ── ListInstancesWithAPI tests ───────────────────────────────────────────────

func TestListInstancesWithAPI(t *testing.T) {
	launch := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		mock    *mockEC2
		region  string
		want    []Instance
		wantErr bool
	}{
		{
			name: "maps all fields correctly",
			mock: &mockEC2{out: &ec2.DescribeInstancesOutput{
				Reservations: []ec2types.Reservation{{
					Instances: []ec2types.Instance{{
						InstanceId:      aws.String("i-abc123"),
						InstanceType:    ec2types.InstanceTypeT3Micro,
						PublicIpAddress: aws.String("1.2.3.4"),
						LaunchTime:      aws.Time(launch),
						State:           &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
						Placement:       &ec2types.Placement{AvailabilityZone: aws.String("us-east-1a")},
						Tags: []ec2types.Tag{
							{Key: aws.String("Name"), Value: aws.String("web-server")},
							{Key: aws.String("Env"), Value: aws.String("prod")},
						},
					}},
				}},
			}},
			region: "us-east-1",
			want: []Instance{{
				ID: "i-abc123", Name: "web-server", State: "running",
				Type: "t3.micro", AZ: "us-east-1a", PublicIP: "1.2.3.4",
				LaunchTime: "2024-03-15", Region: "us-east-1",
			}},
		},
		{
			name: "no public IP",
			mock: &mockEC2{out: &ec2.DescribeInstancesOutput{
				Reservations: []ec2types.Reservation{{
					Instances: []ec2types.Instance{{
						InstanceId:   aws.String("i-priv"),
						InstanceType: ec2types.InstanceTypeM5Large,
						State:        &ec2types.InstanceState{Name: ec2types.InstanceStateNameStopped},
						Placement:    &ec2types.Placement{AvailabilityZone: aws.String("eu-west-1b")},
					}},
				}},
			}},
			region: "eu-west-1",
			want: []Instance{{
				ID: "i-priv", State: "stopped", Type: "m5.large",
				AZ: "eu-west-1b", Region: "eu-west-1",
			}},
		},
		{
			name:   "empty reservations",
			mock:   &mockEC2{out: &ec2.DescribeInstancesOutput{}},
			region: "us-west-2",
			want:   nil,
		},
		{
			name: "multiple reservations",
			mock: &mockEC2{out: &ec2.DescribeInstancesOutput{
				Reservations: []ec2types.Reservation{
					{Instances: []ec2types.Instance{{
						InstanceId: aws.String("i-1"), InstanceType: ec2types.InstanceTypeT3Micro,
						State: &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
						Placement: &ec2types.Placement{AvailabilityZone: aws.String("us-east-1a")},
					}}},
					{Instances: []ec2types.Instance{{
						InstanceId: aws.String("i-2"), InstanceType: ec2types.InstanceTypeT3Small,
						State: &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
						Placement: &ec2types.Placement{AvailabilityZone: aws.String("us-east-1b")},
					}}},
				},
			}},
			region: "us-east-1",
			want: []Instance{
				{ID: "i-1", State: "running", Type: "t3.micro", AZ: "us-east-1a", Region: "us-east-1"},
				{ID: "i-2", State: "running", Type: "t3.small", AZ: "us-east-1b", Region: "us-east-1"},
			},
		},
		{
			name:    "API error",
			mock:    &mockEC2{err: errors.New("AccessDenied")},
			region:  "us-east-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListInstancesWithAPI(context.Background(), tt.mock, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d instances, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("instance[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── ListBucketsWithAPI tests ─────────────────────────────────────────────────

func TestListBucketsWithAPI(t *testing.T) {
	created := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		mock    *mockS3
		want    []Bucket
		wantErr bool
	}{
		{
			name: "maps fields and resolves region",
			mock: &mockS3{
				listOut: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("my-bucket"), CreationDate: aws.Time(created)},
					},
				},
				locOut: map[string]*s3.GetBucketLocationOutput{
					"my-bucket": {LocationConstraint: s3types.BucketLocationConstraintEuWest1},
				},
			},
			want: []Bucket{{Name: "my-bucket", CreationDate: "2023-06-01", Region: "eu-west-1"}},
		},
		{
			name: "empty location defaults to us-east-1",
			mock: &mockS3{
				listOut: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("us-bucket"), CreationDate: aws.Time(created)},
					},
				},
				locOut: map[string]*s3.GetBucketLocationOutput{
					"us-bucket": {LocationConstraint: ""},
				},
			},
			want: []Bucket{{Name: "us-bucket", CreationDate: "2023-06-01", Region: "us-east-1"}},
		},
		{
			name: "GetBucketLocation error leaves region empty",
			mock: &mockS3{
				listOut: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("err-bucket")},
					},
				},
				locErr: errors.New("access denied"),
			},
			want: []Bucket{{Name: "err-bucket"}},
		},
		{
			name:   "no buckets",
			mock:   &mockS3{listOut: &s3.ListBucketsOutput{}},
			want:   nil,
		},
		{
			name:    "ListBuckets API error",
			mock:    &mockS3{listErr: errors.New("network error")},
			wantErr: true,
		},
		{
			name: "multiple buckets different regions",
			mock: &mockS3{
				listOut: &s3.ListBucketsOutput{
					Buckets: []s3types.Bucket{
						{Name: aws.String("b1"), CreationDate: aws.Time(created)},
						{Name: aws.String("b2"), CreationDate: aws.Time(created)},
					},
				},
				locOut: map[string]*s3.GetBucketLocationOutput{
					"b1": {LocationConstraint: s3types.BucketLocationConstraintApSoutheast1},
					"b2": {LocationConstraint: ""},
				},
			},
			want: []Bucket{
				{Name: "b1", CreationDate: "2023-06-01", Region: "ap-southeast-1"},
				{Name: "b2", CreationDate: "2023-06-01", Region: "us-east-1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListBucketsWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d buckets, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("bucket[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
