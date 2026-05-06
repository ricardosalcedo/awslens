package aws

import (
	"context"
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	kafkatypes "github.com/aws/aws-sdk-go-v2/service/kafka/types"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	ostypes "github.com/aws/aws-sdk-go-v2/service/opensearch/types"
)

// ── MSK interface and mock ───────────────────────────────────────────────────

type MSKAPI interface {
	ListClusters(ctx context.Context, params *kafka.ListClustersInput, optFns ...func(*kafka.Options)) (*kafka.ListClustersOutput, error)
}

type mockMSK struct {
	out *kafka.ListClustersOutput
	err error
}

func (m *mockMSK) ListClusters(ctx context.Context, params *kafka.ListClustersInput, optFns ...func(*kafka.Options)) (*kafka.ListClustersOutput, error) {
	return m.out, m.err
}

// ListMSKClustersWithAPI is the testable extraction of ListMSKClusters.
func ListMSKClustersWithAPI(ctx context.Context, api MSKAPI, region string) ([]MSKCluster, error) {
	out, err := api.ListClusters(ctx, nil)
	if err != nil {
		return nil, err
	}
	var clusters []MSKCluster
	for _, cl := range out.ClusterInfoList {
		mc := MSKCluster{
			Name:    awssdk.ToString(cl.ClusterName),
			State:   string(cl.State),
			Brokers: awssdk.ToInt32(cl.NumberOfBrokerNodes),
			Region:  region,
		}
		if cl.CurrentBrokerSoftwareInfo != nil {
			mc.Version = awssdk.ToString(cl.CurrentBrokerSoftwareInfo.KafkaVersion)
		}
		clusters = append(clusters, mc)
	}
	return clusters, nil
}

// ── OpenSearch interface and mock ────────────────────────────────────────────

type OpenSearchAPI interface {
	DescribeDomain(ctx context.Context, params *opensearch.DescribeDomainInput, optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainOutput, error)
}

type mockOpenSearch struct {
	out *opensearch.DescribeDomainOutput
	err error
}

func (m *mockOpenSearch) DescribeDomain(ctx context.Context, params *opensearch.DescribeDomainInput, optFns ...func(*opensearch.Options)) (*opensearch.DescribeDomainOutput, error) {
	return m.out, m.err
}

