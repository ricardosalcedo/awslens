package aws

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	cctypes "github.com/aws/aws-sdk-go-v2/service/codecommit/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// ── CodeCommit mocks ─────────────────────────────────────────────────────────

type mockCodeCommitAPI struct {
	repos       []cctypes.RepositoryNameIdPair
	metadata    map[string]*codecommit.GetRepositoryOutput
	listErr     error
	getErr      error
	getConcur   atomic.Int32 // tracks concurrent GetRepository calls
	maxConcur   atomic.Int32
}

func (m *mockCodeCommitAPI) ListRepositories(ctx context.Context, params *codecommit.ListRepositoriesInput, optFns ...func(*codecommit.Options)) (*codecommit.ListRepositoriesOutput, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &codecommit.ListRepositoriesOutput{Repositories: m.repos}, nil
}

func (m *mockCodeCommitAPI) GetRepository(ctx context.Context, params *codecommit.GetRepositoryInput, optFns ...func(*codecommit.Options)) (*codecommit.GetRepositoryOutput, error) {
	cur := m.getConcur.Add(1)
	for {
		max := m.maxConcur.Load()
		if cur > max {
			if m.maxConcur.CompareAndSwap(max, cur) {
				break
			}
		} else {
			break
		}
	}
	// simulate network latency to allow concurrency to build up
	time.Sleep(10 * time.Millisecond)
	m.getConcur.Add(-1)

	if m.getErr != nil {
		return nil, m.getErr
	}
	name := aws.ToString(params.RepositoryName)
	if out, ok := m.metadata[name]; ok {
		return out, nil
	}
	return &codecommit.GetRepositoryOutput{}, nil
}

// ── DynamoDB mocks ───────────────────────────────────────────────────────────

type mockDynamoDBAPI struct {
	tableNames  []string
	descriptions map[string]*dynamodb.DescribeTableOutput
	listErr     error
	descErr     error
	descConcur  atomic.Int32
	maxConcur   atomic.Int32
}

func (m *mockDynamoDBAPI) ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return &dynamodb.ListTablesOutput{TableNames: m.tableNames}, nil
}

func (m *mockDynamoDBAPI) DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error) {
	cur := m.descConcur.Add(1)
	for {
		max := m.maxConcur.Load()
		if cur > max {
			if m.maxConcur.CompareAndSwap(max, cur) {
				break
			}
		} else {
			break
		}
	}
	time.Sleep(10 * time.Millisecond)
	m.descConcur.Add(-1)

	if m.descErr != nil {
		return nil, m.descErr
	}
	name := aws.ToString(params.TableName)
	if out, ok := m.descriptions[name]; ok {
		return out, nil
	}
	return nil, fmt.Errorf("table not found: %s", name)
}

// ── CodeCommit tests ─────────────────────────────────────────────────────────

func TestListCodeRepos_Parallel(t *testing.T) {
	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	mock := &mockCodeCommitAPI{
		repos: []cctypes.RepositoryNameIdPair{
			{RepositoryName: aws.String("repo-a")},
			{RepositoryName: aws.String("repo-b")},
			{RepositoryName: aws.String("repo-c")},
		},
		metadata: map[string]*codecommit.GetRepositoryOutput{
			"repo-a": {RepositoryMetadata: &cctypes.RepositoryMetadata{
				RepositoryDescription: aws.String("First repo"),
				CloneUrlHttp:          aws.String("https://git.example.com/repo-a"),
				LastModifiedDate:      &now,
			}},
			"repo-b": {RepositoryMetadata: &cctypes.RepositoryMetadata{
				RepositoryDescription: aws.String("Second repo"),
				CloneUrlHttp:          aws.String("https://git.example.com/repo-b"),
				LastModifiedDate:      &now,
			}},
			"repo-c": {RepositoryMetadata: &cctypes.RepositoryMetadata{
				RepositoryDescription: aws.String("Third repo"),
				CloneUrlHttp:          aws.String("https://git.example.com/repo-c"),
				LastModifiedDate:      &now,
			}},
		},
	}

	repos, err := listCodeReposWithAPI(context.Background(), mock, "us-east-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(repos))
	}
	// verify fields mapped correctly
	if repos[0].Name != "repo-a" || repos[0].Description != "First repo" {
		t.Errorf("repo-a: got name=%q desc=%q", repos[0].Name, repos[0].Description)
	}
	if repos[1].CloneURL != "https://git.example.com/repo-b" {
		t.Errorf("repo-b: got cloneURL=%q", repos[1].CloneURL)
	}
	if repos[2].LastModified != "2025-01-15" {
		t.Errorf("repo-c: got lastModified=%q", repos[2].LastModified)
	}
	// verify calls were concurrent (max concurrency > 1 with 3 repos)
	if mock.maxConcur.Load() < 2 {
		t.Errorf("expected concurrent GetRepository calls, max concurrency was %d", mock.maxConcur.Load())
	}
}

func TestListCodeRepos_ListError(t *testing.T) {
	mock := &mockCodeCommitAPI{listErr: fmt.Errorf("access denied")}
	_, err := listCodeReposWithAPI(context.Background(), mock, "us-east-1")
	if err == nil {
		t.Fatal("expected error from ListRepositories")
	}
}

