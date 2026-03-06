package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	awsclient "github.com/awslens/awslens/internal/aws"
)

// ── styles ───────────────────────────────────────────────────────────────────

var (
	orange      = lipgloss.Color("#FF9900")
	cyan        = lipgloss.Color("#00D4FF")
	green       = lipgloss.Color("#00FF87")
	red         = lipgloss.Color("#FF5F5F")
	yellow      = lipgloss.Color("#FFD700")
	muted       = lipgloss.Color("#555555")

	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(orange).Padding(0, 1)
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(cyan).
			BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(muted)
	selectedRow = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	mutedStyle  = lipgloss.NewStyle().Foreground(muted)
	helpStyle   = lipgloss.NewStyle().Foreground(muted)
	errorStyle  = lipgloss.NewStyle().Foreground(red)
	okStyle     = lipgloss.NewStyle().Foreground(green)
	warnStyle   = lipgloss.NewStyle().Foreground(yellow)
)

// ── messages ─────────────────────────────────────────────────────────────────

type errMsg struct{ err error }
type writeOKMsg struct{ info string } // successful write, re-fetch list
type writeErrMsg struct{ err error }  // write failed, show error
type ec2Msg struct{ instances []awsclient.Instance }
type lambdaMsg struct{ functions []awsclient.Function }
type s3Msg struct{ buckets []awsclient.Bucket }
type rdsMsg struct{ dbs []awsclient.DBInstance }
type ecsMsg struct{ clusters []awsclient.Cluster }
type sqsMsg struct{ queues []awsclient.Queue }
type snsMsg struct{ topics []awsclient.Topic }
type cfnMsg struct{ stacks []awsclient.Stack }
type costsMsg struct {
	entries []awsclient.CostEntry
	total   string
}
type dynamoMsg struct{ tables []awsclient.DynamoTable }
type apigwMsg struct{ apis []awsclient.RestAPI }
type ecrMsg struct{ repos []awsclient.ECRRepo }
type sfnMsg struct{ machines []awsclient.StateMachine }
type albMsg struct{ lbs []awsclient.LoadBalancer }
type route53Msg struct{ zones []awsclient.HostedZone }
type secretsMsg struct{ secrets []awsclient.Secret }
type ssmMsg struct{ params []awsclient.SSMParam }
type cwMsg struct{ alarms []awsclient.CWAlarm }
type cacheMsg struct{ clusters []awsclient.CacheCluster }
type osMsg struct{ domains []awsclient.OSDomain }
type mskMsg struct{ clusters []awsclient.MSKCluster }
type glueMsg struct{ dbs []awsclient.GlueDatabase }
type athenaMsg struct{ wgs []awsclient.AthenaWorkgroup }
type codeCommitMsg struct{ repos []awsclient.CodeRepo }
type codePipelineMsg struct{ pipelines []awsclient.Pipeline }
type codeBuildMsg struct{ projects []awsclient.BuildProject }
type ebMsg struct{ rules []awsclient.EBRule }
type wafMsg struct{ acls []awsclient.WAFWebACL }
type insightMsg struct{ text string }

// ── view enum ────────────────────────────────────────────────────────────────

type view int

const (
	viewDashboard view = iota
	viewEC2
	viewLambda
	viewS3
	viewRDS
	viewDynamo
	viewAPIGW
	viewECS
	viewECR
	viewSFN
	viewALB
	viewRoute53
	viewSecrets
	viewSSM
	viewSQS
	viewSNS
	viewCW
	viewCFN
	viewCosts
	viewElastiCache
	viewOpenSearch
	viewMSK
	viewGlue
	viewAthena
	viewCodeCommit
	viewCodePipeline
	viewCodeBuild
	viewEventBridge
	viewWAF
)

type menuItem struct {
	label string
	desc  string
	v     view
}

var menu = []menuItem{
	{"EC2", "Instances & Security Groups", viewEC2},
	{"Lambda", "Functions & Runtimes", viewLambda},
	{"S3", "Buckets & Objects", viewS3},
	{"RDS", "Databases", viewRDS},
	{"DynamoDB", "Tables & Items", viewDynamo},
	{"API Gateway", "REST APIs & Resources", viewAPIGW},
	{"ECS", "Clusters & Tasks", viewECS},
	{"ECR", "Container Registries & Images", viewECR},
	{"Step Functions", "State Machines & Executions", viewSFN},
	{"Load Balancers", "ALB / NLB", viewALB},
	{"Route53", "Hosted Zones & DNS Records", viewRoute53},
	{"Secrets Manager", "Secrets", viewSecrets},
	{"SSM", "Parameter Store", viewSSM},
	{"SQS", "Queues & Messages", viewSQS},
	{"SNS", "Topics & Subscriptions", viewSNS},
	{"CloudWatch", "Alarms & History", viewCW},
	{"CloudFormation", "Stacks & Drift", viewCFN},
	{"Costs", "Monthly breakdown by service", viewCosts},
	{"ElastiCache", "Redis & Memcached clusters", viewElastiCache},
	{"OpenSearch", "Search domains", viewOpenSearch},
	{"MSK", "Managed Kafka clusters", viewMSK},
	{"Glue", "ETL databases & jobs", viewGlue},
	{"Athena", "Workgroups & saved queries", viewAthena},
	{"CodeCommit", "Git repositories", viewCodeCommit},
	{"CodePipeline", "CI/CD pipelines", viewCodePipeline},
	{"CodeBuild", "Build projects", viewCodeBuild},
	{"EventBridge", "Rules & targets", viewEventBridge},
	{"WAF", "Web ACLs", viewWAF},
}

type accessMsg map[string]bool

// ── model ────────────────────────────────────────────────────────────────────

type model struct {
	client  *awsclient.Client
	current view
	cursor  int
	width   int
	height  int
	loading bool
	spinner spinner.Model
	err     error
	probing bool   // true while running access probe
	access  map[string]bool // nil = not probed yet

	// list data
	instances []awsclient.Instance
	functions []awsclient.Function
	buckets   []awsclient.Bucket
	dbs       []awsclient.DBInstance
	clusters  []awsclient.Cluster
	queues    []awsclient.Queue
	topics    []awsclient.Topic
	stacks    []awsclient.Stack
	costs     []awsclient.CostEntry
	costTotal string
	tables    []awsclient.DynamoTable
	apis      []awsclient.RestAPI
	repos     []awsclient.ECRRepo
	machines  []awsclient.StateMachine
	lbs       []awsclient.LoadBalancer
	zones     []awsclient.HostedZone
	secrets   []awsclient.Secret
	params    []awsclient.SSMParam
	alarms       []awsclient.CWAlarm
	cacheClusters []awsclient.CacheCluster
	osDomains     []awsclient.OSDomain
	mskClusters   []awsclient.MSKCluster
	glueDbs       []awsclient.GlueDatabase
	athenaWGs     []awsclient.AthenaWorkgroup
	codeRepos     []awsclient.CodeRepo
	pipelines     []awsclient.Pipeline
	buildProjects []awsclient.BuildProject
	ebRules       []awsclient.EBRule
	wafACLs       []awsclient.WAFWebACL

	// detail
	detail detailModel

	// modal
	modal   modal
	modalOK func() tea.Cmd // action to run on confirm/submit
}

func (m model) Init() tea.Cmd {
	client := m.client
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			access := client.ProbeAccess(context.Background())
			return accessMsg(access)
		},
	)
}

