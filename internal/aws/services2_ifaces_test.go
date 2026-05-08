package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenatypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	cptypes "github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	ectypes "github.com/aws/aws-sdk-go-v2/service/elasticache/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	ebtypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// ── ElastiCache interface and mock ──────────────────────────────────────────

type ElastiCacheAPI interface {
	DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error)
}

type mockElastiCache struct {
	out *elasticache.DescribeCacheClustersOutput
	err error
}

func (m *mockElastiCache) DescribeCacheClusters(ctx context.Context, params *elasticache.DescribeCacheClustersInput, optFns ...func(*elasticache.Options)) (*elasticache.DescribeCacheClustersOutput, error) {
	return m.out, m.err
}

// ListCacheClustersWithAPI is the testable extraction of ListCacheClusters.
func ListCacheClustersWithAPI(ctx context.Context, api ElastiCacheAPI, region string) ([]CacheCluster, error) {
	out, err := api.DescribeCacheClusters(ctx, nil)
	if err != nil {
		return nil, err
	}
	var clusters []CacheCluster
	for _, cl := range out.CacheClusters {
		clusters = append(clusters, CacheCluster{
			ID:       awssdk.ToString(cl.CacheClusterId),
			Engine:   awssdk.ToString(cl.Engine) + " " + awssdk.ToString(cl.EngineVersion),
			Status:   awssdk.ToString(cl.CacheClusterStatus),
			NodeType: awssdk.ToString(cl.CacheNodeType),
			Nodes:    awssdk.ToInt32(cl.NumCacheNodes),
			Region:   region,
		})
	}
	return clusters, nil
}

// ── Athena interface and mock ───────────────────────────────────────────────

type AthenaAPI interface {
	ListWorkGroups(ctx context.Context, params *athena.ListWorkGroupsInput, optFns ...func(*athena.Options)) (*athena.ListWorkGroupsOutput, error)
}

type mockAthena struct {
	out *athena.ListWorkGroupsOutput
	err error
}

func (m *mockAthena) ListWorkGroups(ctx context.Context, params *athena.ListWorkGroupsInput, optFns ...func(*athena.Options)) (*athena.ListWorkGroupsOutput, error) {
	return m.out, m.err
}

// ListAthenaWorkgroupsWithAPI is the testable extraction of ListAthenaWorkgroups.
func ListAthenaWorkgroupsWithAPI(ctx context.Context, api AthenaAPI, region string) ([]AthenaWorkgroup, error) {
	out, err := api.ListWorkGroups(ctx, nil)
	if err != nil {
		return nil, err
	}
	var wgs []AthenaWorkgroup
	for _, wg := range out.WorkGroups {
		wgs = append(wgs, AthenaWorkgroup{
			Name:        awssdk.ToString(wg.Name),
			State:       string(wg.State),
			Description: awssdk.ToString(wg.Description),
			Region:      region,
		})
	}
	return wgs, nil
}

// ── CodePipeline interface and mock ─────────────────────────────────────────

type CodePipelineAPI interface {
	ListPipelines(ctx context.Context, params *codepipeline.ListPipelinesInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelinesOutput, error)
}

type mockCodePipeline struct {
	out *codepipeline.ListPipelinesOutput
	err error
}

func (m *mockCodePipeline) ListPipelines(ctx context.Context, params *codepipeline.ListPipelinesInput, optFns ...func(*codepipeline.Options)) (*codepipeline.ListPipelinesOutput, error) {
	return m.out, m.err
}

// ListPipelinesWithAPI is the testable extraction of ListPipelines.
func ListPipelinesWithAPI(ctx context.Context, api CodePipelineAPI, region string) ([]Pipeline, error) {
	out, err := api.ListPipelines(ctx, nil)
	if err != nil {
		return nil, err
	}
	var pipelines []Pipeline
	for _, p := range out.Pipelines {
		pl := Pipeline{
			Name:    awssdk.ToString(p.Name),
			Version: awssdk.ToInt32(p.Version),
			Region:  region,
		}
		if p.Updated != nil {
			pl.Updated = p.Updated.Format("2006-01-02 15:04")
		}
		pipelines = append(pipelines, pl)
	}
	return pipelines, nil
}

// ── EventBridge interface and mock ──────────────────────────────────────────

type EventBridgeAPI interface {
	ListRules(ctx context.Context, params *eventbridge.ListRulesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRulesOutput, error)
}

type mockEventBridge struct {
	out *eventbridge.ListRulesOutput
	err error
}

func (m *mockEventBridge) ListRules(ctx context.Context, params *eventbridge.ListRulesInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRulesOutput, error) {
	return m.out, m.err
}