func TestListCodeRepos_GetRepoError(t *testing.T) {
	mock := &mockCodeCommitAPI{
		repos: []cctypes.RepositoryNameIdPair{
			{RepositoryName: aws.String("repo-x")},
		},
		metadata: map[string]*codecommit.GetRepositoryOutput{},
		getErr:   fmt.Errorf("throttled"),
	}
	repos, err := listCodeReposWithAPI(context.Background(), mock, "us-west-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should still return the repo with basic info
	if len(repos) != 1 || repos[0].Name != "repo-x" || repos[0].Region != "us-west-2" {
		t.Errorf("expected basic repo info, got %+v", repos)
	}
}

func TestListCodeRepos_Empty(t *testing.T) {
	mock := &mockCodeCommitAPI{repos: nil}
	repos, err := listCodeReposWithAPI(context.Background(), mock, "eu-west-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 0 {
		t.Errorf("expected 0 repos, got %d", len(repos))
	}
}

// ── DynamoDB tests ───────────────────────────────────────────────────────────

func TestListDynamoTables_Parallel(t *testing.T) {
	mock := &mockDynamoDBAPI{
		tableNames: []string{"orders", "users", "sessions"},
		descriptions: map[string]*dynamodb.DescribeTableOutput{
			"orders": {Table: &dtypes.TableDescription{
				TableStatus: dtypes.TableStatusActive,
				ItemCount:   aws.Int64(1000),
				TableSizeBytes: aws.Int64(50000),
				KeySchema: []dtypes.KeySchemaElement{
					{AttributeName: aws.String("orderId"), KeyType: dtypes.KeyTypeHash},
					{AttributeName: aws.String("timestamp"), KeyType: dtypes.KeyTypeRange},
				},
			}},
			"users": {Table: &dtypes.TableDescription{
				TableStatus: dtypes.TableStatusActive,
				ItemCount:   aws.Int64(500),
				TableSizeBytes: aws.Int64(25000),
				BillingModeSummary: &dtypes.BillingModeSummary{BillingMode: dtypes.BillingModePayPerRequest},
				KeySchema: []dtypes.KeySchemaElement{
					{AttributeName: aws.String("userId"), KeyType: dtypes.KeyTypeHash},
				},
			}},
			"sessions": {Table: &dtypes.TableDescription{
				TableStatus: dtypes.TableStatusActive,
				ItemCount:   aws.Int64(200),
				TableSizeBytes: aws.Int64(10000),
				KeySchema: []dtypes.KeySchemaElement{
					{AttributeName: aws.String("sessionId"), KeyType: dtypes.KeyTypeHash},
				},
			}},
		},
	}

	tables, err := listDynamoTablesWithAPI(context.Background(), mock, "us-east-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 3 {
		t.Fatalf("expected 3 tables, got %d", len(tables))
	}
	// verify order preserved
	if tables[0].Name != "orders" || tables[1].Name != "users" || tables[2].Name != "sessions" {
		t.Errorf("order not preserved: %v", []string{tables[0].Name, tables[1].Name, tables[2].Name})
	}
	// verify fields
	if tables[0].PKName != "orderId" || tables[0].SKName != "timestamp" {
		t.Errorf("orders keys: pk=%q sk=%q", tables[0].PKName, tables[0].SKName)
	}
	if tables[0].ItemCount != 1000 {
		t.Errorf("orders itemCount=%d, want 1000", tables[0].ItemCount)
	}
	if tables[1].BillingMode != "PAY_PER_REQUEST" {
		t.Errorf("users billingMode=%q", tables[1].BillingMode)
	}
	// verify concurrency
	if mock.maxConcur.Load() < 2 {
		t.Errorf("expected concurrent DescribeTable calls, max concurrency was %d", mock.maxConcur.Load())
	}
}

func TestListDynamoTables_ListError(t *testing.T) {
	mock := &mockDynamoDBAPI{listErr: fmt.Errorf("access denied")}
	_, err := listDynamoTablesWithAPI(context.Background(), mock, "us-east-1")
	if err == nil {
		t.Fatal("expected error from ListTables")
	}
}

func TestListDynamoTables_DescribeError(t *testing.T) {
	mock := &mockDynamoDBAPI{
		tableNames:   []string{"broken-table"},
		descriptions: map[string]*dynamodb.DescribeTableOutput{},
		descErr:      fmt.Errorf("throttled"),
	}
	tables, err := listDynamoTablesWithAPI(context.Background(), mock, "us-west-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// should still return the table with basic info
	if len(tables) != 1 || tables[0].Name != "broken-table" || tables[0].Region != "us-west-2" {
		t.Errorf("expected basic table info, got %+v", tables)
	}
	// status should be empty since DescribeTable failed
	if tables[0].Status != "" {
		t.Errorf("expected empty status on error, got %q", tables[0].Status)
	}
}

func TestListDynamoTables_Empty(t *testing.T) {
	mock := &mockDynamoDBAPI{tableNames: nil}
	tables, err := listDynamoTablesWithAPI(context.Background(), mock, "eu-west-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tables) != 0 {
		t.Errorf("expected 0 tables, got %d", len(tables))
	}
}