// ── update ───────────────────────────────────────────────────────────────────

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// handle detail data messages first
	if handleDetailMsg(&m.detail, msg) {
		m.loading = false
		return m, nil
	}

	// handle write results
	switch msg := msg.(type) {
	case writeOKMsg:
		m.modal.reset()
		m.modalOK = nil
		m.err = nil
		// re-fetch the current service list
		return m.enterService()
	case writeErrMsg:
		m.modal.reset()
		m.modalOK = nil
		m.err = msg.err
		return m, nil
	}

	// modal intercepts all keys when active
	if m.modal.active() {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "esc":
				m.modal.reset()
				m.modalOK = nil
			case "y":
				if m.modal.kind == modalConfirm && m.modalOK != nil {
					fn := m.modalOK
					m.modal.reset()
					m.modalOK = nil
					return m, fn()
				}
			case "enter":
				if m.modal.kind == modalInput && m.modal.input != "" && m.modalOK != nil {
					fn := m.modalOK
					m.modal.reset()
					m.modalOK = nil
					return m, fn()
				}
			case "backspace":
				if m.modal.kind == modalInput && len(m.modal.input) > 0 {
					m.modal.input = m.modal.input[:len(m.modal.input)-1]
				}
			default:
				if m.modal.kind == modalInput && len(key.String()) == 1 {
					m.modal.input += key.String()
				}
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		// detail view keys
		if m.detail.active != detailNone {
			return m.handleDetailKey(msg)
		}
		// service list keys
		if m.current != viewDashboard {
			return m.handleServiceKey(msg)
		}
		// dashboard keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			for m.cursor > 0 {
				m.cursor--
				if m.access == nil || m.access[menu[m.cursor].label] {
					break
				}
			}
		case "down", "j":
			for m.cursor < len(menu)-1 {
				m.cursor++
				if m.access == nil || m.access[menu[m.cursor].label] {
					break
				}
			}
		case "enter", " ":
			return m.enterService()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case accessMsg:
		m.probing = false
		m.access = map[string]bool(msg)
		return m, nil
	case errMsg:
		m.loading = false
		m.err = msg.err
	case ec2Msg:
		m.loading = false
		m.instances = msg.instances
	case lambdaMsg:
		m.loading = false
		m.functions = msg.functions
	case s3Msg:
		m.loading = false
		m.buckets = msg.buckets
	case rdsMsg:
		m.loading = false
		m.dbs = msg.dbs
	case ecsMsg:
		m.loading = false
		m.clusters = msg.clusters
	case sqsMsg:
		m.loading = false
		m.queues = msg.queues
	case snsMsg:
		m.loading = false
		m.topics = msg.topics
	case cfnMsg:
		m.loading = false
		m.stacks = msg.stacks
	case costsMsg:
		m.loading = false
		m.costs = msg.entries
		m.costTotal = msg.total
	case dynamoMsg:
		m.loading = false
		m.tables = msg.tables
	case apigwMsg:
		m.loading = false
		m.apis = msg.apis
	case ecrMsg:
		m.loading = false
		m.repos = msg.repos
	case sfnMsg:
		m.loading = false
		m.machines = msg.machines
	case albMsg:
		m.loading = false
		m.lbs = msg.lbs
	case route53Msg:
		m.loading = false
		m.zones = msg.zones
	case secretsMsg:
		m.loading = false
		m.secrets = msg.secrets
	case ssmMsg:
		m.loading = false
		m.params = msg.params
	case cwMsg:
		m.loading = false
		m.alarms = msg.alarms
	case cacheMsg:
		m.loading = false
		m.cacheClusters = msg.clusters
	case osMsg:
		m.loading = false
		m.osDomains = msg.domains
	case mskMsg:
		m.loading = false
		m.mskClusters = msg.clusters
	case glueMsg:
		m.loading = false
		m.glueDbs = msg.dbs
	case athenaMsg:
		m.loading = false
		m.athenaWGs = msg.wgs
	case codeCommitMsg:
		m.loading = false
		m.codeRepos = msg.repos
	case codePipelineMsg:
		m.loading = false
		m.pipelines = msg.pipelines
	case codeBuildMsg:
		m.loading = false
		m.buildProjects = msg.projects
	case ebMsg:
		m.loading = false
		m.ebRules = msg.rules
	case wafMsg:
		m.loading = false
		m.wafACLs = msg.acls
	case insightMsg:
		m.loading = false
		m.modal = modal{kind: modalInsight, title: "✨ AI Insight", body: msg.text}
	}

	return m, nil
}

func (m model) handleDetailKey(msg tea.KeyMsg) (model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.detail.reset()
		m.err = nil
	case "up", "k":
		if m.detail.cursor > 0 {
			m.detail.cursor--
		}
	case "down", "j":
		maxD := len(m.detail.s3Objects) - 1
		if m.detail.active != detailS3Objects || m.detail.cursor < maxD {
			m.detail.cursor++
		}
	case "left", "h":
		if m.detail.tab > 0 {
			m.detail.tab--
		}
	case "right", "l":
		if m.detail.active == detailLambda {
			if m.detail.tab < 2 {
				m.detail.tab++
			}
		}
	// Lambda: press L to view logs
	case "L":
		if m.detail.active == detailLambda && m.detail.lambdaDetail != nil {
			m.detail.active = detailLambdaLogs
			m.loading = true
			fn := m.functions[m.detail.parentIdx]
			client := m.client.NewForRegion(fn.Region)
			return m, fetchLambdaLogs(client, m.detail.lambdaDetail.Name)
		}
	// S3: enter to drill into folder
	case "enter":
		if m.detail.active == detailS3Objects && len(m.detail.s3Objects) > 0 {
			if m.detail.cursor >= len(m.detail.s3Objects) {
				m.detail.cursor = len(m.detail.s3Objects) - 1
			}
			obj := m.detail.s3Objects[m.detail.cursor]
			if obj.StorageClass == "PREFIX" {
				m.loading = true
				m.detail.cursor = 0 // reset cursor for new folder
				bucket := m.detail.s3Bucket
				client := m.client.NewForRegion(m.client.Region)
				return m, fetchS3Objects(client, bucket, obj.Key)
			}
		}
		// CFN: enter to cycle sub-views
		if m.detail.active == detailCFNResources {
			idx := m.detail.parentIdx
			if idx < len(m.stacks) {
				m.detail.active = detailCFNEvents
				m.loading = true
				return m, fetchCFNEvents(m.client, m.stacks[idx].Name)
			}
		}
		if m.detail.active == detailCFNEvents {
			idx := m.detail.parentIdx
			if idx < len(m.stacks) {
				m.detail.active = detailCFNOutputs
				m.loading = true
				return m, fetchCFNOutputs(m.client, m.stacks[idx].Name)
			}
		}
	}
	return m, nil
}

func (m model) handleServiceKey(msg tea.KeyMsg) (model, tea.Cmd) {
	maxC := m.maxCursor()
	switch msg.String() {
	case "q", "esc":
		m.current = viewDashboard
		m.cursor = 0
		m.err = nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < maxC {
			m.cursor++
		}
	case "r":
		return m.enterService()
	case "enter", " ":
		return m.enterDetail()
	case "n":
		return m.handleNew()
	case "d":
		return m.handleDelete()
	case "e":
		return m.handleEdit()
	case "i":
		return m.handleInsight()
	case "p":
		return m.handlePurge()
	case "t":
		return m.handleToggle()
	}
	return m, nil
}