// ListEBRulesWithAPI is the testable extraction of ListEBRules.
func ListEBRulesWithAPI(ctx context.Context, api EventBridgeAPI, region string) ([]EBRule, error) {
	out, err := api.ListRules(ctx, nil)
	if err != nil {
		return nil, err
	}
	var rules []EBRule
	for _, r := range out.Rules {
		rules = append(rules, EBRule{
			Name:        awssdk.ToString(r.Name),
			State:       string(r.State),
			Schedule:    awssdk.ToString(r.ScheduleExpression),
			Description: awssdk.ToString(r.Description),
			Region:      region,
		})
	}
	return rules, nil
}

// ── WAF interface and mock ──────────────────────────────────────────────────

type WAFAPI interface {
	ListWebACLs(ctx context.Context, params *wafv2.ListWebACLsInput, optFns ...func(*wafv2.Options)) (*wafv2.ListWebACLsOutput, error)
}

type mockWAF struct {
	out *wafv2.ListWebACLsOutput
	err error
}

func (m *mockWAF) ListWebACLs(ctx context.Context, params *wafv2.ListWebACLsInput, optFns ...func(*wafv2.Options)) (*wafv2.ListWebACLsOutput, error) {
	return m.out, m.err
}

// ListWAFWebACLsWithAPI is the testable extraction of ListWAFWebACLs.
func ListWAFWebACLsWithAPI(ctx context.Context, api WAFAPI, scope waftypes.Scope, region string) ([]WAFWebACL, error) {
	out, err := api.ListWebACLs(ctx, &wafv2.ListWebACLsInput{Scope: scope})
	if err != nil {
		return nil, err
	}
	var acls []WAFWebACL
	for _, a := range out.WebACLs {
		acls = append(acls, WAFWebACL{
			Name:   awssdk.ToString(a.Name),
			ID:     awssdk.ToString(a.Id),
			Scope:  string(scope),
			Region: region,
		})
	}
	return acls, nil
}

// ── Tests ───────────────────────────────────────────────────────────────────

func TestListCacheClustersWithAPI_Success(t *testing.T) {
	mock := &mockElastiCache{out: &elasticache.DescribeCacheClustersOutput{
		CacheClusters: []ectypes.CacheCluster{
			{
				CacheClusterId:     awssdk.String("redis-prod"),
				Engine:             awssdk.String("redis"),
				EngineVersion:      awssdk.String("7.0"),
				CacheClusterStatus: awssdk.String("available"),
				CacheNodeType:      awssdk.String("cache.r6g.large"),
				NumCacheNodes:      awssdk.Int32(3),
			},
		},
	}}
	clusters, err := ListCacheClustersWithAPI(context.Background(), mock, "us-west-2")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].ID != "redis-prod" {
		t.Errorf("expected ID=redis-prod, got %q", clusters[0].ID)
	}
	if clusters[0].Engine != "redis 7.0" {
		t.Errorf("expected Engine='redis 7.0', got %q", clusters[0].Engine)
	}
	if clusters[0].Nodes != 3 {
		t.Errorf("expected Nodes=3, got %d", clusters[0].Nodes)
	}
	if clusters[0].Region != "us-west-2" {
		t.Errorf("expected Region=us-west-2, got %q", clusters[0].Region)
	}
}

