package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

// ── CloudWatch mock ──────────────────────────────────────────────────────────

type mockCW struct {
	out *cloudwatch.DescribeAlarmsOutput
	err error
}

func (m *mockCW) DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	return m.out, m.err
}

// ── SFN mock ─────────────────────────────────────────────────────────────────

type mockSFN struct {
	out *sfn.ListStateMachinesOutput
	err error
}

func (m *mockSFN) ListStateMachines(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
	return m.out, m.err
}

// ── CloudWatch tests ─────────────────────────────────────────────────────────

func TestListAlarmsWithAPI(t *testing.T) {
	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockCW
		region  string
		want    int
		wantErr bool
		check   func(t *testing.T, alarms []CWAlarm)
	}{
		{
			name:   "full field mapping",
			region: "us-east-1",
			mock: &mockCW{out: &cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []cwtypes.MetricAlarm{{
					AlarmName:             aws.String("high-cpu"),
					StateValue:            cwtypes.StateValueAlarm,
					MetricName:            aws.String("CPUUtilization"),
					Namespace:             aws.String("AWS/EC2"),
					Threshold:             aws.Float64(90.0),
					StateUpdatedTimestamp: &ts,
				}},
			}},
			want: 1,
			check: func(t *testing.T, alarms []CWAlarm) {
				a := alarms[0]
				if a.Name != "high-cpu" {
					t.Errorf("Name = %q, want %q", a.Name, "high-cpu")
				}
				if a.State != "ALARM" {
					t.Errorf("State = %q, want %q", a.State, "ALARM")
				}
				if a.Metric != "CPUUtilization" {
					t.Errorf("Metric = %q", a.Metric)
				}
				if a.Namespace != "AWS/EC2" {
					t.Errorf("Namespace = %q", a.Namespace)
				}
				if a.Threshold != "90.00" {
					t.Errorf("Threshold = %q, want %q", a.Threshold, "90.00")
				}
				if a.Updated != "2026-01-15 10:30" {
					t.Errorf("Updated = %q", a.Updated)
				}
				if a.Region != "us-east-1" {
					t.Errorf("Region = %q", a.Region)
				}
			},
		},
		{
			name:   "nil timestamp",
			region: "eu-west-1",
			mock: &mockCW{out: &cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []cwtypes.MetricAlarm{{
					AlarmName:  aws.String("no-ts"),
					StateValue: cwtypes.StateValueOk,
				}},
			}},
			want: 1,
			check: func(t *testing.T, alarms []CWAlarm) {
				if alarms[0].Updated != "" {
					t.Errorf("Updated = %q, want empty", alarms[0].Updated)
				}
			},
		},
		{
			name: "multiple alarms",
			mock: &mockCW{out: &cloudwatch.DescribeAlarmsOutput{
				MetricAlarms: []cwtypes.MetricAlarm{
					{AlarmName: aws.String("a1"), StateValue: cwtypes.StateValueOk},
					{AlarmName: aws.String("a2"), StateValue: cwtypes.StateValueAlarm},
					{AlarmName: aws.String("a3"), StateValue: cwtypes.StateValueInsufficientData},
				},
			}},
			want: 3,
		},
		{
			name: "empty results",
			mock: &mockCW{out: &cloudwatch.DescribeAlarmsOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockCW{err: errors.New("access denied")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarms, err := ListAlarmsWithAPI(context.Background(), tt.mock, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(alarms) != tt.want {
				t.Fatalf("got %d alarms, want %d", len(alarms), tt.want)
			}
			if tt.check != nil {
				tt.check(t, alarms)
			}
		})
	}
}

// ── Step Functions tests ─────────────────────────────────────────────────────

func TestListStateMachinesWithAPI(t *testing.T) {
	ts := time.Date(2026, 3, 20, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockSFN
		region  string
		want    int
		wantErr bool
		check   func(t *testing.T, sms []StateMachine)
	}{
		{
			name:   "full field mapping",
			region: "us-west-2",
			mock: &mockSFN{out: &sfn.ListStateMachinesOutput{
				StateMachines: []sfntypes.StateMachineListItem{{
					Name:            aws.String("order-processor"),
					StateMachineArn: aws.String("arn:aws:states:us-west-2:123456789012:stateMachine:order-processor"),
					Type:            sfntypes.StateMachineTypeStandard,
					CreationDate:    &ts,
				}},
			}},
			want: 1,
			check: func(t *testing.T, sms []StateMachine) {
				sm := sms[0]
				if sm.Name != "order-processor" {
					t.Errorf("Name = %q", sm.Name)
				}
				if sm.ARN != "arn:aws:states:us-west-2:123456789012:stateMachine:order-processor" {
					t.Errorf("ARN = %q", sm.ARN)
				}
				if sm.Type != "STANDARD" {
					t.Errorf("Type = %q, want %q", sm.Type, "STANDARD")
				}
				if sm.Created != "2026-03-20" {
					t.Errorf("Created = %q", sm.Created)
				}
				if sm.Region != "us-west-2" {
					t.Errorf("Region = %q", sm.Region)
				}
			},
		},
		{
			name:   "nil creation date",
			region: "ap-southeast-1",
			mock: &mockSFN{out: &sfn.ListStateMachinesOutput{
				StateMachines: []sfntypes.StateMachineListItem{{
					Name: aws.String("no-date"),
					Type: sfntypes.StateMachineTypeExpress,
				}},
			}},
			want: 1,
			check: func(t *testing.T, sms []StateMachine) {
				if sms[0].Created != "" {
					t.Errorf("Created = %q, want empty", sms[0].Created)
				}
				if sms[0].Type != "EXPRESS" {
					t.Errorf("Type = %q, want %q", sms[0].Type, "EXPRESS")
				}
			},
		},
		{
			name: "multiple state machines",
			mock: &mockSFN{out: &sfn.ListStateMachinesOutput{
				StateMachines: []sfntypes.StateMachineListItem{
					{Name: aws.String("sm1"), Type: sfntypes.StateMachineTypeStandard},
					{Name: aws.String("sm2"), Type: sfntypes.StateMachineTypeExpress},
				},
			}},
			want: 2,
		},
		{
			name: "empty results",
			mock: &mockSFN{out: &sfn.ListStateMachinesOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockSFN{err: errors.New("throttled")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sms, err := ListStateMachinesWithAPI(context.Background(), tt.mock, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(sms) != tt.want {
				t.Fatalf("got %d state machines, want %d", len(sms), tt.want)
			}
			if tt.check != nil {
				tt.check(t, sms)
			}
		})
	}
}