func (m model) maxCursor() int {
	switch m.current {
	case viewEC2:
		return max0(len(m.instances) - 1)
	case viewLambda:
		return max0(len(m.functions) - 1)
	case viewS3:
		return max0(len(m.buckets) - 1)
	case viewRDS:
		return max0(len(m.dbs) - 1)
	case viewECS:
		return max0(len(m.clusters) - 1)
	case viewSQS:
		return max0(len(m.queues) - 1)
	case viewSNS:
		return max0(len(m.topics) - 1)
	case viewCFN:
		return max0(len(m.stacks) - 1)
	case viewCosts:
		return max0(len(m.costs) - 1)
	case viewDynamo:
		return max0(len(m.tables) - 1)
	case viewAPIGW:
		return max0(len(m.apis) - 1)
	case viewECR:
		return max0(len(m.repos) - 1)
	case viewSFN:
		return max0(len(m.machines) - 1)
	case viewALB:
		return max0(len(m.lbs) - 1)
	case viewRoute53:
		return max0(len(m.zones) - 1)
	case viewSecrets:
		return max0(len(m.secrets) - 1)
	case viewSSM:
		return max0(len(m.params) - 1)
	case viewCW:
		return max0(len(m.alarms) - 1)
	case viewElastiCache:
		return max0(len(m.cacheClusters) - 1)
	case viewOpenSearch:
		return max0(len(m.osDomains) - 1)
	case viewMSK:
		return max0(len(m.mskClusters) - 1)
	case viewGlue:
		return max0(len(m.glueDbs) - 1)
	case viewAthena:
		return max0(len(m.athenaWGs) - 1)
	case viewCodeCommit:
		return max0(len(m.codeRepos) - 1)
	case viewCodePipeline:
		return max0(len(m.pipelines) - 1)
	case viewCodeBuild:
		return max0(len(m.buildProjects) - 1)
	case viewEventBridge:
		return max0(len(m.ebRules) - 1)
	case viewWAF:
		return max0(len(m.wafACLs) - 1)
	}
	return 0
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

func (m model) enterDetail() (model, tea.Cmd) {
	m.detail.reset()
	m.detail.parentIdx = m.cursor
	m.loading = true
	m.err = nil
	switch m.current {
	case viewLambda:
		if m.cursor < len(m.functions) {
			fn := m.functions[m.cursor]
			m.detail.active = detailLambda
			client := m.client.NewForRegion(fn.Region)
			return m, fetchLambdaDetail(client, fn.Name)
		}
	case viewS3:
		if m.cursor < len(m.buckets) {
			m.detail.active = detailS3Objects
			m.detail.s3Bucket = m.buckets[m.cursor].Name
			client := m.client.NewForRegion(m.client.Region)
			return m, fetchS3Objects(client, m.buckets[m.cursor].Name, "")
		}
	case viewCFN:
		if m.cursor < len(m.stacks) {
			m.detail.active = detailCFNResources
			client := m.client.NewForRegion(m.stacks[m.cursor].Region)
			return m, fetchCFNResources(client, m.stacks[m.cursor].Name)
		}
	case viewSNS:
		if m.cursor < len(m.topics) {
			m.detail.active = detailSNSSubscriptions
			return m, fetchSNSSubs(m.client, m.topics[m.cursor].ARN)
		}
	case viewDynamo:
		if m.cursor < len(m.tables) {
			m.detail.active = detailDynamoItems
			m.detail.dynamoTable = m.tables[m.cursor].Name
			client := m.client.NewForRegion(m.tables[m.cursor].Region)
			return m, fetchDynamoItems(client, m.tables[m.cursor].Name)
		}
	case viewAPIGW:
		if m.cursor < len(m.apis) {
			api := m.apis[m.cursor]
			m.detail.active = detailAPIResources
			client := m.client.NewForRegion(api.Region)
			return m, fetchAPIResources(client, api.ID)
		}
	case viewECR:
		if m.cursor < len(m.repos) {
			m.detail.active = detailECRImages
			client := m.client.NewForRegion(m.repos[m.cursor].Region)
			return m, fetchECRImages(client, m.repos[m.cursor].Name)
		}
	case viewSFN:
		if m.cursor < len(m.machines) {
			m.detail.active = detailSFNExecutions
			client := m.client.NewForRegion(m.machines[m.cursor].Region)
			return m, fetchSFNExecutions(client, m.machines[m.cursor].ARN)
		}
	case viewRoute53:
		if m.cursor < len(m.zones) {
			m.detail.active = detailDNSRecords
			return m, fetchDNSRecords(m.client, m.zones[m.cursor].ID)
		}
	case viewCW:
		if m.cursor < len(m.alarms) {
			m.detail.active = detailAlarmHistory
			client := m.client.NewForRegion(m.alarms[m.cursor].Region)
			return m, fetchAlarmHistory(client, m.alarms[m.cursor].Name)
		}
	case viewSSM:
		if m.cursor < len(m.params) {
			m.detail.active = detailSSMValue
			return m, fetchSSMValue(m.client, m.params[m.cursor].Name)
		}
	}
	m.loading = false
	return m, nil
}

func (m model) enterService() (model, tea.Cmd) {
	m.current = menu[m.cursor].v
	m.cursor = 0
	m.loading = true
	m.err = nil
	client := m.client
	switch m.current {
	case viewEC2:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsInstances(context.Background(), client)
			return ec2Msg{data}
		}
	case viewLambda:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsFunctions(context.Background(), client)
			return lambdaMsg{data}
		}
	case viewS3:
		return m, func() tea.Msg {
			data, err := client.ListBuckets(context.Background())
			if err != nil { return errMsg{err} }
			return s3Msg{data}
		}
	case viewRDS:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsDBInstances(context.Background(), client)
			return rdsMsg{data}
		}
	case viewECS:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsClusters(context.Background(), client)
			return ecsMsg{data}
		}
	case viewSQS:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsQueues(context.Background(), client)
			return sqsMsg{data}
		}
	case viewSNS:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsTopics(context.Background(), client)
			return snsMsg{data}
		}
	case viewCFN:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsStacks(context.Background(), client)
			return cfnMsg{data}
		}
	case viewCosts:
		return m, func() tea.Msg {
			data, total, err := client.GetMonthlyCosts(context.Background())
			if err != nil { return errMsg{err} }
			return costsMsg{data, total}
		}
	case viewDynamo:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsDynamoTables(context.Background(), client)
			return dynamoMsg{data}
		}
	case viewAPIGW:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsRestAPIs(context.Background(), client)
			return apigwMsg{data}
		}
	case viewECR:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsECRRepos(context.Background(), client)
			return ecrMsg{data}
		}
	case viewSFN:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsStateMachines(context.Background(), client)
			return sfnMsg{data}
		}
	case viewALB:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsLoadBalancers(context.Background(), client)
			return albMsg{data}
		}
	case viewRoute53:
		return m, func() tea.Msg {
			data, err := client.ListHostedZones(context.Background())
			if err != nil { return errMsg{err} }
			return route53Msg{data}
		}
	case viewSecrets:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsSecrets(context.Background(), client)
			return secretsMsg{data}
		}
	case viewSSM:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsSSMParams(context.Background(), client)
			return ssmMsg{data}
		}
	case viewCW:
		return m, func() tea.Msg {
			data := awsclient.AllRegionsAlarms(context.Background(), client)
			return cwMsg{data}
		}
	case viewElastiCache:
		return m, func() tea.Msg {
			return cacheMsg{awsclient.AllRegionsCacheClusters(context.Background(), client)}
		}
	case viewOpenSearch:
		return m, func() tea.Msg {
			return osMsg{awsclient.AllRegionsOSDomains(context.Background(), client)}
		}
	case viewMSK:
		return m, func() tea.Msg {
			return mskMsg{awsclient.AllRegionsMSKClusters(context.Background(), client)}
		}
	case viewGlue:
		return m, func() tea.Msg {
			return glueMsg{awsclient.AllRegionsGlueDatabases(context.Background(), client)}
		}
	case viewAthena:
		return m, func() tea.Msg {
			return athenaMsg{awsclient.AllRegionsAthenaWorkgroups(context.Background(), client)}
		}
	case viewCodeCommit:
		return m, func() tea.Msg {
			return codeCommitMsg{awsclient.AllRegionsCodeRepos(context.Background(), client)}
		}
	case viewCodePipeline:
		return m, func() tea.Msg {
			return codePipelineMsg{awsclient.AllRegionsPipelines(context.Background(), client)}
		}
	case viewCodeBuild:
		return m, func() tea.Msg {
			return codeBuildMsg{awsclient.AllRegionsBuildProjects(context.Background(), client)}
		}
	case viewEventBridge:
		return m, func() tea.Msg {
			return ebMsg{awsclient.AllRegionsEBRules(context.Background(), client)}
		}
	case viewWAF:
		return m, func() tea.Msg {
			return wafMsg{awsclient.AllRegionsWAFWebACLs(context.Background(), client)}
		}
	}
	return m, nil
}

