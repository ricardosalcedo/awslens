package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	awsclient "github.com/awslens/awslens/internal/aws"
)

func (m model) crudHint() string {
	switch m.current {
	case viewS3:
		return " • n new bucket • d delete bucket"
	case viewLambda:
		return " • I invoke • d delete function"
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
				kv := awsclient.SplitKV(input)
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
				kv := awsclient.SplitKV(input)
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

func prodWarning(name string) string {
	lower := strings.ToLower(name)
	for _, kw := range []string{"prod", "production", "live", "main"} {
		if strings.Contains(lower, kw) {
			return "\n\n⚠  WARNING: This looks like a PRODUCTION resource!"
		}
	}
	return ""
}

func (m model) handleDelete() (model, tea.Cmd) {
	switch m.current {
	case viewS3:
		if m.cursor >= len(m.buckets) {
			return m, nil
		}
		name := m.buckets[m.cursor].Name
		m.modal = modal{kind: modalConfirm, title: "Delete S3 Bucket", body: fmt.Sprintf("Delete bucket %q?\n(must be empty)%s", name, prodWarning(name))}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteBucket(context.Background(), name) })
		}
	case viewLambda:
		if m.cursor >= len(m.functions) {
			return m, nil
		}
		fn := m.functions[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete Lambda Function", body: fmt.Sprintf("Delete function %q?%s", fn.Name, prodWarning(fn.Name))}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(fn.Region)
			return writeCmd(func() error { return client.DeleteFunction(context.Background(), fn.Name) })
		}
	case viewDynamo:
		if m.cursor >= len(m.tables) {
			return m, nil
		}
		tbl := m.tables[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete DynamoDB Table", body: fmt.Sprintf("Delete table %q?\n⚠ This deletes ALL data!%s", tbl.Name, prodWarning(tbl.Name))}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(tbl.Region)
			return writeCmd(func() error { return client.DeleteTable(context.Background(), tbl.Name) })
		}
	case viewSQS:
		if m.cursor >= len(m.queues) {
			return m, nil
		}
		q := m.queues[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete SQS Queue", body: fmt.Sprintf("Delete queue %q?%s", q.Name, prodWarning(q.Name))}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteQueue(context.Background(), q.URL) })
		}
	case viewSNS:
		if m.cursor >= len(m.topics) {
			return m, nil
		}
		t := m.topics[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete SNS Topic", body: fmt.Sprintf("Delete topic %q?%s", t.Name, prodWarning(t.Name))}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteTopic(context.Background(), t.ARN) })
		}
	case viewSSM:
		if m.cursor >= len(m.params) {
			return m, nil
		}
		p := m.params[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete SSM Parameter", body: fmt.Sprintf("Delete parameter %q?%s", p.Name, prodWarning(p.Name))}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteSSMParam(context.Background(), p.Name) })
		}
	case viewSecrets:
		if m.cursor >= len(m.secrets) {
			return m, nil
		}
		s := m.secrets[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete Secret", body: fmt.Sprintf("Delete secret %q?\n(7-day recovery window)%s", s.Name, prodWarning(s.Name))}
		m.modalOK = func() tea.Cmd {
			return writeCmd(func() error { return m.client.DeleteSecret(context.Background(), s.ARN) })
		}
	case viewECR:
		if m.cursor >= len(m.repos) {
			return m, nil
		}
		r := m.repos[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete ECR Repository", body: fmt.Sprintf("Delete repo %q?\n⚠ All images will be deleted!%s", r.Name, prodWarning(r.Name))}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(r.Region)
			return writeCmd(func() error { return client.DeleteECRRepo(context.Background(), r.Name) })
		}
	case viewCodeCommit:
		if m.cursor >= len(m.codeRepos) {
			return m, nil
		}
		r := m.codeRepos[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete CodeCommit Repo", body: fmt.Sprintf("Delete repo %q?%s", r.Name, prodWarning(r.Name))}
		m.modalOK = func() tea.Cmd {
			client := m.client.NewForRegion(r.Region)
			return writeCmd(func() error { return client.DeleteCodeRepo(context.Background(), r.Name) })
		}
	case viewEventBridge:
		if m.cursor >= len(m.ebRules) {
			return m, nil
		}
		r := m.ebRules[m.cursor]
		m.modal = modal{kind: modalConfirm, title: "Delete EventBridge Rule", body: fmt.Sprintf("Delete rule %q?\n(targets will be removed first)%s", r.Name, prodWarning(r.Name))}
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

func (m model) handleInvoke() (model, tea.Cmd) {
	if m.current != viewLambda || m.cursor >= len(m.functions) {
		return m, nil
	}
	fn := m.functions[m.cursor]
	m.modal = modal{kind: modalInput, title: "Invoke " + fn.Name, body: "JSON payload (or empty for {}):"}
	client := m.client.NewForRegion(fn.Region)
	name := fn.Name
	m.modalOK = func() tea.Cmd {
		payload := m.modal.input
		if payload == "" {
			payload = "{}"
		}
		return func() tea.Msg {
			out, err := client.InvokeFunction(context.Background(), name, []byte(payload))
			if err != nil {
				return errMsg{err}
			}
			result := string(out)
			if len(result) > 500 {
				result = result[:500] + "\n... (truncated)"
			}
			return invokeResultMsg{result}
		}
	}
	return m, nil
}


func (m model) handleInsight() (model, tea.Cmd) {
	summary := m.summarizeSelected()
	if summary == "" {
		return m, nil
	}
	// return cached insight if available
	if cached, ok := m.insightCache[summary]; ok {
		m.modal = modal{kind: modalInsight, title: "✨ AI Insight (cached)", body: cached}
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
			return insightCacheMsg{summary, text}
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
	case viewAPIGW:
		if m.cursor >= len(m.apis) { return "" }
		a := m.apis[m.cursor]
		return fmt.Sprintf("APIGateway: id=%s name=%s description=%s created=%s", a.ID, a.Name, a.Description, a.CreatedDate)
	case viewSFN:
		if m.cursor >= len(m.machines) { return "" }
		s := m.machines[m.cursor]
		return fmt.Sprintf("StepFunctions: name=%s type=%s created=%s", s.Name, s.Type, s.Created)
	case viewElastiCache:
		if m.cursor >= len(m.cacheClusters) { return "" }
		c := m.cacheClusters[m.cursor]
		return fmt.Sprintf("ElastiCache: id=%s engine=%s status=%s nodeType=%s nodes=%d", c.ID, c.Engine, c.Status, c.NodeType, c.Nodes)
	case viewOpenSearch:
		if m.cursor >= len(m.osDomains) { return "" }
		d := m.osDomains[m.cursor]
		return fmt.Sprintf("OpenSearch: domain=%s region=%s", d.Name, d.Region)
	case viewMSK:
		if m.cursor >= len(m.mskClusters) { return "" }
		c := m.mskClusters[m.cursor]
		return fmt.Sprintf("MSK: name=%s state=%s version=%s brokers=%d", c.Name, c.State, c.Version, c.Brokers)
	case viewGlue:
		if m.cursor >= len(m.glueDbs) { return "" }
		g := m.glueDbs[m.cursor]
		return fmt.Sprintf("Glue: database=%s description=%s", g.Name, g.Description)
	case viewAthena:
		if m.cursor >= len(m.athenaWGs) { return "" }
		a := m.athenaWGs[m.cursor]
		return fmt.Sprintf("Athena: workgroup=%s state=%s description=%s", a.Name, a.State, a.Description)
	case viewCodeCommit:
		if m.cursor >= len(m.codeRepos) { return "" }
		r := m.codeRepos[m.cursor]
		return fmt.Sprintf("CodeCommit: repo=%s description=%s modified=%s", r.Name, r.Description, r.LastModified)
	case viewCodePipeline:
		if m.cursor >= len(m.pipelines) { return "" }
		p := m.pipelines[m.cursor]
		return fmt.Sprintf("CodePipeline: name=%s version=%d updated=%s", p.Name, p.Version, p.Updated)
	case viewCodeBuild:
		if m.cursor >= len(m.buildProjects) { return "" }
		b := m.buildProjects[m.cursor]
		return fmt.Sprintf("CodeBuild: project=%s description=%s lastBuild=%s", b.Name, b.Description, b.LastBuild)
	case viewEventBridge:
		if m.cursor >= len(m.ebRules) { return "" }
		r := m.ebRules[m.cursor]
		return fmt.Sprintf("EventBridge: rule=%s state=%s schedule=%s description=%s", r.Name, r.State, r.Schedule, r.Description)
	case viewWAF:
		if m.cursor >= len(m.wafACLs) { return "" }
		w := m.wafACLs[m.cursor]
		return fmt.Sprintf("WAF: name=%s scope=%s rules=%d", w.Name, w.Scope, w.Rules)
	case viewCosts:
		if m.cursor >= len(m.costs) { return "" }
		c := m.costs[m.cursor]
		return fmt.Sprintf("Cost: service=%s amount=$%s", c.Service, c.Amount)
	}
	return ""
}


