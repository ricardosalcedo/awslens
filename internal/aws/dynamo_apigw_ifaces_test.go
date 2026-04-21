package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	agwtypes "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ── DynamoDB mock ────────────────────────────────────────────────────────────

type mockDynamoDB struct {
	listOut     *dynamodb.ListTablesOutput
	listErr     error
	describeOut map[string]*dynamodb.DescribeTableOutput
	describeErr error
}

func (m *mockDynamoDB) ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	return m.listOut, m.listErr
}

func (m *mockDynamoDB) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	if m.describeErr != nil {
		return nil, m.describeErr
	}
	if out, ok := m.describeOut[aws.ToString(params.TableName)]; ok {
		return out, nil
	}
	return nil, errors.New("table not found")
}

// ── API Gateway mock ─────────────────────────────────────────────────────────

type mockAPIGateway struct {
	out *apigateway.GetRestApisOutput
	err error
}

func (m *mockAPIGateway) GetRestApis(ctx context.Context, params *apigateway.GetRestApisInput, optFns ...func(*apigateway.Options)) (*apigateway.GetRestApisOutput, error) {
	return m.out, m.err
}

// ── DynamoDB tests ───────────────────────────────────────────────────────────

func TestListDynamoTablesWithAPI(t *testing.T) {
	tests := []struct {
		name    string
		mock    *mockDynamoDB
		want    int
		wantErr bool
		check   func(t *testing.T, tables []DynamoTable)
	}{
		{
			name: "full field mapping",
			mock: &mockDynamoDB{
				listOut: &dynamodb.ListTablesOutput{TableNames: []string{"users"}},
				describeOut: map[string]*dynamodb.DescribeTableOutput{
					"users": {Table: &ddbtypes.TableDescription{
						TableStatus:    ddbtypes.TableStatusActive,
						ItemCount:      aws.Int64(1000),
						TableSizeBytes: aws.Int64(50000),
						BillingModeSummary: &ddbtypes.BillingModeSummary{
							BillingMode: ddbtypes.BillingModePayPerRequest,
						},
						KeySchema: []ddbtypes.KeySchemaElement{
							{AttributeName: aws.String("pk"), KeyType: ddbtypes.KeyTypeHash},
							{AttributeName: aws.String("sk"), KeyType: ddbtypes.KeyTypeRange},
						},
					}},
				},
			},
			want: 1,
			check: func(t *testing.T, tables []DynamoTable) {
				tb := tables[0]
				if tb.Name != "users" {
					t.Errorf("Name = %q, want %q", tb.Name, "users")
				}
				if tb.Status != "ACTIVE" {
					t.Errorf("Status = %q, want %q", tb.Status, "ACTIVE")
				}
				if tb.ItemCount != 1000 {
					t.Errorf("ItemCount = %d, want 1000", tb.ItemCount)
				}
				if tb.SizeBytes != 50000 {
					t.Errorf("SizeBytes = %d, want 50000", tb.SizeBytes)
				}
				if tb.BillingMode != "PAY_PER_REQUEST" {
					t.Errorf("BillingMode = %q, want %q", tb.BillingMode, "PAY_PER_REQUEST")
				}
				if tb.PKName != "pk" {
					t.Errorf("PKName = %q, want %q", tb.PKName, "pk")
				}
				if tb.SKName != "sk" {
					t.Errorf("SKName = %q, want %q", tb.SKName, "sk")
				}
				if tb.Region != "us-east-1" {
					t.Errorf("Region = %q, want %q", tb.Region, "us-east-1")
				}
			},
		},
		{
			name: "empty results",
			mock: &mockDynamoDB{
				listOut: &dynamodb.ListTablesOutput{},
			},
			want: 0,
		},
		{
			name: "DescribeTable error falls back to name-only entry",
			mock: &mockDynamoDB{
				listOut:     &dynamodb.ListTablesOutput{TableNames: []string{"broken"}},
				describeErr: errors.New("access denied"),
			},
			want: 1,
			check: func(t *testing.T, tables []DynamoTable) {
				if tables[0].Name != "broken" {
					t.Errorf("Name = %q, want %q", tables[0].Name, "broken")
				}
				if tables[0].Status != "" {
					t.Errorf("Status = %q, want empty", tables[0].Status)
				}
			},
		},
		{
			name: "multiple tables",
			mock: &mockDynamoDB{
				listOut: &dynamodb.ListTablesOutput{TableNames: []string{"t1", "t2"}},
				describeOut: map[string]*dynamodb.DescribeTableOutput{
					"t1": {Table: &ddbtypes.TableDescription{TableStatus: ddbtypes.TableStatusActive}},
					"t2": {Table: &ddbtypes.TableDescription{TableStatus: ddbtypes.TableStatusCreating}},
				},
			},
			want: 2,
		},
		{
			name:    "ListTables API error",
			mock:    &mockDynamoDB{listErr: errors.New("network error")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables, err := ListDynamoTablesWithAPI(context.Background(), tt.mock, "us-east-1")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(tables) != tt.want {
				t.Fatalf("got %d tables, want %d", len(tables), tt.want)
			}
			if tt.check != nil {
				tt.check(t, tables)
			}
		})
	}
}