// ── view ─────────────────────────────────────────────────────────────────────

func (m model) View() string {
	region := m.client.Config.Region
	profile := m.client.Profile
	if profile == "" {
		profile = "default"
	}

	bar := titleStyle.Render("⬡ awslens") +
		mutedStyle.Render(fmt.Sprintf("  %s  |  %s", profile, region))

	if m.current == viewDashboard {
		return bar + "\n\n" + m.dashboardView()
	}

	svcIdx := m.serviceIndex()
	breadcrumb := mutedStyle.Render("dashboard") + " › " +
		lipgloss.NewStyle().Foreground(orange).Render(menu[svcIdx].label)

	// detail view
	if m.detail.active != detailNone {
		detailLabel := m.detailLabel()
		breadcrumb += " › " + lipgloss.NewStyle().Foreground(cyan).Render(detailLabel)
		body := m.detail.view(m.loading, m.err)
		return bar + "  " + breadcrumb + "\n\n" + body
	}

	body := m.serviceView()
	help := helpStyle.Render("\n↑/↓ navigate • enter drill-down • i AI insight • r refresh • esc/q back" + m.crudHint())
	page := bar + "  " + breadcrumb + "\n\n" + body + help
	if m.modal.active() {
		lines := strings.Split(page, "\n")
		h := len(lines)
		overlay := m.modal.view(m.width)
		olines := strings.Split(overlay, "\n")
		start := (h - len(olines)) / 2
		if start < 0 {
			start = 0
		}
		for i, ol := range olines {
			idx := start + i
			if idx < len(lines) {
				lines[idx] = ol
			} else {
				lines = append(lines, ol)
			}
		}
		return strings.Join(lines, "\n")
	}
	return page
}

func (m model) detailLabel() string {
	switch m.detail.active {
	case detailLambda:
		if m.detail.lambdaDetail != nil {
			return m.detail.lambdaDetail.Name
		}
	case detailLambdaLogs:
		if m.detail.lambdaDetail != nil {
			return m.detail.lambdaDetail.Name + " › logs"
		}
	case detailS3Objects:
		prefix := m.detail.s3Prefix
		if prefix == "" {
			prefix = "/"
		}
		return m.detail.s3Bucket + " › " + prefix
	case detailCFNResources:
		if m.cursor < len(m.stacks) {
			return m.stacks[m.cursor].Name + " › resources"
		}
	case detailCFNEvents:
		if m.cursor < len(m.stacks) {
			return m.stacks[m.cursor].Name + " › events"
		}
	case detailCFNOutputs:
		if m.cursor < len(m.stacks) {
			return m.stacks[m.cursor].Name + " › outputs"
		}
	case detailSNSSubscriptions:
		if m.cursor < len(m.topics) {
			return m.topics[m.cursor].Name + " › subscriptions"
		}
	}
	return "detail"
}

func (m model) serviceIndex() int {
	for i, item := range menu {
		if item.v == m.current {
			return i
		}
	}
	return 0
}

func (m model) dashboardView() string {
	if m.probing {
		return warnStyle.Render("  " + m.spinner.View() + "  Checking service access for this profile...")
	}
	var b strings.Builder
	for i, item := range menu {
		allowed := m.access == nil || m.access[item.label]
		prefix := "  "
		label := fmt.Sprintf("%-16s", item.label)
		desc := mutedStyle.Render(item.desc)

		if !allowed {
			b.WriteString("  " + mutedStyle.Render(fmt.Sprintf("%-16s", item.label)) +
				" " + mutedStyle.Render("no access") + "\n")
			continue
		}
		if i == m.cursor {
			prefix = "▶ "
			label = selectedRow.Render(label)
		}
		b.WriteString(prefix + label + " " + desc + "\n")
	}
	b.WriteString(helpStyle.Render("\n↑/↓ navigate • enter select • ctrl+c quit"))
	return b.String()
}

func (m model) serviceView() string {
	if m.loading {
		return warnStyle.Render("  " + m.spinner.View() + "  Scanning all regions...")
	}
	if m.err != nil {
		return errorStyle.Render("  " + friendlyErr(m.err))
	}
	switch m.current {
	case viewEC2:
		return m.ec2View()
	case viewLambda:
		return m.lambdaView()
	case viewS3:
		return m.s3View()
	case viewRDS:
		return m.rdsView()
	case viewECS:
		return m.ecsView()
	case viewSQS:
		return m.sqsView()
	case viewSNS:
		return m.snsView()
	case viewCFN:
		return m.cfnView()
	case viewCosts:
		return m.costsView()
	case viewDynamo:
		return m.dynamoView()
	case viewAPIGW:
		return m.apigwView()
	case viewECR:
		return m.ecrView()
	case viewSFN:
		return m.sfnView()
	case viewALB:
		return m.albView()
	case viewRoute53:
		return m.route53View()
	case viewSecrets:
		return m.secretsView()
	case viewSSM:
		return m.ssmView()
	case viewCW:
		return m.cwView()
	case viewElastiCache:
		return m.elastiCacheView()
	case viewOpenSearch:
		return m.openSearchView()
	case viewMSK:
		return m.mskView()
	case viewGlue:
		return m.glueView()
	case viewAthena:
		return m.athenaView()
	case viewCodeCommit:
		return m.codeCommitView()
	case viewCodePipeline:
		return m.codePipelineView()
	case viewCodeBuild:
		return m.codeBuildView()
	case viewEventBridge:
		return m.eventBridgeView()
	case viewWAF:
		return m.wafView()
	}
	return ""
}

func stateColor(state string) string {
	switch strings.ToLower(state) {
	case "running", "active", "available", "create_complete", "update_complete":
		return okStyle.Render(state)
	case "stopped", "inactive", "stopping", "not_checked":
		return warnStyle.Render(state)
	case "terminated", "failed", "delete_complete", "rollback_complete":
		return errorStyle.Render(state)
	default:
		return warnStyle.Render(state)
	}
}

func rowLine(cursor, i int, line string) string {
	if i == cursor {
		return "▶ " + selectedRow.Render(line)
	}
	return "  " + line
}

