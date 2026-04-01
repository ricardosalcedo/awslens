package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	awsclient "github.com/awslens/awslens/internal/aws"
)

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

	// period label
	ref := time.Now().AddDate(0, -m.costPeriod, 0)
	periodLabel := ref.Format("January 2006")
	prevLabel := ref.AddDate(0, -1, 0).Format("Jan 2006")

	var maxAmt float64
	for _, c := range sorted {
		var f float64
		fmt.Sscanf(c.Amount, "%f", &f)
		if f > maxAmt {
			maxAmt = f
		}
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Foreground(orange).Bold(true).
		Render(fmt.Sprintf("  📅 %s", periodLabel)) +
		helpStyle.Render("  [ older • ] newer") + "\n\n")
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-45s %10s %10s  %s  %s", "SERVICE", periodLabel[:3], prevLabel, "", "TREND")) + "\n")
	for i, c := range sorted {
		var amt, prev float64
		fmt.Sscanf(c.Amount, "%f", &amt)
		fmt.Sscanf(c.PrevMonth, "%f", &prev)
		bar := awsclient.CostBar(amt, maxAmt, 15)
		barColored := lipgloss.NewStyle().Foreground(orange).Render(bar)
		trend := ""
		if prev > 0 && amt > prev*1.5 {
			trend = errorStyle.Render("▲ spike")
		} else if prev > 0 && amt > prev*1.1 {
			trend = warnStyle.Render("▲ up")
		} else if prev > 0 && amt < prev*0.5 {
			trend = okStyle.Render("▼ down")
		}
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-45s %10s %10s  %s  %s",
			truncate(c.Service, 45), fmt.Sprintf("$%.2f", amt), fmt.Sprintf("$%.2f", prev), barColored, trend)) + "\n")
	}
	b.WriteString("\n" + lipgloss.NewStyle().Foreground(orange).Bold(true).
		Render(fmt.Sprintf("  Total for %s: $%s", periodLabel, m.costTotal)) + "\n")
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


func (m model) securityView() string {
	if len(m.secFindings) == 0 {
		return okStyle.Render("  ✓ No security findings — looking good!")
	}
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-8s %-10s %-30s %s", "SEV", "SERVICE", "RESOURCE", "ISSUE")) + "\n")
	for i, f := range m.secFindings {
		sev := f.Severity
		switch sev {
		case "HIGH":
			sev = errorStyle.Render("HIGH")
		case "MEDIUM":
			sev = warnStyle.Render("MEDIUM")
		default:
			sev = mutedStyle.Render("LOW")
		}
		b.WriteString(rowLine(m.cursor, i, fmt.Sprintf("%-8s %-10s %-30s %s",
			sev, f.Service, truncate(f.Resource, 30), f.Issue)) + "\n")
	}
	high, med, low := 0, 0, 0
	for _, f := range m.secFindings {
		switch f.Severity {
		case "HIGH":
			high++
		case "MEDIUM":
			med++
		default:
			low++
		}
	}
	b.WriteString(fmt.Sprintf("\n  %s %d high  %s %d medium  %s %d low\n",
		errorStyle.Render("●"), high, warnStyle.Render("●"), med, mutedStyle.Render("●"), low))
	return b.String()
}
