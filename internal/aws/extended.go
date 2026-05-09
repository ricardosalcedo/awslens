package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

// ── API Gateway ───────────────────────────────────────────────────────────────

type RestAPI struct {
	ID          string
	Name        string
	Description string
	CreatedDate string
	Region      string
}

type APIResource struct {
	Path    string
	Methods []string
}

func (c *Client) ListRestAPIs(ctx context.Context) ([]RestAPI, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := apigateway.NewFromConfig(c.Config)
	out, err := svc.GetRestApis(ctx, &apigateway.GetRestApisInput{})
	if err != nil {
		return nil, err
	}
	var apis []RestAPI
	for _, a := range out.Items {
		api := RestAPI{
			ID:          aws.ToString(a.Id),
			Name:        aws.ToString(a.Name),
			Description: aws.ToString(a.Description),
			Region:      c.Region,
		}
		if a.CreatedDate != nil {
			api.CreatedDate = a.CreatedDate.Format("2006-01-02")
		}
		apis = append(apis, api)
	}
	return apis, nil
}

func (c *Client) GetAPIResources(ctx context.Context, apiID string) ([]APIResource, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := apigateway.NewFromConfig(c.Config)
	out, err := svc.GetResources(ctx, &apigateway.GetResourcesInput{
		RestApiId: aws.String(apiID),
		Embed:     []string{"methods"},
	})
	if err != nil {
		return nil, err
	}
	var resources []APIResource
	for _, r := range out.Items {
		var methods []string
		for m := range r.ResourceMethods {
			methods = append(methods, m)
		}
		resources = append(resources, APIResource{
			Path:    aws.ToString(r.Path),
			Methods: methods,
		})
	}
	return resources, nil
}

func (c *Client) GetAPIStages(ctx context.Context, apiID string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := apigateway.NewFromConfig(c.Config)
	out, err := svc.GetStages(ctx, &apigateway.GetStagesInput{RestApiId: aws.String(apiID)})
	if err != nil {
		return nil, err
	}
	var stages []string
	for _, s := range out.Item {
		stages = append(stages, aws.ToString(s.StageName))
	}
	return stages, nil
}

// ── DynamoDB ──────────────────────────────────────────────────────────────────

type DynamoTable struct {
	Name        string
	Status      string
	ItemCount   int64
	SizeBytes   int64
	BillingMode string
	PKName      string
	SKName      string
	Region      string
}

