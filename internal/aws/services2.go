package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	"github.com/aws/aws-sdk-go-v2/service/codebuild"
	"github.com/aws/aws-sdk-go-v2/service/codecommit"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/glue"
	"github.com/aws/aws-sdk-go-v2/service/kafka"
	"github.com/aws/aws-sdk-go-v2/service/opensearch"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// ── ElastiCache ───────────────────────────────────────────────────────────────

type CacheCluster struct {
	ID       string
	Engine   string
	Status   string
	NodeType string
	Nodes    int32
	Region   string
}

func (c *Client) ListCacheClusters(ctx context.Context) ([]CacheCluster, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := elasticache.NewFromConfig(c.Config)
	out, err := svc.DescribeCacheClusters(ctx, nil)
	if err != nil {
		return nil, err
	}
	var clusters []CacheCluster
	for _, cl := range out.CacheClusters {
		clusters = append(clusters, CacheCluster{
			ID:       aws.ToString(cl.CacheClusterId),
			Engine:   aws.ToString(cl.Engine) + " " + aws.ToString(cl.EngineVersion),
			Status:   aws.ToString(cl.CacheClusterStatus),
			NodeType: aws.ToString(cl.CacheNodeType),
			Nodes:    aws.ToInt32(cl.NumCacheNodes),
			Region:   c.Region,
		})
	}
	return clusters, nil
}

// ── OpenSearch ────────────────────────────────────────────────────────────────

type OSDomain struct {
	Name    string
	Region  string
}

type OSDomainDetail struct {
	Name          string
	EngineVersion string
	InstanceType  string
	InstanceCount int32
	Endpoint      string
	Status        string
}

func (c *Client) ListOSDomains(ctx context.Context) ([]OSDomain, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := opensearch.NewFromConfig(c.Config)
	out, err := svc.ListDomainNames(ctx, nil)
	if err != nil {
		return nil, err
	}
	var domains []OSDomain
	for _, d := range out.DomainNames {
		domains = append(domains, OSDomain{Name: aws.ToString(d.DomainName), Region: c.Region})
	}
	return domains, nil
}

func (c *Client) GetOSDomainDetail(ctx context.Context, name string) (*OSDomainDetail, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := opensearch.NewFromConfig(c.Config)
	out, err := svc.DescribeDomain(ctx, &opensearch.DescribeDomainInput{DomainName: aws.String(name)})
	if err != nil {
		return nil, err
	}
	d := out.DomainStatus
	detail := &OSDomainDetail{
		Name:          aws.ToString(d.DomainName),
		EngineVersion: aws.ToString(d.EngineVersion),
	}
	if d.ClusterConfig != nil {
		detail.InstanceType = string(d.ClusterConfig.InstanceType)
		detail.InstanceCount = aws.ToInt32(d.ClusterConfig.InstanceCount)
	}
	if d.Endpoints != nil {
		detail.Endpoint = d.Endpoints["vpc"]
	}
	return detail, nil
}

// ── MSK / Kafka ───────────────────────────────────────────────────────────────

type MSKCluster struct {
	Name    string
	State   string
	Version string
	Brokers int32
	Region  string
}

func (c *Client) ListMSKClusters(ctx context.Context) ([]MSKCluster, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := kafka.NewFromConfig(c.Config)
	out, err := svc.ListClusters(ctx, nil)
	if err != nil {
		return nil, err
	}
	var clusters []MSKCluster
	for _, cl := range out.ClusterInfoList {
		mc := MSKCluster{
			Name:   aws.ToString(cl.ClusterName),
			State:  string(cl.State),
			Region: c.Region,
		}
		if cl.CurrentBrokerSoftwareInfo != nil {
			mc.Version = aws.ToString(cl.CurrentBrokerSoftwareInfo.KafkaVersion)
		}
		if cl.BrokerNodeGroupInfo != nil {
			mc.Brokers = aws.ToInt32(cl.NumberOfBrokerNodes)
		}
		clusters = append(clusters, mc)
	}
	return clusters, nil
}

