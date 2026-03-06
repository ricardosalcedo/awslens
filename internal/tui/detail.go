package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	awsclient "github.com/awslens/awslens/internal/aws"
)

// ── detail view enum ──────────────────────────────────────────────────────────

type detailView int

const (
	detailNone detailView = iota
	detailLambda
	detailLambdaLogs
	detailS3Objects
	detailCFNResources
	detailCFNEvents
	detailCFNOutputs
	detailSNSSubscriptions
	detailCosts
	detailDynamoItems
	detailAPIResources
	detailECRImages
	detailSFNExecutions
	detailDNSRecords
	detailAlarmHistory
	detailSSMValue
)

// ── detail messages ───────────────────────────────────────────────────────────

type lambdaDetailMsg struct{ d *awsclient.FunctionDetail }
type lambdaLogsMsg struct{ lines []string }
type s3ObjectsMsg struct {
	bucket  string
	prefix  string
	objects []awsclient.S3Object
}
type cfnResourcesMsg struct{ resources []awsclient.StackResource }
type cfnEventsMsg struct{ events []awsclient.StackEvent }
type cfnOutputsMsg struct{ outputs []awsclient.StackOutput }
type snsSubsMsg struct{ subs []awsclient.Subscription }

type dynamoItemsMsg struct{ items []map[string]string }
type apiResourcesMsg struct{ resources []awsclient.APIResource }
type ecrImagesMsg struct{ images []awsclient.ECRImage }
type sfnExecutionsMsg struct{ execs []awsclient.SFNExecution }
type dnsRecordsMsg struct{ records []awsclient.DNSRecord }
type alarmHistoryMsg struct{ lines []string }
type ssmValueMsg struct{ name, value string }
// (embedded into main model via detailModel)

type detailModel struct {
	active       detailView
	cursor       int
	parentIdx    int // index of the parent resource in the service list
	lambdaDetail *awsclient.FunctionDetail
	lambdaLogs   []string
	s3Bucket     string
	s3Prefix     string
	s3Objects    []awsclient.S3Object
	cfnResources []awsclient.StackResource
	cfnEvents    []awsclient.StackEvent
	cfnOutputs   []awsclient.StackOutput
	snsSubs      []awsclient.Subscription
	dynamoTable  string
	dynamoItems  []map[string]string
	apiResources []awsclient.APIResource
	ecrImages    []awsclient.ECRImage
	sfnExecs     []awsclient.SFNExecution
	dnsRecords   []awsclient.DNSRecord
	alarmHistory []string
	ssmParamName string
	ssmValue     string
	masked       bool // true = hide sensitive values (default)
	// tab within a detail view (e.g. lambda: config/env/logs)
	tab int
}

func (d *detailModel) reset() {
	*d = detailModel{masked: true}
}

// ── detail update (called from main model.Update) ─────────────────────────────

func handleDetailMsg(d *detailModel, msg tea.Msg) bool {
	switch msg := msg.(type) {
	case lambdaDetailMsg:
		d.lambdaDetail = msg.d
		return true
	case lambdaLogsMsg:
		d.lambdaLogs = msg.lines
		return true
	case s3ObjectsMsg:
		d.s3Bucket = msg.bucket
		d.s3Prefix = msg.prefix
		d.s3Objects = msg.objects
		return true
	case cfnResourcesMsg:
		d.cfnResources = msg.resources
		return true
	case cfnEventsMsg:
		d.cfnEvents = msg.events
		return true
	case cfnOutputsMsg:
		d.cfnOutputs = msg.outputs
		return true
	case snsSubsMsg:
		d.snsSubs = msg.subs
		return true
	case dynamoItemsMsg:
		d.dynamoItems = msg.items
		return true
	case apiResourcesMsg:
		d.apiResources = msg.resources
		return true
	case ecrImagesMsg:
		d.ecrImages = msg.images
		return true
	case sfnExecutionsMsg:
		d.sfnExecs = msg.execs
		return true
	case dnsRecordsMsg:
		d.dnsRecords = msg.records
		return true
	case alarmHistoryMsg:
		d.alarmHistory = msg.lines
		return true
	case ssmValueMsg:
		d.ssmParamName = msg.name
		d.ssmValue = msg.value
		return true
	}
	return false
}