func (c *Client) ListDynamoTables(ctx context.Context) ([]DynamoTable, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	svc := dynamodb.NewFromConfig(c.Config)
	paginator := dynamodb.NewListTablesPaginator(svc, &dynamodb.ListTablesInput{})
	var names []string
	for paginator.HasMorePages() {
		out, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		names = append(names, out.TableNames...)
	}
	tables := make([]DynamoTable, len(names))
	var wg sync.WaitGroup
	for i, name := range names {
		tables[i] = DynamoTable{Name: name, Region: c.Region}
		wg.Add(1)
		go func(idx int, tblName string) {
			defer wg.Done()
			desc, err := svc.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(tblName)})
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

func (c *Client) ScanDynamoTable(ctx context.Context, tableName string, limit int32) ([]map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := dynamodb.NewFromConfig(c.Config)
	out, err := svc.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(tableName),
		Limit:     aws.Int32(limit),
	})
	if err != nil {
		return nil, err
	}
	var rows []map[string]string
	for _, item := range out.Items {
		row := make(map[string]string)
		for k, v := range item {
			row[k] = dynamoAttrToString(v)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func dynamoAttrToString(av interface{}) string {
	// use fmt since we don't want to import dynamodb/types just for this
	return strings.TrimPrefix(fmt.Sprintf("%v", av), "&{")
}

// ── Load Balancers ────────────────────────────────────────────────────────────

type LoadBalancer struct {
	Name    string
	Type    string
	Scheme  string
	State   string
	DNS     string
	VPC     string
	Created string
}

type LBListener struct {
	Port     int32
	Protocol string
	Rules    int
}

type LBTargetGroup struct {
	Name     string
	Protocol string
	Port     int32
	Healthy  int32
	Unhealthy int32
}

func (c *Client) ListLoadBalancers(ctx context.Context) ([]LoadBalancer, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := elasticloadbalancingv2.NewFromConfig(c.Config)
	out, err := svc.DescribeLoadBalancers(ctx, &elasticloadbalancingv2.DescribeLoadBalancersInput{})
	if err != nil {
		return nil, err
	}
	var lbs []LoadBalancer
	for _, lb := range out.LoadBalancers {
		l := LoadBalancer{
			Name:   aws.ToString(lb.LoadBalancerName),
			Type:   string(lb.Type),
			Scheme: string(lb.Scheme),
			State:  string(lb.State.Code),
			DNS:    aws.ToString(lb.DNSName),
			VPC:    aws.ToString(lb.VpcId),
		}
		if lb.CreatedTime != nil {
			l.Created = lb.CreatedTime.Format("2006-01-02")
		}
		lbs = append(lbs, l)
	}
	return lbs, nil
}

func (c *Client) GetLBListeners(ctx context.Context, lbARN string) ([]LBListener, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := elasticloadbalancingv2.NewFromConfig(c.Config)
	out, err := svc.DescribeListeners(ctx, &elasticloadbalancingv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(lbARN),
	})
	if err != nil {
		return nil, err
	}
	var listeners []LBListener
	for _, l := range out.Listeners {
		listeners = append(listeners, LBListener{
			Port:     aws.ToInt32(l.Port),
			Protocol: string(l.Protocol),
		})
	}
	return listeners, nil
}

// ── Secrets Manager ───────────────────────────────────────────────────────────

type Secret struct {
	Name        string
	ARN         string
	Description string
	LastChanged string
	LastAccessed string
}

func (c *Client) ListSecrets(ctx context.Context) ([]Secret, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := secretsmanager.NewFromConfig(c.Config)
	out, err := svc.ListSecrets(ctx, &secretsmanager.ListSecretsInput{})
	if err != nil {
		return nil, err
	}
	var secrets []Secret
	for _, s := range out.SecretList {
		sec := Secret{
			Name:        aws.ToString(s.Name),
			ARN:         aws.ToString(s.ARN),
			Description: aws.ToString(s.Description),
		}
		if s.LastChangedDate != nil {
			sec.LastChanged = s.LastChangedDate.Format("2006-01-02")
		}
		if s.LastAccessedDate != nil {
			sec.LastAccessed = s.LastAccessedDate.Format("2006-01-02")
		}
		secrets = append(secrets, sec)
	}
	return secrets, nil
}

// ── SSM Parameters ────────────────────────────────────────────────────────────

type SSMParam struct {
	Name        string
	Type        string
	LastModified string
	Version     int64
}

func (c *Client) ListSSMParams(ctx context.Context) ([]SSMParam, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	
	svc := ssm.NewFromConfig(c.Config)
	out, err := svc.DescribeParameters(ctx, &ssm.DescribeParametersInput{})
	if err != nil {
		return nil, err
	}
	var params []SSMParam
	for _, p := range out.Parameters {
		param := SSMParam{
			Name:    aws.ToString(p.Name),
			Type:    string(p.Type),
			Version: p.Version,
		}
		if p.LastModifiedDate != nil {
			param.LastModified = p.LastModifiedDate.Format("2006-01-02")
		}
		params = append(params, param)
	}
	return params, nil
}

func (c *Client) GetSSMParamValue(ctx context.Context, name string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := ssm.NewFromConfig(c.Config)
	out, err := svc.GetParameter(ctx, &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", err
	}
	return aws.ToString(out.Parameter.Value), nil
}

// ── Route53 ───────────────────────────────────────────────────────────────────

type HostedZone struct {
	ID      string
	Name    string
	Private bool
	Records int64
}

type DNSRecord struct {
	Name   string
	Type   string
	TTL    int64
	Values []string
}

func (c *Client) ListHostedZones(ctx context.Context) ([]HostedZone, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := route53.NewFromConfig(c.Config)
	out, err := svc.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		return nil, err
	}
	var zones []HostedZone
	for _, z := range out.HostedZones {
		zones = append(zones, HostedZone{
			ID:      aws.ToString(z.Id),
			Name:    aws.ToString(z.Name),
			Private: z.Config.PrivateZone,
			Records: aws.ToInt64(z.ResourceRecordSetCount),
		})
	}
	return zones, nil
}

func (c *Client) ListDNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := route53.NewFromConfig(c.Config)
	out, err := svc.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	})
	if err != nil {
		return nil, err
	}
	var records []DNSRecord
	for _, r := range out.ResourceRecordSets {
		rec := DNSRecord{
			Name: aws.ToString(r.Name),
			Type: string(r.Type),
			TTL:  aws.ToInt64(r.TTL),
		}
		for _, v := range r.ResourceRecords {
			rec.Values = append(rec.Values, aws.ToString(v.Value))
		}
		if r.AliasTarget != nil {
			rec.Values = append(rec.Values, "ALIAS → "+aws.ToString(r.AliasTarget.DNSName))
		}
		records = append(records, rec)
	}
	return records, nil
}

// ── ECR ───────────────────────────────────────────────────────────────────────

type ECRRepo struct {
	Name      string
	URI       string
	ImageCount int
	ScanOnPush bool
	Created   string
	Region    string
}

type ECRImage struct {
	Tag      string
	Digest   string
	PushedAt string
	SizeBytes int64
}

func (c *Client) ListECRRepos(ctx context.Context) ([]ECRRepo, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := ecr.NewFromConfig(c.Config)
	out, err := svc.DescribeRepositories(ctx, &ecr.DescribeRepositoriesInput{})
	if err != nil {
		return nil, err
	}
	var repos []ECRRepo
	for _, r := range out.Repositories {
		repo := ECRRepo{
			Name:       aws.ToString(r.RepositoryName),
			URI:        aws.ToString(r.RepositoryUri),
			ScanOnPush: r.ImageScanningConfiguration.ScanOnPush,
		}
		if r.CreatedAt != nil {
			repo.Created = r.CreatedAt.Format("2006-01-02")
		}
		repo.Region = c.Region
		repos = append(repos, repo)
	}
	return repos, nil
}

