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
	scroll  int // scroll offset for viewport
	filter  string // active filter text
	filtering bool // true when typing filter
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
		// ensure cursor is on an accessible item
		if !m.access[menu[m.cursor].label] {
			for i, item := range menu {
				if m.access[item.label] {
					m.cursor = i
					break
				}
			}
		}
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
	case "s":
		m.detail.masked = !m.detail.masked
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
	// filter input mode
	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filter = ""
			m.cursor = 0
			m.scroll = 0
		case "enter":
			m.filtering = false
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.cursor = 0
				m.scroll = 0
			}
		default:
			if len(msg.String()) == 1 {
				m.filter += msg.String()
				m.cursor = 0
				m.scroll = 0
			}
		}
		return m, nil
	}

	maxC := m.maxCursor()
	switch msg.String() {
	case "q", "esc":
		if m.filter != "" {
			m.filter = ""
			m.cursor = 0
			m.scroll = 0
			return m, nil
		}
		m.current = viewDashboard
		m.cursor = 0
		m.scroll = 0
		m.err = nil
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.adjustScroll()
	case "down", "j":
		if m.cursor < maxC {
			m.cursor++
		}
		m.adjustScroll()
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
	case "/":
		m.filtering = true
		m.filter = ""
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
	m.scroll = 0
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
	// apply filter
	if m.filter != "" {
		bodyLines := strings.Split(body, "\n")
		var filtered []string
		lowerFilter := strings.ToLower(m.filter)
		for i, line := range bodyLines {
			if i == 0 || strings.Contains(strings.ToLower(line), lowerFilter) {
				filtered = append(filtered, line)
			}
		}
		body = strings.Join(filtered, "\n")
	}
	// apply scroll viewport
	bodyLines := strings.Split(body, "\n")
	visible := m.height - 6
	if visible < 5 {
		visible = 5
	}
	if len(bodyLines) > visible {
		bodyLines = scrollLines(bodyLines, m.scroll, visible)
		body = strings.Join(bodyLines, "\n")
	}
	help := helpStyle.Render("\n↑/↓ navigate • enter drill-down • i AI insight • / filter • r refresh • esc/q back" + m.crudHint())
	if m.filtering {
		help = warnStyle.Render("\n  filter: " + m.filter + "█  (enter confirm • esc clear)")
	} else if m.filter != "" {
		help = helpStyle.Render(fmt.Sprintf("\n  filtered: %q • esc clear • / change", m.filter)) +
			helpStyle.Render(" • ↑/↓ navigate • enter drill-down")
	}
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


func (m *model) adjustScroll() {
	visible := m.height - 6 // header + help lines
	if visible < 5 {
		visible = 5
	}
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+visible {
		m.scroll = m.cursor - visible + 1
	}
}

func scrollLines(lines []string, scroll, visible int) []string {
	if scroll >= len(lines) {
		return nil
	}
	end := scroll + visible
	if end > len(lines) {
		end = len(lines)
	}
	return lines[scroll:end]
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