func (m model) ec2View() string {
	if len(m.instances) == 0 {
		return mutedStyle.Render("  No EC2 instances running in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-20s %-12s %-14s %-14s %-16s %s",
		"NAME/ID", "STATE", "TYPE", "REGION", "PUBLIC IP", "LAUNCHED")) + "\n")
	for i, inst := range m.instances {
		name := inst.Name
		if name == "" { name = inst.ID }
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-20s %-12s %-14s %-14s %-16s %s",
			truncate(name, 20), stateColor(inst.State), inst.Type,
			inst.Region, inst.PublicIP, inst.LaunchTime)) + "\n")
	}
	return b.String()
}

func (m model) lambdaView() string {
	if len(m.functions) == 0 {
		return mutedStyle.Render("  No Lambda functions deployed in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-36s %-14s %-14s %6s MB %6s s",
		"NAME", "RUNTIME", "REGION", "MEM", "TIMEOUT")) + "\n")
	for i, fn := range m.functions {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-36s %-14s %-14s %6d %6d",
			truncate(fn.Name, 36), fn.Runtime, fn.Region, fn.Memory, fn.Timeout)) + "\n")
	}
	return b.String()
}

func (m model) s3View() string {
	if len(m.buckets) == 0 {
		return mutedStyle.Render("  No S3 buckets found in this account")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %-16s %s", "BUCKET", "REGION", "CREATED")) + "\n")
	for i, bkt := range m.buckets {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-50s %-16s %s",
			truncate(bkt.Name, 50), bkt.Region, bkt.CreationDate)) + "\n")
	}
	return b.String()
}

func (m model) rdsView() string {
	if len(m.dbs) == 0 {
		return mutedStyle.Render("  No RDS databases found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-20s %-12s %-16s %s",
		"ID", "ENGINE", "STATUS", "CLASS", "MULTI-AZ")) + "\n")
	for i, db := range m.dbs {
		multiAZ := "no"
		if db.MultiAZ {
			multiAZ = okStyle.Render("yes")
		}
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-30s %-20s %-12s %-16s %s",
			truncate(db.ID, 30), truncate(db.Engine, 20),
			stateColor(db.Status), db.Class, multiAZ)) + "\n")
	}
	return b.String()
}

func (m model) ecsView() string {
	if len(m.clusters) == 0 {
		return mutedStyle.Render("  No ECS clusters found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-12s %8s %8s %8s",
		"CLUSTER", "STATUS", "RUNNING", "PENDING", "SERVICES")) + "\n")
	for i, cl := range m.clusters {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-30s %-12s %8d %8d %8d",
			truncate(cl.Name, 30), stateColor(cl.Status),
			cl.RunningTasks, cl.PendingTasks, cl.ActiveServices)) + "\n")
	}
	return b.String()
}

func (m model) sqsView() string {
	if len(m.queues) == 0 {
		return mutedStyle.Render("  No SQS queues found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %10s %10s", "QUEUE", "MESSAGES", "IN-FLIGHT")) + "\n")
	for i, q := range m.queues {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-50s %10s %10s",
			truncate(q.Name, 50), q.Messages, q.MessagesInFlight)) + "\n")
	}
	return b.String()
}

func (m model) snsView() string {
	if len(m.topics) == 0 {
		return mutedStyle.Render("  No SNS topics found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %s", "TOPIC", "ARN")) + "\n")
	for i, t := range m.topics {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-50s %s",
			truncate(t.Name, 50), mutedStyle.Render(truncate(t.ARN, 60)))) + "\n")
	}
	return b.String()
}

func (m model) cfnView() string {
	if len(m.stacks) == 0 {
		return mutedStyle.Render("  No CloudFormation stacks found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-30s %s", "STACK", "STATUS", "DRIFT")) + "\n")
	for i, s := range m.stacks {
		drift := mutedStyle.Render(s.Drift)
		if strings.Contains(s.Drift, "DRIFTED") {
			drift = warnStyle.Render(s.Drift)
		}
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-30s %s",
			truncate(s.Name, 40), stateColor(s.Status), drift)) + "\n")
	}
	return b.String()
}

func (m model) costsView() string {
	if len(m.costs) == 0 {
		return mutedStyle.Render("  No cost data — Cost Explorer may not be enabled for this account")
	}
	sorted := awsclient.SortCostsByAmount(m.costs)

	// find max for bar chart
	var maxAmt float64
	for _, c := range sorted {
		var f float64
		fmt.Sscanf(c.Amount, "%f", &f)
		if f > maxAmt {
			maxAmt = f
		}
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-45s %10s  %s", "SERVICE", "COST (USD)", "")) + "\n")
	for i, c := range sorted {
		var amt float64
		fmt.Sscanf(c.Amount, "%f", &amt)
		bar := awsclient.CostBar(amt, maxAmt, 20)
		barColored := lipgloss.NewStyle().Foreground(orange).Render(bar)
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-45s %10s  %s",
			truncate(c.Service, 45), fmt.Sprintf("$%.4f", amt), barColored)) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(orange).Bold(true).
		Render(fmt.Sprintf("  Total this month: $%s", m.costTotal)) + "\n")
	return b.String()
}

func (m model) dynamoView() string {
	if len(m.tables) == 0 {
		return mutedStyle.Render("  No DynamoDB tables found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-12s %10s %12s %-12s %s",
		"TABLE", "STATUS", "ITEMS", "SIZE", "BILLING", "PK")) + "\n")
	for i, t := range m.tables {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-12s %10d %12s %-12s %s",
			truncate(t.Name, 40), stateColor(t.Status),
			t.ItemCount, humanBytes(t.SizeBytes),
			t.BillingMode, t.PKName)) + "\n")
	}
	return b.String()
}

func (m model) apigwView() string {
	if len(m.apis) == 0 {
		return mutedStyle.Render("  No API Gateway REST APIs found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-12s %-40s %-30s %s",
		"ID", "NAME", "DESCRIPTION", "CREATED")) + "\n")
	for i, a := range m.apis {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-12s %-40s %-30s %s",
			a.ID, truncate(a.Name, 40),
			truncate(a.Description, 30), a.CreatedDate)) + "\n")
	}
	return b.String()
}

func (m model) ecrView() string {
	if len(m.repos) == 0 {
		return mutedStyle.Render("  No ECR repositories found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-10s %-60s %s",
		"NAME", "SCAN", "URI", "CREATED")) + "\n")
	for i, r := range m.repos {
		scan := mutedStyle.Render("off")
		if r.ScanOnPush {
			scan = okStyle.Render("on")
		}
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-10s %-60s %s",
			truncate(r.Name, 40), scan,
			truncate(r.URI, 60), r.Created)) + "\n")
	}
	return b.String()
}

func (m model) sfnView() string {
	if len(m.machines) == 0 {
		return mutedStyle.Render("  No Step Functions state machines found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-10s %s",
		"NAME", "TYPE", "CREATED")) + "\n")
	for i, s := range m.machines {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-10s %s",
			truncate(s.Name, 40), s.Type, s.Created)) + "\n")
	}
	return b.String()
}

func (m model) albView() string {
	if len(m.lbs) == 0 {
		return mutedStyle.Render("  No load balancers found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-6s %-12s %-12s %-50s %s",
		"NAME", "TYPE", "SCHEME", "STATE", "DNS", "CREATED")) + "\n")
	for i, lb := range m.lbs {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-30s %-6s %-12s %-12s %-50s %s",
			truncate(lb.Name, 30), lb.Type, lb.Scheme,
			stateColor(lb.State), truncate(lb.DNS, 50), lb.Created)) + "\n")
	}
	return b.String()
}