// ── API Gateway tests ────────────────────────────────────────────────────────

func TestListRestAPIsWithAPI(t *testing.T) {
	created := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockAPIGateway
		want    int
		wantErr bool
		check   func(t *testing.T, apis []RestAPI)
	}{
		{
			name: "full field mapping",
			mock: &mockAPIGateway{out: &apigateway.GetRestApisOutput{
				Items: []agwtypes.RestApi{{
					Id:          aws.String("abc123"),
					Name:        aws.String("my-api"),
					Description: aws.String("My API"),
					CreatedDate: &created,
				}},
			}},
			want: 1,
			check: func(t *testing.T, apis []RestAPI) {
				a := apis[0]
				if a.ID != "abc123" {
					t.Errorf("ID = %q, want %q", a.ID, "abc123")
				}
				if a.Name != "my-api" {
					t.Errorf("Name = %q, want %q", a.Name, "my-api")
				}
				if a.Description != "My API" {
					t.Errorf("Description = %q, want %q", a.Description, "My API")
				}
				if a.CreatedDate != "2025-06-15" {
					t.Errorf("CreatedDate = %q, want %q", a.CreatedDate, "2025-06-15")
				}
				if a.Region != "us-west-2" {
					t.Errorf("Region = %q, want %q", a.Region, "us-west-2")
				}
			},
		},
		{
			name: "nil CreatedDate",
			mock: &mockAPIGateway{out: &apigateway.GetRestApisOutput{
				Items: []agwtypes.RestApi{{
					Id:   aws.String("x"),
					Name: aws.String("no-date"),
				}},
			}},
			want: 1,
			check: func(t *testing.T, apis []RestAPI) {
				if apis[0].CreatedDate != "" {
					t.Errorf("CreatedDate = %q, want empty", apis[0].CreatedDate)
				}
			},
		},
		{
			name: "empty results",
			mock: &mockAPIGateway{out: &apigateway.GetRestApisOutput{}},
			want: 0,
		},
		{
			name: "multiple APIs",
			mock: &mockAPIGateway{out: &apigateway.GetRestApisOutput{
				Items: []agwtypes.RestApi{
					{Id: aws.String("a1"), Name: aws.String("api1")},
					{Id: aws.String("a2"), Name: aws.String("api2")},
					{Id: aws.String("a3"), Name: aws.String("api3")},
				},
			}},
			want: 3,
		},
		{
			name:    "API error",
			mock:    &mockAPIGateway{err: errors.New("throttled")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apis, err := ListRestAPIsWithAPI(context.Background(), tt.mock, "us-west-2")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(apis) != tt.want {
				t.Fatalf("got %d APIs, want %d", len(apis), tt.want)
			}
			if tt.check != nil {
				tt.check(t, apis)
			}
		})
	}
}