func TestListCacheClustersWithAPI_Error(t *testing.T) {
	mock := &mockElastiCache{err: errors.New("access denied")}
	_, err := ListCacheClustersWithAPI(context.Background(), mock, "us-east-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListCacheClustersWithAPI_Empty(t *testing.T) {
	mock := &mockElastiCache{out: &elasticache.DescribeCacheClustersOutput{}}
	clusters, err := ListCacheClustersWithAPI(context.Background(), mock, "us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 0 {
		t.Fatalf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestListAthenaWorkgroupsWithAPI_Success(t *testing.T) {
	mock := &mockAthena{out: &athena.ListWorkGroupsOutput{
		WorkGroups: []athenatypes.WorkGroupSummary{
			{
				Name:        awssdk.String("primary"),
				State:       athenatypes.WorkGroupStateEnabled,
				Description: awssdk.String("default workgroup"),
			},
			{
				Name:  awssdk.String("analytics"),
				State: athenatypes.WorkGroupStateDisabled,
			},
		},
	}}
	wgs, err := ListAthenaWorkgroupsWithAPI(context.Background(), mock, "eu-west-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(wgs) != 2 {
		t.Fatalf("expected 2 workgroups, got %d", len(wgs))
	}
	if wgs[0].Name != "primary" {
		t.Errorf("expected Name=primary, got %q", wgs[0].Name)
	}
	if wgs[0].Description != "default workgroup" {
		t.Errorf("expected Description='default workgroup', got %q", wgs[0].Description)
	}
	if wgs[1].State != "DISABLED" {
		t.Errorf("expected State=DISABLED, got %q", wgs[1].State)
	}
}

func TestListAthenaWorkgroupsWithAPI_Error(t *testing.T) {
	mock := &mockAthena{err: errors.New("throttled")}
	_, err := ListAthenaWorkgroupsWithAPI(context.Background(), mock, "us-east-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListPipelinesWithAPI_Success(t *testing.T) {
	updated := time.Date(2026, 5, 1, 10, 30, 0, 0, time.UTC)
	mock := &mockCodePipeline{out: &codepipeline.ListPipelinesOutput{
		Pipelines: []cptypes.PipelineSummary{
			{
				Name:    awssdk.String("deploy-prod"),
				Version: awssdk.Int32(5),
				Updated: &updated,
			},
			{
				Name:    awssdk.String("deploy-staging"),
				Version: awssdk.Int32(2),
				Updated: nil,
			},
		},
	}}
	pipelines, err := ListPipelinesWithAPI(context.Background(), mock, "us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(pipelines) != 2 {
		t.Fatalf("expected 2 pipelines, got %d", len(pipelines))
	}
	if pipelines[0].Name != "deploy-prod" {
		t.Errorf("expected Name=deploy-prod, got %q", pipelines[0].Name)
	}
	if pipelines[0].Updated != "2026-05-01 10:30" {
		t.Errorf("expected Updated='2026-05-01 10:30', got %q", pipelines[0].Updated)
	}
	if pipelines[1].Updated != "" {
		t.Errorf("expected empty Updated for nil time, got %q", pipelines[1].Updated)
	}
}

func TestListPipelinesWithAPI_Error(t *testing.T) {
	mock := &mockCodePipeline{err: errors.New("not found")}
	_, err := ListPipelinesWithAPI(context.Background(), mock, "us-east-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListEBRulesWithAPI_Success(t *testing.T) {
	mock := &mockEventBridge{out: &eventbridge.ListRulesOutput{
		Rules: []ebtypes.Rule{
			{
				Name:               awssdk.String("daily-backup"),
				State:              ebtypes.RuleStateEnabled,
				ScheduleExpression: awssdk.String("rate(1 day)"),
				Description:        awssdk.String("daily backup trigger"),
			},
			{
				Name:  awssdk.String("event-forwarder"),
				State: ebtypes.RuleStateDisabled,
			},
		},
	}}
	rules, err := ListEBRulesWithAPI(context.Background(), mock, "ap-southeast-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	if rules[0].Name != "daily-backup" {
		t.Errorf("expected Name=daily-backup, got %q", rules[0].Name)
	}
	if rules[0].Schedule != "rate(1 day)" {
		t.Errorf("expected Schedule='rate(1 day)', got %q", rules[0].Schedule)
	}
	if rules[0].Region != "ap-southeast-1" {
		t.Errorf("expected Region=ap-southeast-1, got %q", rules[0].Region)
	}
	if rules[1].Schedule != "" {
		t.Errorf("expected empty Schedule for nil, got %q", rules[1].Schedule)
	}
}

func TestListEBRulesWithAPI_Error(t *testing.T) {
	mock := &mockEventBridge{err: errors.New("service unavailable")}
	_, err := ListEBRulesWithAPI(context.Background(), mock, "us-east-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListWAFWebACLsWithAPI_Success(t *testing.T) {
	mock := &mockWAF{out: &wafv2.ListWebACLsOutput{
		WebACLs: []waftypes.WebACLSummary{
			{
				Name: awssdk.String("block-bots"),
				Id:   awssdk.String("acl-123"),
			},
			{
				Name: awssdk.String("rate-limit"),
				Id:   awssdk.String("acl-456"),
			},
		},
	}}
	acls, err := ListWAFWebACLsWithAPI(context.Background(), mock, waftypes.ScopeRegional, "us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(acls) != 2 {
		t.Fatalf("expected 2 ACLs, got %d", len(acls))
	}
	if acls[0].Name != "block-bots" {
		t.Errorf("expected Name=block-bots, got %q", acls[0].Name)
	}
	if acls[0].Scope != "REGIONAL" {
		t.Errorf("expected Scope=REGIONAL, got %q", acls[0].Scope)
	}
	if acls[1].ID != "acl-456" {
		t.Errorf("expected ID=acl-456, got %q", acls[1].ID)
	}
}

func TestListWAFWebACLsWithAPI_Error(t *testing.T) {
	mock := &mockWAF{err: errors.New("waf error")}
	_, err := ListWAFWebACLsWithAPI(context.Background(), mock, waftypes.ScopeRegional, "us-east-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListWAFWebACLsWithAPI_Empty(t *testing.T) {
	mock := &mockWAF{out: &wafv2.ListWebACLsOutput{}}
	acls, err := ListWAFWebACLsWithAPI(context.Background(), mock, waftypes.ScopeRegional, "us-east-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(acls) != 0 {
		t.Fatalf("expected 0 ACLs, got %d", len(acls))
	}
}