func (m model) route53View() string {
	if len(m.zones) == 0 {
		return mutedStyle.Render("  No Route53 hosted zones found in this account")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %-10s %s",
		"NAME", "TYPE", "RECORDS")) + "\n")
	for i, z := range m.zones {
		ztype := okStyle.Render("public")
		if z.Private {
			ztype = mutedStyle.Render("private")
		}
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-50s %-10s %d",
			truncate(z.Name, 50), ztype, z.Records)) + "\n")
	}
	return b.String()
}

func (m model) secretsView() string {
	if len(m.secrets) == 0 {
		return mutedStyle.Render("  No secrets found in Secrets Manager")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-30s %-12s %s",
		"NAME", "DESCRIPTION", "CHANGED", "ACCESSED")) + "\n")
	for i, s := range m.secrets {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-30s %-12s %s",
			truncate(s.Name, 40), truncate(s.Description, 30),
			s.LastChanged, s.LastAccessed)) + "\n")
	}
	return b.String()
}

func (m model) ssmView() string {
	if len(m.params) == 0 {
		return mutedStyle.Render("  No SSM parameters found in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %-16s %-12s %s",
		"NAME", "TYPE", "MODIFIED", "VERSION")) + "\n")
	for i, p := range m.params {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-50s %-16s %-12s %d",
			truncate(p.Name, 50), p.Type, p.LastModified, p.Version)) + "\n")
	}
	return b.String()
}

func (m model) cwView() string {
	if len(m.alarms) == 0 {
		return mutedStyle.Render("  No CloudWatch alarms configured in this region")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-12s %-30s %-20s %-12s %s",
		"ALARM", "STATE", "METRIC", "NAMESPACE", "THRESHOLD", "UPDATED")) + "\n")
	for i, a := range m.alarms {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-12s %-30s %-20s %-12s %s",
			truncate(a.Name, 40), alarmStateColor(a.State),
			truncate(a.Metric, 30), truncate(a.Namespace, 20),
			a.Threshold, a.Updated)) + "\n")
	}
	return b.String()
}

func (m model) elastiCacheView() string {
	if len(m.cacheClusters) == 0 { return mutedStyle.Render("  No ElastiCache clusters found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-20s %-12s %-20s %-6s %s", "ID", "ENGINE", "STATUS", "NODE TYPE", "NODES", "REGION")) + "\n")
	for i, c := range m.cacheClusters {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-30s %-20s %-12s %-20s %-6d %s",
			truncate(c.ID, 30), truncate(c.Engine, 20), stateColor(c.Status), c.NodeType, c.Nodes, c.Region)) + "\n")
	}
	return b.String()
}

func (m model) openSearchView() string {
	if len(m.osDomains) == 0 { return mutedStyle.Render("  No OpenSearch domains found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %s", "DOMAIN", "REGION")) + "\n")
	for i, d := range m.osDomains {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %s", truncate(d.Name, 40), d.Region)) + "\n")
	}
	return b.String()
}

func (m model) mskView() string {
	if len(m.mskClusters) == 0 { return mutedStyle.Render("  No MSK clusters found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-12s %-12s %-6s %s", "NAME", "STATE", "VERSION", "BROKERS", "REGION")) + "\n")
	for i, c := range m.mskClusters {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-12s %-12s %-6d %s",
			truncate(c.Name, 40), stateColor(c.State), c.Version, c.Brokers, c.Region)) + "\n")
	}
	return b.String()
}

func (m model) glueView() string {
	if len(m.glueDbs) == 0 { return mutedStyle.Render("  No Glue databases found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-40s %s", "DATABASE", "DESCRIPTION", "REGION")) + "\n")
	for i, d := range m.glueDbs {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-40s %s",
			truncate(d.Name, 40), truncate(d.Description, 40), d.Region)) + "\n")
	}
	return b.String()
}

func (m model) athenaView() string {
	if len(m.athenaWGs) == 0 { return mutedStyle.Render("  No Athena workgroups found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-12s %-40s %s", "WORKGROUP", "STATE", "DESCRIPTION", "REGION")) + "\n")
	for i, wg := range m.athenaWGs {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-30s %-12s %-40s %s",
			truncate(wg.Name, 30), stateColor(wg.State), truncate(wg.Description, 40), wg.Region)) + "\n")
	}
	return b.String()
}

func (m model) codeCommitView() string {
	if len(m.codeRepos) == 0 { return mutedStyle.Render("  No CodeCommit repositories found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-30s %-12s %s", "REPO", "DESCRIPTION", "MODIFIED", "REGION")) + "\n")
	for i, r := range m.codeRepos {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-30s %-12s %s",
			truncate(r.Name, 40), truncate(r.Description, 30), r.LastModified, r.Region)) + "\n")
	}
	return b.String()
}

func (m model) codePipelineView() string {
	if len(m.pipelines) == 0 { return mutedStyle.Render("  No CodePipeline pipelines found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-6s %-20s %s", "PIPELINE", "VER", "UPDATED", "REGION")) + "\n")
	for i, p := range m.pipelines {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-6d %-20s %s",
			truncate(p.Name, 40), p.Version, p.Updated, p.Region)) + "\n")
	}
	return b.String()
}

func (m model) codeBuildView() string {
	if len(m.buildProjects) == 0 { return mutedStyle.Render("  No CodeBuild projects found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-30s %-20s %s", "PROJECT", "DESCRIPTION", "LAST BUILD", "REGION")) + "\n")
	for i, p := range m.buildProjects {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-30s %-20s %s",
			truncate(p.Name, 40), truncate(p.Description, 30), p.LastBuild, p.Region)) + "\n")
	}
	return b.String()
}

func (m model) eventBridgeView() string {
	if len(m.ebRules) == 0 { return mutedStyle.Render("  No EventBridge rules found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-10s %-30s %-30s %s", "RULE", "STATE", "SCHEDULE", "DESCRIPTION", "REGION")) + "\n")
	for i, r := range m.ebRules {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-10s %-30s %-30s %s",
			truncate(r.Name, 40), stateColor(r.State),
			truncate(r.Schedule, 30), truncate(r.Description, 30), r.Region)) + "\n")
	}
	return b.String()
}

func (m model) wafView() string {
	if len(m.wafACLs) == 0 { return mutedStyle.Render("  No WAF Web ACLs found") }
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-10s %-6s %s", "NAME", "SCOPE", "RULES", "REGION")) + "\n")
	for i, a := range m.wafACLs {
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-40s %-10s %-6d %s",
			truncate(a.Name, 40), a.Scope, a.Rules, a.Region)) + "\n")
	}
	return b.String()
}

func alarmStateColor(state string) string {
	switch state {
	case "OK":
		return okStyle.Render(state)
	case "ALARM":
		return errorStyle.Render(state)
	default:
		return warnStyle.Render(state)
	}
}