// ── Glue ──────────────────────────────────────────────────────────────────────

type GlueDatabase struct {
	Name        string
	Description string
	Tables      int
	Region      string
}

type GlueJob struct {
	Name        string
	Role        string
	Type        string
	LastModified string
	Region      string
}

func (c *Client) ListGlueDatabases(ctx context.Context) ([]GlueDatabase, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := glue.NewFromConfig(c.Config)
	out, err := svc.GetDatabases(ctx, nil)
	if err != nil {
		return nil, err
	}
	var dbs []GlueDatabase
	for _, db := range out.DatabaseList {
		dbs = append(dbs, GlueDatabase{
			Name:        aws.ToString(db.Name),
			Description: aws.ToString(db.Description),
			Region:      c.Region,
		})
	}
	return dbs, nil
}

func (c *Client) ListGlueJobs(ctx context.Context) ([]GlueJob, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := glue.NewFromConfig(c.Config)
	out, err := svc.GetJobs(ctx, nil)
	if err != nil {
		return nil, err
	}
	var jobs []GlueJob
	for _, j := range out.Jobs {
		job := GlueJob{
			Name:   aws.ToString(j.Name),
			Role:   aws.ToString(j.Role),
			Type:   aws.ToString(j.Command.ScriptLocation),
			Region: c.Region,
		}
		if j.LastModifiedOn != nil {
			job.LastModified = j.LastModifiedOn.Format("2006-01-02")
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// ── Athena ────────────────────────────────────────────────────────────────────

type AthenaWorkgroup struct {
	Name        string
	State       string
	Description string
	Region      string
}

type AthenaQuery struct {
	ID          string
	Name        string
	Database    string
	Description string
}

func (c *Client) ListAthenaWorkgroups(ctx context.Context) ([]AthenaWorkgroup, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := athena.NewFromConfig(c.Config)
	out, err := svc.ListWorkGroups(ctx, nil)
	if err != nil {
		return nil, err
	}
	var wgs []AthenaWorkgroup
	for _, wg := range out.WorkGroups {
		wgs = append(wgs, AthenaWorkgroup{
			Name:        aws.ToString(wg.Name),
			State:       string(wg.State),
			Description: aws.ToString(wg.Description),
			Region:      c.Region,
		})
	}
	return wgs, nil
}

func (c *Client) ListAthenaSavedQueries(ctx context.Context) ([]AthenaQuery, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := athena.NewFromConfig(c.Config)
	ids, err := svc.ListNamedQueries(ctx, nil)
	if err != nil || len(ids.NamedQueryIds) == 0 {
		return nil, err
	}
	out, err := svc.BatchGetNamedQuery(ctx, &athena.BatchGetNamedQueryInput{NamedQueryIds: ids.NamedQueryIds})
	if err != nil {
		return nil, err
	}
	var queries []AthenaQuery
	for _, q := range out.NamedQueries {
		queries = append(queries, AthenaQuery{
			ID:          aws.ToString(q.NamedQueryId),
			Name:        aws.ToString(q.Name),
			Database:    aws.ToString(q.Database),
			Description: aws.ToString(q.Description),
		})
	}
	return queries, nil
}

// ── CodeCommit ────────────────────────────────────────────────────────────────

type CodeRepo struct {
	Name        string
	Description string
	LastModified string
	CloneURL    string
	Region      string
}

func (c *Client) ListCodeRepos(ctx context.Context) ([]CodeRepo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := codecommit.NewFromConfig(c.Config)
	out, err := svc.ListRepositories(ctx, nil)
	if err != nil {
		return nil, err
	}
	var repos []CodeRepo
	for _, r := range out.Repositories {
		repo := CodeRepo{
			Name:   aws.ToString(r.RepositoryName),
			Region: c.Region,
		}
		// get details for clone URL
		detail, err := svc.GetRepository(ctx, &codecommit.GetRepositoryInput{RepositoryName: r.RepositoryName})
		if err == nil && detail.RepositoryMetadata != nil {
			repo.Description = aws.ToString(detail.RepositoryMetadata.RepositoryDescription)
			repo.CloneURL = aws.ToString(detail.RepositoryMetadata.CloneUrlHttp)
			if detail.RepositoryMetadata.LastModifiedDate != nil {
				repo.LastModified = detail.RepositoryMetadata.LastModifiedDate.Format("2006-01-02")
			}
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

// ── CodePipeline ──────────────────────────────────────────────────────────────

type Pipeline struct {
	Name    string
	Version int32
	Updated string
	Region  string
}

type PipelineExecution struct {
	ID      string
	Status  string
	Trigger string
	Started string
}

func (c *Client) ListPipelines(ctx context.Context) ([]Pipeline, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := codepipeline.NewFromConfig(c.Config)
	out, err := svc.ListPipelines(ctx, nil)
	if err != nil {
		return nil, err
	}
	var pipelines []Pipeline
	for _, p := range out.Pipelines {
		pl := Pipeline{
			Name:    aws.ToString(p.Name),
			Version: aws.ToInt32(p.Version),
			Region:  c.Region,
		}
		if p.Updated != nil {
			pl.Updated = p.Updated.Format("2006-01-02 15:04")
		}
		pipelines = append(pipelines, pl)
	}
	return pipelines, nil
}

func (c *Client) ListPipelineExecutions(ctx context.Context, name string) ([]PipelineExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := codepipeline.NewFromConfig(c.Config)
	out, err := svc.ListPipelineExecutions(ctx, &codepipeline.ListPipelineExecutionsInput{
		PipelineName: aws.String(name),
		MaxResults:   aws.Int32(10),
	})
	if err != nil {
		return nil, err
	}
	var execs []PipelineExecution
	for _, e := range out.PipelineExecutionSummaries {
		ex := PipelineExecution{
			ID:     aws.ToString(e.PipelineExecutionId),
			Status: string(e.Status),
		}
		if e.StartTime != nil {
			ex.Started = e.StartTime.Format("2006-01-02 15:04")
		}
		if e.Trigger != nil {
			ex.Trigger = string(e.Trigger.TriggerType)
		}
		execs = append(execs, ex)
	}
	return execs, nil
}

// ── CodeBuild ─────────────────────────────────────────────────────────────────

type BuildProject struct {
	Name        string
	Description string
	Runtime     string
	LastBuild   string
	Region      string
}

func (c *Client) ListBuildProjects(ctx context.Context) ([]BuildProject, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := codebuild.NewFromConfig(c.Config)
	names, err := svc.ListProjects(ctx, nil)
	if err != nil || len(names.Projects) == 0 {
		return nil, err
	}
	out, err := svc.BatchGetProjects(ctx, &codebuild.BatchGetProjectsInput{Names: names.Projects})
	if err != nil {
		return nil, err
	}
	var projects []BuildProject
	for _, p := range out.Projects {
		proj := BuildProject{
			Name:        aws.ToString(p.Name),
			Description: aws.ToString(p.Description),
			Region:      c.Region,
		}
		if p.Environment != nil {
			proj.Runtime = string(p.Environment.Type)
		}
		if p.LastModified != nil {
			proj.LastBuild = p.LastModified.Format("2006-01-02")
		}
		projects = append(projects, proj)
	}
	return projects, nil
}

// ── EventBridge ───────────────────────────────────────────────────────────────

type EBRule struct {
	Name        string
	State       string
	Schedule    string
	Description string
	Region      string
}

type EBTarget struct {
	ID  string
	ARN string
}

func (c *Client) ListEBRules(ctx context.Context) ([]EBRule, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := eventbridge.NewFromConfig(c.Config)
	out, err := svc.ListRules(ctx, nil)
	if err != nil {
		return nil, err
	}
	var rules []EBRule
	for _, r := range out.Rules {
		rules = append(rules, EBRule{
			Name:        aws.ToString(r.Name),
			State:       string(r.State),
			Schedule:    aws.ToString(r.ScheduleExpression),
			Description: aws.ToString(r.Description),
			Region:      c.Region,
		})
	}
	return rules, nil
}

func (c *Client) ListEBTargets(ctx context.Context, rule string) ([]EBTarget, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := eventbridge.NewFromConfig(c.Config)
	out, err := svc.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{Rule: aws.String(rule)})
	if err != nil {
		return nil, err
	}
	var targets []EBTarget
	for _, t := range out.Targets {
		targets = append(targets, EBTarget{
			ID:  aws.ToString(t.Id),
			ARN: aws.ToString(t.Arn),
		})
	}
	return targets, nil
}

// ── WAF ───────────────────────────────────────────────────────────────────────

type WAFWebACL struct {
	Name        string
	ID          string
	Scope       string
	Rules       int
	Region      string
}

func (c *Client) ListWAFWebACLs(ctx context.Context) ([]WAFWebACL, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := wafv2.NewFromConfig(c.Config)
	var acls []WAFWebACL
	for _, scope := range []waftypes.Scope{waftypes.ScopeRegional} {
		out, err := svc.ListWebACLs(ctx, &wafv2.ListWebACLsInput{Scope: scope})
		if err != nil {
			continue
		}
		for _, a := range out.WebACLs {
			acls = append(acls, WAFWebACL{
				Name:   aws.ToString(a.Name),
				ID:     aws.ToString(a.Id),
				Scope:  string(scope),
				Region: c.Region,
			})
		}
	}
	return acls, nil
}

// ── Multi-region wrappers ─────────────────────────────────────────────────────

func AllRegionsCacheClusters(ctx context.Context, c *Client) ([]CacheCluster, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]CacheCluster, error) {
		return rc.ListCacheClusters(ctx)
	})
}

func AllRegionsOSDomains(ctx context.Context, c *Client) ([]OSDomain, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]OSDomain, error) {
		return rc.ListOSDomains(ctx)
	})
}