func (c *Client) ListECRImages(ctx context.Context, repoName string) ([]ECRImage, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := ecr.NewFromConfig(c.Config)
	out, err := svc.DescribeImages(ctx, &ecr.DescribeImagesInput{
		RepositoryName: aws.String(repoName),
	})
	if err != nil {
		return nil, err
	}
	var images []ECRImage
	for _, img := range out.ImageDetails {
		i := ECRImage{
			Digest:    aws.ToString(img.ImageDigest),
			SizeBytes: aws.ToInt64(img.ImageSizeInBytes),
		}
		if img.ImagePushedAt != nil {
			i.PushedAt = img.ImagePushedAt.Format("2006-01-02 15:04")
		}
		if len(img.ImageTags) > 0 {
			i.Tag = strings.Join(img.ImageTags, ", ")
		} else {
			i.Tag = "<untagged>"
		}
		images = append(images, i)
	}
	return images, nil
}

// ── Step Functions ────────────────────────────────────────────────────────────

type StateMachine struct {
	Name    string
	ARN     string
	Type    string
	Created string
	Region  string
}

type SFNExecution struct {
	Name    string
	Status  string
	Started string
	Stopped string
}

func (c *Client) ListStateMachines(ctx context.Context) ([]StateMachine, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := sfn.NewFromConfig(c.Config)
	out, err := svc.ListStateMachines(ctx, &sfn.ListStateMachinesInput{})
	if err != nil {
		return nil, err
	}
	var sms []StateMachine
	for _, s := range out.StateMachines {
		sm := StateMachine{
			Name: aws.ToString(s.Name),
			ARN:  aws.ToString(s.StateMachineArn),
			Type: string(s.Type),
		}
		if s.CreationDate != nil {
			sm.Created = s.CreationDate.Format("2006-01-02")
		}
		sm.Region = c.Region
		sms = append(sms, sm)
	}
	return sms, nil
}

func (c *Client) ListSFNExecutions(ctx context.Context, smARN string) ([]SFNExecution, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := sfn.NewFromConfig(c.Config)
	out, err := svc.ListExecutions(ctx, &sfn.ListExecutionsInput{
		StateMachineArn: aws.String(smARN),
		MaxResults:      20,
	})
	if err != nil {
		return nil, err
	}
	var execs []SFNExecution
	for _, e := range out.Executions {
		ex := SFNExecution{
			Name:   aws.ToString(e.Name),
			Status: string(e.Status),
		}
		if e.StartDate != nil {
			ex.Started = e.StartDate.Format("2006-01-02 15:04")
		}
		if e.StopDate != nil {
			ex.Stopped = e.StopDate.Format("2006-01-02 15:04")
		}
		execs = append(execs, ex)
	}
	return execs, nil
}

// ── CloudWatch Alarms ─────────────────────────────────────────────────────────

type CWAlarm struct {
	Name      string
	State     string
	Metric    string
	Namespace string
	Threshold string
	Updated   string
	Region    string
}

func (c *Client) ListAlarms(ctx context.Context) ([]CWAlarm, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := cloudwatch.NewFromConfig(c.Config)
	out, err := svc.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{})
	if err != nil {
		return nil, err
	}
	var alarms []CWAlarm
	for _, a := range out.MetricAlarms {
		alarm := CWAlarm{
			Name:      aws.ToString(a.AlarmName),
			State:     string(a.StateValue),
			Metric:    aws.ToString(a.MetricName),
			Namespace: aws.ToString(a.Namespace),
			Threshold: fmt.Sprintf("%.2f", aws.ToFloat64(a.Threshold)),
		}
		if a.StateUpdatedTimestamp != nil {
			alarm.Updated = a.StateUpdatedTimestamp.Format("2006-01-02 15:04")
		}
		alarm.Region = c.Region
		alarms = append(alarms, alarm)
	}
	return alarms, nil
}

func (c *Client) GetAlarmHistory(ctx context.Context, alarmName string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	svc := cloudwatch.NewFromConfig(c.Config)
	out, err := svc.DescribeAlarmHistory(ctx, &cloudwatch.DescribeAlarmHistoryInput{
		AlarmName:       aws.String(alarmName),
		HistoryItemType: cwtypes.HistoryItemTypeStateUpdate,
		MaxRecords:      aws.Int32(10),
	})
	if err != nil {
		return nil, err
	}
	var lines []string
	for _, h := range out.AlarmHistoryItems {
		ts := ""
		if h.Timestamp != nil {
			ts = h.Timestamp.Format("2006-01-02 15:04")
		}
		lines = append(lines, fmt.Sprintf("%s  %s", ts, aws.ToString(h.HistorySummary)))
	}
	return lines, nil
}