// ── fetch commands ────────────────────────────────────────────────────────────

func fetchLambdaDetail(client *awsclient.Client, name string) tea.Cmd {
	return func() tea.Msg {
		d, err := client.GetFunctionDetail(context.Background(), name)
		if err != nil {
			return errMsg{err}
		}
		return lambdaDetailMsg{d}
	}
}

func fetchLambdaLogs(client *awsclient.Client, name string) tea.Cmd {
	return func() tea.Msg {
		lines, err := client.GetFunctionLogs(context.Background(), name, 50)
		if err != nil {
			return errMsg{err}
		}
		return lambdaLogsMsg{lines}
	}
}

func fetchS3Objects(client *awsclient.Client, bucket, prefix string) tea.Cmd {
	return func() tea.Msg {
		objects, err := client.ListObjects(context.Background(), bucket, prefix)
		if err != nil {
			return errMsg{err}
		}
		return s3ObjectsMsg{bucket, prefix, objects}
	}
}

func fetchCFNResources(client *awsclient.Client, stackName string) tea.Cmd {
	return func() tea.Msg {
		r, err := client.GetStackResources(context.Background(), stackName)
		if err != nil {
			return errMsg{err}
		}
		return cfnResourcesMsg{r}
	}
}

func fetchCFNEvents(client *awsclient.Client, stackName string) tea.Cmd {
	return func() tea.Msg {
		e, err := client.GetStackEvents(context.Background(), stackName)
		if err != nil {
			return errMsg{err}
		}
		return cfnEventsMsg{e}
	}
}

func fetchCFNOutputs(client *awsclient.Client, stackName string) tea.Cmd {
	return func() tea.Msg {
		o, err := client.GetStackOutputs(context.Background(), stackName)
		if err != nil {
			return errMsg{err}
		}
		return cfnOutputsMsg{o}
	}
}

func fetchSNSSubs(client *awsclient.Client, arn string) tea.Cmd {
	return func() tea.Msg {
		s, err := client.GetTopicSubscriptions(context.Background(), arn)
		if err != nil {
			return errMsg{err}
		}
		return snsSubsMsg{s}
	}
}

func fetchDynamoItems(client *awsclient.Client, table string) tea.Cmd {
	return func() tea.Msg {
		items, err := client.ScanDynamoTable(context.Background(), table, 50)
		if err != nil {
			return errMsg{err}
		}
		return dynamoItemsMsg{items}
	}
}

func fetchAPIResources(client *awsclient.Client, apiID string) tea.Cmd {
	return func() tea.Msg {
		resources, err := client.GetAPIResources(context.Background(), apiID)
		if err != nil {
			return errMsg{err}
		}
		return apiResourcesMsg{resources}
	}
}

func fetchECRImages(client *awsclient.Client, repo string) tea.Cmd {
	return func() tea.Msg {
		images, err := client.ListECRImages(context.Background(), repo)
		if err != nil {
			return errMsg{err}
		}
		return ecrImagesMsg{images}
	}
}

func fetchSFNExecutions(client *awsclient.Client, smARN string) tea.Cmd {
	return func() tea.Msg {
		execs, err := client.ListSFNExecutions(context.Background(), smARN)
		if err != nil {
			return errMsg{err}
		}
		return sfnExecutionsMsg{execs}
	}
}

func fetchDNSRecords(client *awsclient.Client, zoneID string) tea.Cmd {
	return func() tea.Msg {
		records, err := client.ListDNSRecords(context.Background(), zoneID)
		if err != nil {
			return errMsg{err}
		}
		return dnsRecordsMsg{records}
	}
}