func friendlyErr(err error) string {
	if err == nil {
		return ""
	}
	s := err.Error()
	if strings.Contains(s, "StatusCode: 403") || strings.Contains(s, "AccessDenied") || strings.Contains(s, "UnauthorizedOperation") {
		return "⚠  This profile doesn't have permission to access this service.\n\n" +
			"  Try switching to a profile with broader permissions (e.g. AdminRole).\n" +
			"  Press esc to go back and choose a different service or profile."
	}
	if strings.Contains(s, "StatusCode: 404") || strings.Contains(s, "NoSuchEntity") {
		return "⚠  Resource not found — it may have been deleted or is in a different region."
	}
	if strings.Contains(s, "no such host") || strings.Contains(s, "connection refused") || strings.Contains(s, "dial tcp") {
		return "⚠  Can't reach AWS — check your internet connection or VPN."
	}
	if strings.Contains(s, "ExpiredToken") || strings.Contains(s, "ExpiredTokenException") {
		return "⚠  Your AWS credentials have expired.\n\n" +
			"  Run: aws sso login --profile <name>  or  aws configure\n" +
			"  Then restart awslens."
	}
	if strings.Contains(s, "NoCredentialProviders") || strings.Contains(s, "no EC2 IMDS role found") {
		return "⚠  No AWS credentials found for this profile.\n\n" +
			"  Run: aws configure --profile <name>"
	}
	return "⚠  " + s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// ── CRUD actions ─────────────────────────────────────────────────────────────

func (m model) crudHint() string {
	switch m.current {
	case viewS3:
		return " • n new bucket • d delete bucket"
	case viewLambda:
		return " • d delete function"
	case viewDynamo:
		return " • d delete table"
	case viewSQS:
		return " • n new queue • p purge • d delete"
	case viewSNS:
		return " • n new topic • d delete"
	case viewSSM:
		return " • n new param • e edit value • d delete"
	case viewSecrets:
		return " • n new secret • e update value • d delete"
	case viewECR:
		return " • n new repo • d delete repo"
	case viewCodeCommit:
		return " • n new repo • d delete repo"
	case viewEventBridge:
		return " • t toggle enable/disable • d delete rule"
	}
	return ""
}

func writeCmd(fn func() error) tea.Cmd {
	return func() tea.Msg {
		if err := fn(); err != nil {
			return writeErrMsg{err}
		}
		return writeOKMsg{}
	}
}

func (m model) handleNew() (model, tea.Cmd) {
	switch m.current {
	case viewS3:
		m.modal = modal{kind: modalInput, title: "Create S3 Bucket", body: "Bucket name:"}
		m.modalOK = func() tea.Cmd {
			name := m.modal.input
			return writeCmd(func() error { return m.client.CreateBucket(context.Background(), name) })
		}
	case viewSQS:
		m.modal = modal{kind: modalInput, title: "Create SQS Queue", body: "Queue name:"}
		m.modalOK = func() tea.Cmd {
			name := m.modal.input
			return writeCmd(func() error { return m.client.CreateQueue(context.Background(), name) })
		}
	case viewSNS:
		m.modal = modal{kind: modalInput, title: "Create SNS Topic", body: "Topic name:"}
		m.modalOK = func() tea.Cmd {
			name := m.modal.input
			return writeCmd(func() error { return m.client.CreateTopic(context.Background(), name) })
		}
	case viewSSM:
		m.modal = modal{kind: modalInput, title: "New SSM Parameter", body: "name=value (e.g. /app/key=secret):"}
		m.modalOK = func() tea.Cmd {
			input := m.modal.input
			return writeCmd(func() error {
				kv := splitKV(input)
				if len(kv) != 2 {
					return fmt.Errorf("use format: /name=value")
				}
				return m.client.PutSSMParam(context.Background(), kv[0], kv[1], false)
			})
		}
	case viewSecrets:
		m.modal = modal{kind: modalInput, title: "Create Secret", body: "name=value:"}
		m.modalOK = func() tea.Cmd {
			input := m.modal.input
			return writeCmd(func() error {
				kv := splitKV(input)
				if len(kv) != 2 {
					return fmt.Errorf("use format: name=value")
				}
				return m.client.CreateSecret(context.Background(), kv[0], kv[1])
			})
		}
	case viewECR:
		m.modal = modal{kind: modalInput, title: "Create ECR Repository", body: "Repository name:"}
		m.modalOK = func() tea.Cmd {
			name := m.modal.input
			return writeCmd(func() error { return m.client.CreateECRRepo(context.Background(), name) })
		}
	case viewCodeCommit:
		m.modal = modal{kind: modalInput, title: "Create CodeCommit Repo", body: "Repository name:"}
		m.modalOK = func() tea.Cmd {
			name := m.modal.input
			return writeCmd(func() error { return m.client.CreateCodeRepo(context.Background(), name) })
		}
	}
	return m, nil
}

func (m model) handleDelete() (model, tea.Cmd) {
	switch m.current {
	case viewS3:
		if m.cursor >= len(m.buckets) {
			return m, nil
		}
		name := m.buckets[m.cursor].Name
		m.modal = modal{kind: modalConfirm, title: "Delete S3 Bucket", body: fmt.Sprintf("Delete bucket %q?\n(must be empty)", name)}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteBucket(context.Background(), name) })
		}
	case viewLambda:
		if m.cursor >= len(m.functions) {
			return m, nil
		}
		fn := m.functions[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete Lambda Function", body: fmt.Sprintf("Delete function %q?", fn.Name)}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(fn.Region)
			return writeCmd(func() error { return client.DeleteFunction(context.Background(), fn.Name) })
		}
	case viewDynamo:
		if m.cursor >= len(m.tables) {
			return m, nil
		}
		tbl := m.tables[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete DynamoDB Table", body: fmt.Sprintf("Delete table %q?\n⚠ This deletes ALL data!", tbl.Name)}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(tbl.Region)
			return writeCmd(func() error { return client.DeleteTable(context.Background(), tbl.Name) })
		}
	case viewSQS:
		if m.cursor >= len(m.queues) {
			return m, nil
		}
		q := m.queues[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete SQS Queue", body: fmt.Sprintf("Delete queue %q?", q.Name)}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteQueue(context.Background(), q.URL) })
		}
	case viewSNS:
		if m.cursor >= len(m.topics) {
			return m, nil
		}
		t := m.topics[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete SNS Topic", body: fmt.Sprintf("Delete topic %q?", t.Name)}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteTopic(context.Background(), t.ARN) })
		}
	case viewSSM:
		if m.cursor >= len(m.params) {
			return m, nil
		}
		p := m.params[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete SSM Parameter", body: fmt.Sprintf("Delete parameter %q?", p.Name)}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteSSMParam(context.Background(), p.Name) })
		}
	case viewSecrets:
		if m.cursor >= len(m.secrets) {
			return m, nil
		}
		s := m.secrets[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete Secret", body: fmt.Sprintf("Delete secret %q?\n(7-day recovery window)", s.Name)}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteSecret(context.Background(), s.ARN) })
		}
	case viewECR:
		if m.cursor >= len(m.repos) {
			return m, nil
		}
		r := m.repos[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete ECR Repository", body: fmt.Sprintf("Delete repo %q?\n⚠ All images will be deleted!", r.Name)}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(r.Region)
			return writeCmd(func() error { return client.DeleteECRRepo(context.Background(), r.Name) })
		}
	case viewCodeCommit:
		if m.cursor >= len(m.codeRepos) {
			return m, nil
		}
		r := m.codeRepos[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete CodeCommit Repo", body: fmt.Sprintf("Delete repo %q?", r.Name)}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(r.Region)
			return writeCmd(func() error { return client.DeleteCodeRepo(context.Background(), r.Name) })
		}
	case viewEventBridge:
		if m.cursor >= len(m.ebRules) {
			return m, nil
		}
		r := m.ebRules[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete EventBridge Rule", body: fmt.Sprintf("Delete rule %q?\n(targets will be removed first)", r.Name)}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(r.Region)
			return writeCmd(func() error { return client.DeleteEBRule(context.Background(), r.Name) })
		}
	}
	return m, nil
}

