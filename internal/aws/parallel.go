package aws

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

// CodeCommitAPI is the subset of the CodeCommit SDK client used by this package.
type CodeCommitAPI interface {
	ListRepositories(ctx context.Context, params *codecommit.ListRepositoriesInput, optFns ...func(*codecommit.Options)) (*codecommit.ListRepositoriesOutput, error)
	GetRepository(ctx context.Context, params *codecommit.GetRepositoryInput, optFns ...func(*codecommit.Options)) (*codecommit.GetRepositoryOutput, error)
}

// DynamoDBListAPI is the subset of the DynamoDB SDK client used by ListDynamoTables.
type DynamoDBListAPI interface {
	ListTables(ctx context.Context, params *dynamodb.ListTablesInput, optFns ...func(*dynamodb.Options)) (*dynamodb.ListTablesOutput, error)
	DescribeTable(ctx context.Context, params *dynamodb.DescribeTableInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DescribeTableOutput, error)
}

// listCodeReposWithAPI lists CodeCommit repositories using the provided API.
// GetRepository calls are parallelized to prevent timeouts on accounts with many repos.
func listCodeReposWithAPI(ctx context.Context, api CodeCommitAPI, region string) ([]CodeRepo, error) {
	out, err := api.ListRepositories(ctx, nil)
	if err != nil {
		return nil, err
	}
	repos := make([]CodeRepo, len(out.Repositories))
	var wg sync.WaitGroup
	for i, r := range out.Repositories {
		repos[i] = CodeRepo{
			Name:   aws.ToString(r.RepositoryName),
			Region: region,
		}
		wg.Add(1)
		go func(idx int, repoName *string) {
			defer wg.Done()
			detail, err := api.GetRepository(ctx, &codecommit.GetRepositoryInput{RepositoryName: repoName})
			if err == nil && detail.RepositoryMetadata != nil {
				repos[idx].Description = aws.ToString(detail.RepositoryMetadata.RepositoryDescription)
				repos[idx].CloneURL = aws.ToString(detail.RepositoryMetadata.CloneUrlHttp)
				if detail.RepositoryMetadata.LastModifiedDate != nil {
					repos[idx].LastModified = detail.RepositoryMetadata.LastModifiedDate.Format("2006-01-02")
				}
			}
		}(i, r.RepositoryName)
	}
	wg.Wait()
	return repos, nil
}

// listDynamoTablesWithAPI lists DynamoDB tables using the provided API.
// DescribeTable calls are parallelized to prevent timeouts on accounts with many tables.
func listDynamoTablesWithAPI(ctx context.Context, api DynamoDBListAPI, region string) ([]DynamoTable, error) {
	// Collect all table names first (single-page for the testable version)
	out, err := api.ListTables(ctx, &dynamodb.ListTablesInput{})
	if err != nil {
		return nil, err
	}
	names := out.TableNames
	tables := make([]DynamoTable, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		tables[i] = DynamoTable{Name: name, Region: region}
		wg.Add(1)
		go func(idx int, tblName string) {
			defer wg.Done()
			desc, err := api.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tblName)})
			if err != nil {
				return
			}
			t := desc.Table
			tables[idx].Status = string(t.TableStatus)
			tables[idx].ItemCount = aws.ToInt64(t.ItemCount)
			tables[idx].SizeBytes = aws.ToInt64(t.TableSizeBytes)
			if t.BillingModeSummary != nil {
				tables[idx].BillingMode = string(t.BillingModeSummary.BillingMode)
			}
			for _, ks := range t.KeySchema {
				if ks.KeyType == "HASH" {
					tables[idx].PKName = aws.ToString(ks.AttributeName)
				} else {
					tables[idx].SKName = aws.ToString(ks.AttributeName)
				}
			}
		}(i, name)
	}
	wg.Wait()
	return tables, nil
}