func AllRegionsMSKClusters(ctx context.Context, c *Client) ([]MSKCluster, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]MSKCluster, error) {
		return rc.ListMSKClusters(ctx)
	})
}

func AllRegionsGlueDatabases(ctx context.Context, c *Client) ([]GlueDatabase, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]GlueDatabase, error) {
		return rc.ListGlueDatabases(ctx)
	})
}

func AllRegionsAthenaWorkgroups(ctx context.Context, c *Client) ([]AthenaWorkgroup, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]AthenaWorkgroup, error) {
		return rc.ListAthenaWorkgroups(ctx)
	})
}

func AllRegionsCodeRepos(ctx context.Context, c *Client) ([]CodeRepo, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]CodeRepo, error) {
		return rc.ListCodeRepos(ctx)
	})
}

func AllRegionsPipelines(ctx context.Context, c *Client) ([]Pipeline, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]Pipeline, error) {
		return rc.ListPipelines(ctx)
	})
}

func AllRegionsBuildProjects(ctx context.Context, c *Client) ([]BuildProject, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]BuildProject, error) {
		return rc.ListBuildProjects(ctx)
	})
}

func AllRegionsEBRules(ctx context.Context, c *Client) ([]EBRule, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]EBRule, error) {
		return rc.ListEBRules(ctx)
	})
}

func AllRegionsWAFWebACLs(ctx context.Context, c *Client) ([]WAFWebACL, []string) {
	return aggregateRegions(ctx, c, func(ctx context.Context, rc *Client) ([]WAFWebACL, error) {
		return rc.ListWAFWebACLs(ctx)
	})
}

// suppress unused import warning
var _ = fmt.Sprintf