func fetchAlarmHistory(client *awsclient.Client, alarmName string) tea.Cmd {
	return func() tea.Msg {
		lines, err := client.GetAlarmHistory(context.Background(), alarmName)
		if err != nil {
			return errMsg{err}
		}
		return alarmHistoryMsg{lines}
	}
}

func fetchSSMValue(client *awsclient.Client, name string) tea.Cmd {
	return func() tea.Msg {
		val, err := client.GetSSMParamValue(context.Background(), name)
		if err != nil {
			return errMsg{err}
		}
		return ssmValueMsg{name, val}
	}
}

// ── detail views ──────────────────────────────────────────────────────────────

func (d *detailModel) view(loading bool, err error) string {
	if loading {
		return warnStyle.Render("  ⟳  Fetching from AWS...")
	}
	if err != nil {
		return errorStyle.Render("  " + friendlyErr(err))
	}
	switch d.active {
	case detailLambda:
		return d.lambdaDetailView()
	case detailLambdaLogs:
		return d.lambdaLogsView()
	case detailS3Objects:
		return d.s3ObjectsView()
	case detailCFNResources:
		return d.cfnResourcesView()
	case detailCFNEvents:
		return d.cfnEventsView()
	case detailCFNOutputs:
		return d.cfnOutputsView()
	case detailSNSSubscriptions:
		return d.snsSubsView()
	case detailDynamoItems:
		return d.dynamoItemsView()
	case detailAPIResources:
		return d.apiResourcesView()
	case detailECRImages:
		return d.ecrImagesView()
	case detailSFNExecutions:
		return d.sfnExecutionsView()
	case detailDNSRecords:
		return d.dnsRecordsView()
	case detailAlarmHistory:
		return d.alarmHistoryView()
	case detailSSMValue:
		return d.ssmValueView()
	}
	return ""
}

func (d *detailModel) lambdaDetailView() string {
	if d.lambdaDetail == nil {
		return warnStyle.Render("  ⟳  Fetching from AWS...")
	}
	fn := d.lambdaDetail
	tabs := []string{"Config", "Env Vars", "Triggers"}
	tabBar := renderTabs(tabs, d.tab)

	var body string
	switch d.tab {
	case 0: // Config
		body = fmt.Sprintf(`
  %-16s %s
  %-16s %s
  %-16s %d MB
  %-16s %d s
  %-16s %s
  %-16s %s
  %-16s %s
`,
			"Handler:", fn.Handler,
			"Runtime:", fn.Runtime,
			"Memory:", fn.Memory,
			"Timeout:", fn.Timeout,
			"Code Size:", humanBytes(fn.CodeSize),
			"Log Group:", fn.LogGroup,
			"Last Modified:", fn.LastModified,
		)
		if fn.Description != "" {
			body += fmt.Sprintf("  %-16s %s\n", "Description:", fn.Description)
		}
	case 1: // Env Vars
		if len(fn.EnvVars) == 0 {
			body = mutedStyle.Render("\n  no environment variables")
		} else {
			var lines []string
			for k, v := range fn.EnvVars {
				if d.masked {
					v = "••••••••"
				}
				lines = append(lines, fmt.Sprintf("  %-40s = %s", k, v))
			}
			body = "\n" + strings.Join(lines, "\n")
			body += "\n" + helpStyle.Render("\n  s toggle mask")
		}
	case 2: // Triggers
		if len(fn.Triggers) == 0 {
			body = mutedStyle.Render("\n  no event source mappings")
		} else {
			var lines []string
			for _, t := range fn.Triggers {
				lines = append(lines, "  "+t)
			}
			body = "\n" + strings.Join(lines, "\n")
		}
	}

	help := helpStyle.Render("\n←/→ switch tabs • l view logs • esc back")
	return tabBar + body + help
}

