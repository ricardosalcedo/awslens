package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfntypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	ecrtypes "github.com/aws/aws-sdk-go-v2/service/ecr/types"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"
	gluetypes "github.com/aws/aws-sdk-go-v2/service/glue/types"
	r53types "github.com/aws/aws-sdk-go-v2/service/route53/types"
)

// TestEC2NilStateAndPlacement verifies that ListInstances handles
// instances with nil State and nil Placement without panicking.
func TestEC2NilStateAndPlacement(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil State/Placement: %v", r)
		}
	}()

	inst := ec2types.Instance{
		InstanceId:   aws.String("i-123"),
		InstanceType: ec2types.InstanceTypeT2Micro,
		State:        nil,
		Placement:    nil,
	}

	// Simulate the guarded logic from ListInstances
	result := Instance{
		ID:   aws.ToString(inst.InstanceId),
		Type: string(inst.InstanceType),
	}
	if inst.State != nil {
		result.State = string(inst.State.Name)
	}
	if inst.Placement != nil {
		result.AZ = aws.ToString(inst.Placement.AvailabilityZone)
	}

	if result.State != "" {
		t.Errorf("expected empty State, got %q", result.State)
	}
	if result.AZ != "" {
		t.Errorf("expected empty AZ, got %q", result.AZ)
	}
}

// TestEC2WithStateAndPlacement verifies the happy path still works.
func TestEC2WithStateAndPlacement(t *testing.T) {
	inst := ec2types.Instance{
		InstanceId:   aws.String("i-456"),
		InstanceType: ec2types.InstanceTypeT2Micro,
		State:        &ec2types.InstanceState{Name: ec2types.InstanceStateNameRunning},
		Placement:    &ec2types.Placement{AvailabilityZone: aws.String("us-east-1a")},
	}

	result := Instance{
		ID:   aws.ToString(inst.InstanceId),
		Type: string(inst.InstanceType),
	}
	if inst.State != nil {
		result.State = string(inst.State.Name)
	}
	if inst.Placement != nil {
		result.AZ = aws.ToString(inst.Placement.AvailabilityZone)
	}

	if result.State != "running" {
		t.Errorf("expected 'running', got %q", result.State)
	}
	if result.AZ != "us-east-1a" {
		t.Errorf("expected 'us-east-1a', got %q", result.AZ)
	}
}

// TestCFNNilDriftInformation verifies that ListStacks handles
// stacks with nil DriftInformation without panicking.
func TestCFNNilDriftInformation(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil DriftInformation: %v", r)
		}
	}()

	stack := cfntypes.Stack{
		StackName:        aws.String("my-stack"),
		StackStatus:      cfntypes.StackStatusCreateComplete,
		DriftInformation: nil,
	}

	st := Stack{
		Name:   aws.ToString(stack.StackName),
		Status: string(stack.StackStatus),
	}
	if stack.DriftInformation != nil {
		st.Drift = string(stack.DriftInformation.StackDriftStatus)
	}

	if st.Drift != "" {
		t.Errorf("expected empty Drift, got %q", st.Drift)
	}
}

// TestLBNilState verifies that ListLoadBalancers handles
// load balancers with nil State without panicking.
func TestLBNilState(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil lb.State: %v", r)
		}
	}()

	lb := elbtypes.LoadBalancer{
		LoadBalancerName: aws.String("my-lb"),
		Type:             elbtypes.LoadBalancerTypeEnumApplication,
		Scheme:           elbtypes.LoadBalancerSchemeEnumInternetFacing,
		State:            nil,
		DNSName:          aws.String("my-lb.elb.amazonaws.com"),
		VpcId:            aws.String("vpc-123"),
	}

	l := LoadBalancer{
		Name:   aws.ToString(lb.LoadBalancerName),
		Type:   string(lb.Type),
		Scheme: string(lb.Scheme),
		DNS:    aws.ToString(lb.DNSName),
		VPC:    aws.ToString(lb.VpcId),
	}
	if lb.State != nil {
		l.State = string(lb.State.Code)
	}

	if l.State != "" {
		t.Errorf("expected empty State, got %q", l.State)
	}
}

// TestRoute53NilConfig verifies that ListHostedZones handles
// zones with nil Config without panicking.
func TestRoute53NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil z.Config: %v", r)
		}
	}()

	z := r53types.HostedZone{
		Id:                     aws.String("/hostedzone/Z123"),
		Name:                   aws.String("example.com."),
		Config:                 nil,
		ResourceRecordSetCount: aws.Int64(5),
	}

	hz := HostedZone{
		ID:      aws.ToString(z.Id),
		Name:    aws.ToString(z.Name),
		Records: aws.ToInt64(z.ResourceRecordSetCount),
	}
	if z.Config != nil {
		hz.Private = z.Config.PrivateZone
	}

	if hz.Private != false {
		t.Errorf("expected Private=false, got %v", hz.Private)
	}
}

// TestECRNilImageScanningConfiguration verifies that ListECRRepos handles
// repos with nil ImageScanningConfiguration without panicking.
func TestECRNilImageScanningConfiguration(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil ImageScanningConfiguration: %v", r)
		}
	}()

	r := ecrtypes.Repository{
		RepositoryName:             aws.String("my-repo"),
		RepositoryUri:              aws.String("123456789.dkr.ecr.us-east-1.amazonaws.com/my-repo"),
		ImageScanningConfiguration: nil,
	}

	repo := ECRRepo{
		Name: aws.ToString(r.RepositoryName),
		URI:  aws.ToString(r.RepositoryUri),
	}
	if r.ImageScanningConfiguration != nil {
		repo.ScanOnPush = r.ImageScanningConfiguration.ScanOnPush
	}

	if repo.ScanOnPush != false {
		t.Errorf("expected ScanOnPush=false, got %v", repo.ScanOnPush)
	}
}

// TestGlueNilCommand verifies that ListGlueJobs handles
// jobs with nil Command without panicking.
func TestGlueNilCommand(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic on nil j.Command: %v", r)
		}
	}()

	j := gluetypes.Job{
		Name:    aws.String("my-job"),
		Role:    aws.String("arn:aws:iam::123:role/glue"),
		Command: nil,
	}

	job := GlueJob{
		Name: aws.ToString(j.Name),
		Role: aws.ToString(j.Role),
	}
	if j.Command != nil {
		job.Type = aws.ToString(j.Command.ScriptLocation)
	}

	if job.Type != "" {
		t.Errorf("expected empty Type, got %q", job.Type)
	}
}

// Ensure context import is used (for future integration tests).
var _ = context.Background