// GetOSDomainDetailWithAPI is the testable extraction of GetOSDomainDetail.
func GetOSDomainDetailWithAPI(ctx context.Context, api OpenSearchAPI, name string) (*OSDomainDetail, error) {
	out, err := api.DescribeDomain(ctx, &opensearch.DescribeDomainInput{DomainName: awssdk.String(name)})
	if err != nil {
		return nil, err
	}
	d := out.DomainStatus
	if d == nil {
		return &OSDomainDetail{Name: name}, nil
	}
	detail := &OSDomainDetail{
		Name:          awssdk.ToString(d.DomainName),
		EngineVersion: awssdk.ToString(d.EngineVersion),
	}
	if d.ClusterConfig != nil {
		detail.InstanceType = string(d.ClusterConfig.InstanceType)
		detail.InstanceCount = awssdk.ToInt32(d.ClusterConfig.InstanceCount)
	}
	if d.Endpoints != nil {
		detail.Endpoint = d.Endpoints["vpc"]
	}
	return detail, nil
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestListMSKClusters_NilBrokerNodeGroupInfo(t *testing.T) {
	// NumberOfBrokerNodes is on ClusterInfo, not inside BrokerNodeGroupInfo.
	// Brokers should be populated even when BrokerNodeGroupInfo is nil.
	mock := &mockMSK{out: &kafka.ListClustersOutput{
		ClusterInfoList: []kafkatypes.ClusterInfo{
			{
				ClusterName:        awssdk.String("my-cluster"),
				State:              kafkatypes.ClusterStateActive,
				NumberOfBrokerNodes: awssdk.Int32(3),
				BrokerNodeGroupInfo: nil, // explicitly nil
				CurrentBrokerSoftwareInfo: &kafkatypes.BrokerSoftwareInfo{
					KafkaVersion: awssdk.String("3.5.1"),
				},
			},
		},
	}}
	clusters, err := ListMSKClustersWithAPI(context.Background(), mock, "us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].Brokers != 3 {
		t.Errorf("expected Brokers=3, got %d", clusters[0].Brokers)
	}
	if clusters[0].Version != "3.5.1" {
		t.Errorf("expected Version=3.5.1, got %q", clusters[0].Version)
	}
}

func TestListMSKClusters_NilNumberOfBrokerNodes(t *testing.T) {
	// aws.ToInt32 should safely return 0 for nil *int32
	mock := &mockMSK{out: &kafka.ListClustersOutput{
		ClusterInfoList: []kafkatypes.ClusterInfo{
			{
				ClusterName:         awssdk.String("serverless-cluster"),
				State:               kafkatypes.ClusterStateActive,
				NumberOfBrokerNodes: nil,
			},
		},
	}}
	clusters, err := ListMSKClustersWithAPI(context.Background(), mock, "us-west-2")
	if err != nil {
		t.Fatal(err)
	}
	if clusters[0].Brokers != 0 {
		t.Errorf("expected Brokers=0 for nil, got %d", clusters[0].Brokers)
	}
}

func TestListMSKClusters_NilCurrentBrokerSoftwareInfo(t *testing.T) {
	mock := &mockMSK{out: &kafka.ListClustersOutput{
		ClusterInfoList: []kafkatypes.ClusterInfo{
			{
				ClusterName:               awssdk.String("no-version"),
				State:                     kafkatypes.ClusterStateCreating,
				NumberOfBrokerNodes:        awssdk.Int32(2),
				CurrentBrokerSoftwareInfo: nil,
			},
		},
	}}
	clusters, err := ListMSKClustersWithAPI(context.Background(), mock, "eu-west-1")
	if err != nil {
		t.Fatal(err)
	}
	if clusters[0].Version != "" {
		t.Errorf("expected empty Version, got %q", clusters[0].Version)
	}
}

func TestGetOSDomainDetail_NilDomainStatus(t *testing.T) {
	mock := &mockOpenSearch{out: &opensearch.DescribeDomainOutput{
		DomainStatus: nil,
	}}
	detail, err := GetOSDomainDetailWithAPI(context.Background(), mock, "my-domain")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Name != "my-domain" {
		t.Errorf("expected Name=my-domain, got %q", detail.Name)
	}
}

func TestGetOSDomainDetail_NilClusterConfig(t *testing.T) {
	mock := &mockOpenSearch{out: &opensearch.DescribeDomainOutput{
		DomainStatus: &ostypes.DomainStatus{
			DomainName:    awssdk.String("test-domain"),
			EngineVersion: awssdk.String("OpenSearch_2.11"),
			ClusterConfig: nil,
			Endpoints:     map[string]string{"vpc": "vpc-endpoint.example.com"},
		},
	}}
	detail, err := GetOSDomainDetailWithAPI(context.Background(), mock, "test-domain")
	if err != nil {
		t.Fatal(err)
	}
	if detail.InstanceType != "" {
		t.Errorf("expected empty InstanceType, got %q", detail.InstanceType)
	}
	if detail.Endpoint != "vpc-endpoint.example.com" {
		t.Errorf("expected vpc endpoint, got %q", detail.Endpoint)
	}
}

func TestGetOSDomainDetail_NilEndpoints(t *testing.T) {
	mock := &mockOpenSearch{out: &opensearch.DescribeDomainOutput{
		DomainStatus: &ostypes.DomainStatus{
			DomainName:    awssdk.String("no-endpoints"),
			EngineVersion: awssdk.String("OpenSearch_2.11"),
			ClusterConfig: &ostypes.ClusterConfig{
				InstanceType:  ostypes.OpenSearchPartitionInstanceTypeM5LargeSearch,
				InstanceCount: awssdk.Int32(2),
			},
			Endpoints: nil,
		},
	}}
	detail, err := GetOSDomainDetailWithAPI(context.Background(), mock, "no-endpoints")
	if err != nil {
		t.Fatal(err)
	}
	if detail.Endpoint != "" {
		t.Errorf("expected empty Endpoint, got %q", detail.Endpoint)
	}
	if detail.InstanceCount != 2 {
		t.Errorf("expected InstanceCount=2, got %d", detail.InstanceCount)
	}
}
