package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdatypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// ── Lambda mock ──────────────────────────────────────────────────────────────

type mockLambda struct {
	pages []*lambda.ListFunctionsOutput // returned in order
	err   error
	calls int
}

func (m *mockLambda) ListFunctions(ctx context.Context, params *lambda.ListFunctionsInput, optFns ...func(*lambda.Options)) (*lambda.ListFunctionsOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.calls >= len(m.pages) {
		return &lambda.ListFunctionsOutput{}, nil
	}
	out := m.pages[m.calls]
	m.calls++
	return out, nil
}

// ── SQS mock ─────────────────────────────────────────────────────────────────

type mockSQS struct {
	listOut *sqs.ListQueuesOutput
	listErr error
	attrs   map[string]*sqs.GetQueueAttributesOutput // keyed by queue URL
	attrErr error
}

func (m *mockSQS) ListQueues(ctx context.Context, params *sqs.ListQueuesInput, optFns ...func(*sqs.Options)) (*sqs.ListQueuesOutput, error) {
	return m.listOut, m.listErr
}

func (m *mockSQS) GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error) {
	if m.attrErr != nil {
		return nil, m.attrErr
	}
	if out, ok := m.attrs[aws.ToString(params.QueueUrl)]; ok {
		return out, nil
	}
	return &sqs.GetQueueAttributesOutput{Attributes: map[string]string{}}, nil
}

// ── ListFunctionsWithAPI tests ───────────────────────────────────────────────

func TestListFunctionsWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockLambda
		region  string
		want    []Function
		wantErr bool
	}{
		{
			name: "maps all fields",
			mock: &mockLambda{pages: []*lambda.ListFunctionsOutput{{
				Functions: []lambdatypes.FunctionConfiguration{{
					FunctionName: aws.String("my-func"),
					Runtime:      lambdatypes.RuntimePython312,
					MemorySize:   aws.Int32(256),
					Timeout:      aws.Int32(30),
					LastModified: aws.String("2024-06-01T12:00:00.000+0000"),
					Handler:      aws.String("index.handler"),
					Role:         aws.String("arn:aws:iam::123:role/MyRole"),
				}},
			}}},
			region: "us-east-1",
			want: []Function{{
				Name: "my-func", Runtime: "python3.12", Memory: 256, Timeout: 30,
				LastModified: "2024-06-01 12:00", Handler: "index.handler",
				Role: "arn:aws:iam::123:role/MyRole", Region: "us-east-1",
			}},
		},
		{
			name: "pagination across two pages",
			mock: &mockLambda{pages: []*lambda.ListFunctionsOutput{
				{
					Functions: []lambdatypes.FunctionConfiguration{
						{FunctionName: aws.String("fn-1"), Runtime: lambdatypes.RuntimeNodejs20x, MemorySize: aws.Int32(128), Timeout: aws.Int32(3)},
					},
					NextMarker: aws.String("token"),
				},
				{
					Functions: []lambdatypes.FunctionConfiguration{
						{FunctionName: aws.String("fn-2"), Runtime: lambdatypes.RuntimeGo1x, MemorySize: aws.Int32(512), Timeout: aws.Int32(60)},
					},
				},
			}},
			region: "eu-west-1",
			want: []Function{
				{Name: "fn-1", Runtime: "nodejs20.x", Memory: 128, Timeout: 3, Region: "eu-west-1"},
				{Name: "fn-2", Runtime: "go1.x", Memory: 512, Timeout: 60, Region: "eu-west-1"},
			},
		},
		{
			name:   "empty result",
			mock:   &mockLambda{pages: []*lambda.ListFunctionsOutput{{}}},
			region: "us-west-2",
			want:   nil,
		},
		{
			name:    "API error",
			mock:    &mockLambda{err: errors.New("AccessDenied")},
			region:  "us-east-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListFunctionsWithAPI(context.Background(), tt.mock, tt.region)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d functions, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("function[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// ── ListQueuesWithAPI tests ──────────────────────────────────────────────────

func TestListQueuesWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockSQS
		want    []Queue
		wantErr bool
	}{
		{
			name: "maps fields and attributes",
			mock: &mockSQS{
				listOut: &sqs.ListQueuesOutput{
					QueueUrls: []string{"https://sqs.us-east-1.amazonaws.com/123/my-queue"},
				},
				attrs: map[string]*sqs.GetQueueAttributesOutput{
					"https://sqs.us-east-1.amazonaws.com/123/my-queue": {
						Attributes: map[string]string{
							"ApproximateNumberOfMessages":           "42",
							"ApproximateNumberOfMessagesNotVisible": "5",
						},
					},
				},
			},
			want: []Queue{{
				URL: "https://sqs.us-east-1.amazonaws.com/123/my-queue", Name: "my-queue",
				Messages: "42", MessagesInFlight: "5",
			}},
		},
		{
			name: "GetQueueAttributes error leaves fields empty",
			mock: &mockSQS{
				listOut: &sqs.ListQueuesOutput{
					QueueUrls: []string{"https://sqs.us-east-1.amazonaws.com/123/err-queue"},
				},
				attrErr: errors.New("access denied"),
			},
			want: []Queue{{
				URL: "https://sqs.us-east-1.amazonaws.com/123/err-queue", Name: "err-queue",
			}},
		},
		{
			name:    "no queues",
			mock:    &mockSQS{listOut: &sqs.ListQueuesOutput{}},
			want:    []Queue{},
			wantErr: false,
		},
		{
			name:    "ListQueues API error",
			mock:    &mockSQS{listErr: errors.New("network error")},
			wantErr: true,
		},
		{
			name: "multiple queues preserve order",
			mock: &mockSQS{
				listOut: &sqs.ListQueuesOutput{
					QueueUrls: []string{
						"https://sqs.us-east-1.amazonaws.com/123/q1",
						"https://sqs.us-east-1.amazonaws.com/123/q2",
						"https://sqs.us-east-1.amazonaws.com/123/q3",
					},
				},
				attrs: map[string]*sqs.GetQueueAttributesOutput{
					"https://sqs.us-east-1.amazonaws.com/123/q1": {Attributes: map[string]string{"ApproximateNumberOfMessages": "10"}},
					"https://sqs.us-east-1.amazonaws.com/123/q2": {Attributes: map[string]string{"ApproximateNumberOfMessages": "20"}},
					"https://sqs.us-east-1.amazonaws.com/123/q3": {Attributes: map[string]string{"ApproximateNumberOfMessages": "30"}},
				},
			},
			want: []Queue{
				{URL: "https://sqs.us-east-1.amazonaws.com/123/q1", Name: "q1", Messages: "10"},
				{URL: "https://sqs.us-east-1.amazonaws.com/123/q2", Name: "q2", Messages: "20"},
				{URL: "https://sqs.us-east-1.amazonaws.com/123/q3", Name: "q3", Messages: "30"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ListQueuesWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d queues, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("queue[%d] = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