func (d *detailModel) lambdaLogsView() string {
	if len(d.lambdaLogs) == 0 {
		return mutedStyle.Render("  No log events found — function may not have been invoked recently")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render("  TIME      MESSAGE") + "\n")
	start := 0
	if len(d.lambdaLogs) > 40 {
		start = len(d.lambdaLogs) - 40
	}
	for _, line := range d.lambdaLogs[start:] {
		b.WriteString("  " + line + "\n")
	}
	return b.String() + helpStyle.Render("\nesc back")
}

func (d *detailModel) s3ObjectsView() string {
	prefix := d.s3Prefix
	if prefix == "" {
		prefix = "/"
	}
	header := headerStyle.Render(fmt.Sprintf("  %-60s %10s %-20s %s",
		"KEY", "SIZE", "MODIFIED", "CLASS")) + "\n"
	if len(d.s3Objects) == 0 {
		return header + mutedStyle.Render("  empty")
	}
	var b strings.Builder
	b.WriteString(header)
	for i, o := range d.s3Objects {
		size := humanBytes(o.Size)
		if o.StorageClass == "PREFIX" {
			size = mutedStyle.Render("DIR")
		}
		key := truncate(o.Key, 60)
		line := fmt.Sprintf("  %-60s %10s %-20s %s", key, size, o.LastModified, mutedStyle.Render(o.StorageClass))
		if i == d.cursor {
			line = selectedRow.Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(helpStyle.Render("\nenter open folder • esc back"))
	return b.String()
}

func (d *detailModel) cfnResourcesView() string {
	if len(d.cfnResources) == 0 {
		return mutedStyle.Render("  No resources in this stack")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-40s %s", "LOGICAL ID", "TYPE", "STATUS")) + "\n")
	for i, r := range d.cfnResources {
		line := fmt.Sprintf("  %-40s %-40s %s",
			truncate(r.LogicalID, 40),
			truncate(r.Type, 40),
			stateColor(r.Status),
		)
		if i == d.cursor {
			line = "▶" + line[1:]
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) cfnEventsView() string {
	if len(d.cfnEvents) == 0 {
		return mutedStyle.Render("  No stack events found")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-14s %-30s %-30s %-25s %s",
		"TIME", "RESOURCE", "TYPE", "STATUS", "REASON")) + "\n")
	limit := 30
	if len(d.cfnEvents) < limit {
		limit = len(d.cfnEvents)
	}
	for i, e := range d.cfnEvents[:limit] {
		line := fmt.Sprintf("  %-14s %-30s %-30s %-25s %s",
			e.Time,
			truncate(e.Resource, 30),
			truncate(e.Type, 30),
			stateColor(e.Status),
			mutedStyle.Render(truncate(e.Reason, 40)),
		)
		if i == d.cursor {
			line = "▶" + line[1:]
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) cfnOutputsView() string {
	if len(d.cfnOutputs) == 0 {
		return mutedStyle.Render("  This stack has no outputs defined")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-50s %s", "KEY", "VALUE", "DESCRIPTION")) + "\n")
	for _, o := range d.cfnOutputs {
		b.WriteString(fmt.Sprintf("  %-30s %-50s %s\n",
			truncate(o.Key, 30),
			truncate(o.Value, 50),
			mutedStyle.Render(truncate(o.Description, 40)),
		))
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) snsSubsView() string {
	if len(d.snsSubs) == 0 {
		return mutedStyle.Render("  No subscriptions on this topic yet")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-12s %-60s", "PROTOCOL", "ENDPOINT")) + "\n")
	for _, s := range d.snsSubs {
		b.WriteString(fmt.Sprintf("  %-12s %s\n", s.Protocol, truncate(s.Endpoint, 80)))
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) dynamoItemsView() string {
	if len(d.dynamoItems) == 0 {
		return mutedStyle.Render("  table is empty or scan returned no items")
	}
	// collect all keys for header
	keySet := map[string]struct{}{}
	for _, item := range d.dynamoItems {
		for k := range item {
			keySet[k] = struct{}{}
		}
	}
	var keys []string
	for k := range keySet {
		keys = append(keys, k)
	}

	var b strings.Builder
	colW := 25
	header := "  "
	for _, k := range keys {
		header += fmt.Sprintf("%-*s", colW, truncate(k, colW))
	}
	b.WriteString(headerStyle.Render(header) + "\n")
	for i, item := range d.dynamoItems {
		line := ""
		for _, k := range keys {
			line += fmt.Sprintf("%-*s", colW, truncate(item[k], colW))
		}
		if i == d.cursor {
			b.WriteString("▶ " + selectedRow.Render(line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("\n  showing first %d items • esc back", len(d.dynamoItems))))
	return b.String()
}

func (d *detailModel) apiResourcesView() string {
	if len(d.apiResources) == 0 {
		return mutedStyle.Render("  No resources in this stack")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-60s %s", "PATH", "METHODS")) + "\n")
	for i, r := range d.apiResources {
		methods := strings.Join(r.Methods, " ")
		line := fmt.Sprintf("%-60s %s", truncate(r.Path, 60), okStyle.Render(methods))
		if i == d.cursor {
			b.WriteString("▶ " + selectedRow.Render(fmt.Sprintf("%-60s", truncate(r.Path, 60))) + " " + okStyle.Render(methods) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) ecrImagesView() string {
	if len(d.ecrImages) == 0 {
		return mutedStyle.Render("  No images pushed to this repository yet")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-30s %-20s %-10s %s",
		"TAG", "PUSHED", "SIZE", "DIGEST")) + "\n")
	for i, img := range d.ecrImages {
		b.WriteString(rowLine(d.cursor, i, fmt.Sprintf("%-30s %-20s %-10s %s",
			truncate(img.Tag, 30), img.PushedAt,
			humanBytes(img.SizeBytes), mutedStyle.Render(truncate(img.Digest, 20)))) + "\n")
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) sfnExecutionsView() string {
	if len(d.sfnExecs) == 0 {
		return mutedStyle.Render("  No executions found for this state machine")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-40s %-12s %-20s %s",
		"NAME", "STATUS", "STARTED", "STOPPED")) + "\n")
	for i, e := range d.sfnExecs {
		b.WriteString(rowLine(d.cursor, i, fmt.Sprintf("%-40s %-12s %-20s %s",
			truncate(e.Name, 40), stateColor(e.Status),
			e.Started, e.Stopped)) + "\n")
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) dnsRecordsView() string {
	if len(d.dnsRecords) == 0 {
		return mutedStyle.Render("  No DNS records found in this zone")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-50s %-8s %-8s %s",
		"NAME", "TYPE", "TTL", "VALUE")) + "\n")
	for i, r := range d.dnsRecords {
		val := strings.Join(r.Values, ", ")
		b.WriteString(rowLine(d.cursor, i, fmt.Sprintf("%-50s %-8s %-8d %s",
			truncate(r.Name, 50), r.Type, r.TTL,
			truncate(val, 60))) + "\n")
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) alarmHistoryView() string {
	if len(d.alarmHistory) == 0 {
		return mutedStyle.Render("  No state change history found for this alarm")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render("  ALARM STATE HISTORY") + "\n")
	for _, line := range d.alarmHistory {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString(helpStyle.Render("\nesc back"))
	return b.String()
}

func (d *detailModel) ssmValueView() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  SSM PARAMETER VALUE") + "\n\n")
	b.WriteString(fmt.Sprintf("  Name:  %s\n\n", lipgloss.NewStyle().Foreground(cyan).Render(d.ssmParamName)))
	val := d.ssmValue
	if d.masked {
		val = "••••••••  (press s to reveal)"
	}
	b.WriteString(fmt.Sprintf("  Value: %s\n", val))
	b.WriteString(helpStyle.Render("\ns toggle mask • esc back"))
	return b.String()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func renderTabs(tabs []string, active int) string {
	var parts []string
	for i, t := range tabs {
		if i == active {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(orange).Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(orange).
				Padding(0, 1).Render(t))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(muted).Padding(0, 1).Render(t))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...) + "\n"
}

func humanBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