func (m model) handleEdit() (model, tea.Cmd) {
	switch m.current {
	case viewSSM:
		if m.cursor >= len(m.params) {
			return m, nil
		}
		p := m.params[m.cursor]
		m.modal = modal{kind: modalInput, title: "Update SSM Parameter", body: fmt.Sprintf("New value for %q:", p.Name)}
		m.modalOK = func() tea.Cmd {
			val := m.modal.input
			return writeCmd(func() error { return m.client.PutSSMParam(context.Background(), p.Name, val, true) })
		}
	case viewSecrets:
		if m.cursor >= len(m.secrets) {
			return m, nil
		}
		s := m.secrets[m.cursor]
		m.modal = modal{kind: modalInput, title: "Update Secret Value", body: fmt.Sprintf("New value for %q:", s.Name)}
		m.modalOK = func() tea.Cmd {
			val := m.modal.input
			return writeCmd(func() error { return m.client.UpdateSecret(context.Background(), s.ARN, val) })
		}
	}
	return m, nil
}

func (m model) handlePurge() (model, tea.Cmd) {
	if m.current != viewSQS || m.cursor >= len(m.queues) {
		return m, nil
	}
	q := m.queues[m.cursor]
	m.modal = modal{kind: modalConfirm, title: "Purge SQS Queue", body: fmt.Sprintf("Purge ALL messages from %q?", q.Name)}
	m.modalOK = func() tea.Cmd {
		return writeCmd(func() error { return m.client.PurgeQueue(context.Background(), q.URL) })
	}
	return m, nil
}

func (m model) handleToggle() (model, tea.Cmd) {
	if m.current != viewEventBridge || m.cursor >= len(m.ebRules) {
		return m, nil
	}
	r := m.ebRules[m.cursor]
	action := "Disable"
	if r.State == "DISABLED" {
		action = "Enable"
	}
	m.modal = modal{kind: modalConfirm, title: action + " EventBridge Rule", body: fmt.Sprintf("%s rule %q?", action, r.Name)}
	m.modalOK = func() tea.Cmd {
		client := m.client.NewForRegion(r.Region)
		return writeCmd(func() error {
			if r.State == "DISABLED" {
				return client.EnableEBRule(context.Background(), r.Name)
			}
			return client.DisableEBRule(context.Background(), r.Name)
		})
	}
	return m, nil
}

func (m model) handleInsight() (model, tea.Cmd) {
	summary := m.summarizeSelected()
	if summary == "" {
		return m, nil
	}
	tokens := len(summary)/4 + 50 // rough input estimate + system prompt
	cost := float64(tokens)*0.00000025 + 200*0.00000125 // haiku input + output pricing
	m.modal = modal{
		kind:  modalConfirm,
		title: "✨ AI Insight (Bedrock)",
		body:  fmt.Sprintf("Analyze this resource using Claude 3 Haiku?\n\nEstimated cost: ~$%.5f\n(~%d input + 200 output tokens)", cost, tokens),
	}
	client := m.client
	m.modalOK = func() tea.Cmd {
		return func() tea.Msg {
			text, err := client.GetInsight(context.Background(), summary)
			if err != nil {
				return errMsg{err}
			}
			return insightMsg{text}
		}
	}
	return m, nil
}

func (m model) summarizeSelected() string {
	switch m.current {
	case viewEC2:
		if m.cursor >= len(m.instances) { return "" }
		i := m.instances[m.cursor]
		return fmt.Sprintf("EC2: id=%s type=%s state=%s az=%s publicIP=%s launched=%s", i.ID, i.Type, i.State, i.AZ, i.PublicIP, i.LaunchTime)
	case viewLambda:
		if m.cursor >= len(m.functions) { return "" }
		f := m.functions[m.cursor]
		return fmt.Sprintf("Lambda: name=%s runtime=%s memory=%dMB timeout=%ds handler=%s modified=%s", f.Name, f.Runtime, f.Memory, f.Timeout, f.Handler, f.LastModified)
	case viewS3:
		if m.cursor >= len(m.buckets) { return "" }
		b := m.buckets[m.cursor]
		return fmt.Sprintf("S3: name=%s region=%s created=%s", b.Name, b.Region, b.CreationDate)
	case viewRDS:
		if m.cursor >= len(m.dbs) { return "" }
		d := m.dbs[m.cursor]
		return fmt.Sprintf("RDS: id=%s engine=%s class=%s status=%s multiAZ=%v", d.ID, d.Engine, d.Class, d.Status, d.MultiAZ)
	case viewDynamo:
		if m.cursor >= len(m.tables) { return "" }
		t := m.tables[m.cursor]
		return fmt.Sprintf("DynamoDB: table=%s status=%s items=%d size=%d billing=%s pk=%s", t.Name, t.Status, t.ItemCount, t.SizeBytes, t.BillingMode, t.PKName)
	case viewECS:
		if m.cursor >= len(m.clusters) { return "" }
		c := m.clusters[m.cursor]
		return fmt.Sprintf("ECS: cluster=%s status=%s running=%d pending=%d services=%d", c.Name, c.Status, c.RunningTasks, c.PendingTasks, c.ActiveServices)
	case viewSQS:
		if m.cursor >= len(m.queues) { return "" }
		q := m.queues[m.cursor]
		return fmt.Sprintf("SQS: name=%s messages=%s inflight=%s", q.Name, q.Messages, q.MessagesInFlight)
	case viewSNS:
		if m.cursor >= len(m.topics) { return "" }
		t := m.topics[m.cursor]
		return fmt.Sprintf("SNS: topic=%s arn=%s", t.Name, t.ARN)
	case viewCFN:
		if m.cursor >= len(m.stacks) { return "" }
		s := m.stacks[m.cursor]
		return fmt.Sprintf("CloudFormation: stack=%s status=%s drift=%s", s.Name, s.Status, s.Drift)
	case viewALB:
		if m.cursor >= len(m.lbs) { return "" }
		l := m.lbs[m.cursor]
		return fmt.Sprintf("LoadBalancer: name=%s type=%s scheme=%s state=%s", l.Name, l.Type, l.Scheme, l.State)
	case viewCW:
		if m.cursor >= len(m.alarms) { return "" }
		a := m.alarms[m.cursor]
		return fmt.Sprintf("CloudWatch: alarm=%s state=%s metric=%s namespace=%s threshold=%s", a.Name, a.State, a.Metric, a.Namespace, a.Threshold)
	case viewSecrets:
		if m.cursor >= len(m.secrets) { return "" }
		s := m.secrets[m.cursor]
		return fmt.Sprintf("Secret: name=%s lastChanged=%s lastAccessed=%s", s.Name, s.LastChanged, s.LastAccessed)
	case viewSSM:
		if m.cursor >= len(m.params) { return "" }
		p := m.params[m.cursor]
		return fmt.Sprintf("SSM: name=%s type=%s version=%d modified=%s", p.Name, p.Type, p.Version, p.LastModified)
	case viewECR:
		if m.cursor >= len(m.repos) { return "" }
		r := m.repos[m.cursor]
		return fmt.Sprintf("ECR: name=%s scanOnPush=%v created=%s", r.Name, r.ScanOnPush, r.Created)
	case viewRoute53:
		if m.cursor >= len(m.zones) { return "" }
		z := m.zones[m.cursor]
		return fmt.Sprintf("Route53: zone=%s private=%v records=%d", z.Name, z.Private, z.Records)
	}
	return ""
}

func splitKV(s string) []string {
	for i, r := range s {
		if r == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

// ── entry ────────────────────────────────────────────────────────────────────

func Start(profile, region string) error {
	client, err := awsclient.New(profile, region)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(orange)
	_, err = tea.NewProgram(model{client: client, probing: true, spinner: s}, tea.WithAltScreen()).Run()
	return err
}
