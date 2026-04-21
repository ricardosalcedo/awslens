package tui

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) handleExport() (model, tea.Cmd) {
	data := m.exportData()
	if data == nil {
		return m, nil
	}
	svc := m.currentServiceName()
	ts := time.Now().Format("20060102-150405")
	jsonFile := fmt.Sprintf("awslens-%s-%s.json", svc, ts)
	csvFile := fmt.Sprintf("awslens-%s-%s.csv", svc, ts)

	m.modal = modal{kind: modalConfirm, title: "Export Data",
		body: fmt.Sprintf("Export %d items to:\n  %s\n  %s", len(data), jsonFile, csvFile)}
	m.modalOK = func() tea.Cmd {
		return func() tea.Msg {
			if err := writeJSON(jsonFile, data); err != nil {
				return writeErrMsg{err}
			}
			if err := writeCSV(csvFile, data); err != nil {
				return writeErrMsg{err}
			}
			return writeOKMsg{info: fmt.Sprintf("Exported to %s and %s", jsonFile, csvFile)}
		}
	}
	return m, nil
}

func (m model) currentServiceName() string {
	for _, item := range menu {
		if item.v == m.current {
			return item.label
		}
	}
	return "unknown"
}

func (m model) exportData() []map[string]string {
	return m.genericExport()
}

func (m model) genericExport() []map[string]string {
	var raw interface{}
	switch m.current {
	case viewEC2:
		raw = m.instances
	case viewLambda:
		raw = m.functions
	case viewS3:
		raw = m.buckets
	case viewRDS:
		raw = m.dbs
	case viewDynamo:
		raw = m.tables
	case viewAPIGW:
		raw = m.apis
	case viewECS:
		raw = m.clusters
	case viewECR:
		raw = m.repos
	case viewSFN:
		raw = m.machines
	case viewALB:
		raw = m.lbs
	case viewRoute53:
		raw = m.zones
	case viewSecrets:
		raw = m.secrets
	case viewSSM:
		raw = m.params
	case viewSQS:
		raw = m.queues
	case viewSNS:
		raw = m.topics
	case viewCW:
		raw = m.alarms
	case viewCFN:
		raw = m.stacks
	case viewCosts:
		raw = m.costs
	case viewElastiCache:
		raw = m.cacheClusters
	case viewOpenSearch:
		raw = m.osDomains
	case viewMSK:
		raw = m.mskClusters
	case viewGlue:
		raw = m.glueDbs
	case viewAthena:
		raw = m.athenaWGs
	case viewCodeCommit:
		raw = m.codeRepos
	case viewCodePipeline:
		raw = m.pipelines
	case viewCodeBuild:
		raw = m.buildProjects
	case viewEventBridge:
		raw = m.ebRules
	case viewWAF:
		raw = m.wafACLs
	default:
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var items []map[string]string
	if err := json.Unmarshal(b, &items); err != nil {
		// try array of objects with any values
		var generic []map[string]interface{}
		if err := json.Unmarshal(b, &generic); err != nil {
			return nil
		}
		for _, g := range generic {
			m := map[string]string{}
			for k, v := range g {
				m[k] = fmt.Sprintf("%v", v)
			}
			items = append(items, m)
		}
	}
	return items
}

func writeJSON(path string, data []map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func writeCSV(path string, data []map[string]string) error {
	if len(data) == 0 {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	// collect headers deterministically
	headerSet := map[string]bool{}
	for _, row := range data {
		for k := range row {
			headerSet[k] = true
		}
	}
	headers := make([]string, 0, len(headerSet))
	for k := range headerSet {
		headers = append(headers, k)
	}
	sort.Strings(headers)
	w.Write(headers) //nolint:errcheck // flushed below
	for _, row := range data {
		var vals []string
		for _, h := range headers {
			vals = append(vals, row[h])
		}
		w.Write(vals) //nolint:errcheck // flushed below
	}
	w.Flush()
	return w.Error()
}

// unused helper removed
