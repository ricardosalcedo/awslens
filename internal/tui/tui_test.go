package tui

import (
	"fmt"
	"os"
	"testing"

	awsclient "github.com/awslens/awslens/internal/aws"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		n    int
		want string
	}{
		{"hello", 10, "hello"},
		{"hello world", 5, "hell…"},
		{"abc", 3, "abc"},
		{"ab", 1, "…"},
	}
	for _, tt := range tests {
		got := truncate(tt.s, tt.n)
		if got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
		}
	}
}

func TestMax0(t *testing.T) {
	if max0(-5) != 0 {
		t.Error("max0(-5) should be 0")
	}
	if max0(3) != 3 {
		t.Error("max0(3) should be 3")
	}
	if max0(0) != 0 {
		t.Error("max0(0) should be 0")
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, tt := range tests {
		got := humanBytes(tt.input)
		if got != tt.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestProdWarning(t *testing.T) {
	if prodWarning("my-dev-bucket") != "" {
		t.Error("dev should not trigger warning")
	}
	if prodWarning("my-prod-api") == "" {
		t.Error("prod should trigger warning")
	}
	if prodWarning("Production-DB") == "" {
		t.Error("Production (case insensitive) should trigger warning")
	}
	if prodWarning("live-service") == "" {
		t.Error("live should trigger warning")
	}
}

func TestFriendlyErr(t *testing.T) {
	tests := []struct {
		msg      string
		contains string
	}{
		{"StatusCode: 403 AccessDenied", "permission"},
		{"ExpiredTokenException", "expired"},
		{"no such host", "internet"},
		{"NoCredentialProviders", "No AWS credentials"},
		{"something random", "something random"},
	}
	for _, tt := range tests {
		got := friendlyErr(fmt.Errorf("%s", tt.msg))
		if !containsStr(got, tt.contains) {
			t.Errorf("friendlyErr(%q) = %q, should contain %q", tt.msg, got, tt.contains)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestScrollLines(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}

	got := scrollLines(lines, 0, 3)
	if len(got) != 3 || got[0] != "a" {
		t.Errorf("scrollLines(0,3) = %v", got)
	}

	got = scrollLines(lines, 2, 3)
	if len(got) != 3 || got[0] != "c" {
		t.Errorf("scrollLines(2,3) = %v", got)
	}

	got = scrollLines(lines, 4, 3)
	if len(got) != 1 || got[0] != "e" {
		t.Errorf("scrollLines(4,3) = %v", got)
	}

	got = scrollLines(lines, 10, 3)
	if got != nil {
		t.Errorf("scrollLines(10,3) should be nil, got %v", got)
	}
}

func TestRowLine(t *testing.T) {
	selected := rowLine(2, 2, "test")
	if !containsStr(selected, "▶") {
		t.Error("selected row should contain ▶")
	}
	normal := rowLine(0, 1, "test")
	if containsStr(normal, "▶") {
		t.Error("non-selected row should not contain ▶")
	}
}

func TestServiceStat(t *testing.T) {
	m := model{}

	// empty data returns empty string
	if m.serviceStat(viewEC2) != "" {
		t.Error("empty instances should return empty stat")
	}

	// with data
	m.instances = []awsclient.Instance{
		{State: "running"},
		{State: "stopped"},
		{State: "running"},
	}
	got := m.serviceStat(viewEC2)
	if got != "3 instances (2 running)" {
		t.Errorf("serviceStat(EC2) = %q", got)
	}

	m.costTotal = "142.37"
	got = m.serviceStat(viewCosts)
	if got != "$142.37 this month" {
		t.Errorf("serviceStat(Costs) = %q", got)
	}

	m.alarms = []awsclient.CWAlarm{
		{State: "OK"},
		{State: "ALARM"},
	}
	got = m.serviceStat(viewCW)
	if got != "2 alarms (1 firing)" {
		t.Errorf("serviceStat(CW) = %q", got)
	}
}

func TestProfileMeta(t *testing.T) {
	p := awsclient.Profile{Name: "default"}
	got := profileMeta(p)
	if got == "" {
		t.Error("default profile should have some meta")
	}

	p = awsclient.Profile{Name: "admin", RoleARN: "arn:aws:iam::123:role/Admin", SourceProfile: "default"}
	got = profileMeta(p)
	if !containsStr(got, "Admin") {
		t.Errorf("role profile meta should contain role name, got %q", got)
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", "••••"},
		{"ab", "••••"},
		{"abcd", "••••"},
		{"abcde", "•bcde"},
		{"AKIAIOSFODNN7EXAMPLE", "••••••••••••••••MPLE"},
	}
	for _, tt := range tests {
		got := maskKey(tt.input)
		if got != tt.want {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStateColor(t *testing.T) {
	// Just verify it doesn't panic and returns non-empty for known states
	states := []string{"running", "stopped", "terminated", "unknown", "active", "available"}
	for _, s := range states {
		got := stateColor(s)
		if got == "" {
			t.Errorf("stateColor(%q) returned empty", s)
		}
	}
}

func TestAlarmStateColor(t *testing.T) {
	states := []string{"OK", "ALARM", "INSUFFICIENT_DATA"}
	for _, s := range states {
		got := alarmStateColor(s)
		if got == "" {
			t.Errorf("alarmStateColor(%q) returned empty", s)
		}
	}
}

func TestCrudHint(t *testing.T) {
	// services with CRUD hints
	crudViews := []view{viewS3, viewLambda, viewDynamo, viewSQS, viewSNS, viewSSM, viewSecrets, viewECR, viewCodeCommit, viewEventBridge}
	for _, v := range crudViews {
		m := model{current: v}
		if m.crudHint() == "" {
			t.Errorf("crudHint() empty for view %d", v)
		}
	}
	// service without CRUD hint
	m := model{current: viewRDS}
	if m.crudHint() != "" {
		t.Error("RDS should not have a CRUD hint")
	}
}

func TestWriteJSONAndCSV(t *testing.T) {
	dir := t.TempDir()
	data := []map[string]string{
		{"name": "a", "value": "1"},
		{"name": "b", "value": "2"},
	}

	jsonPath := dir + "/test.json"
	if err := writeJSON(jsonPath, data); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
	jsonBytes, _ := os.ReadFile(jsonPath)
	if !containsStr(string(jsonBytes), `"name"`) {
		t.Error("JSON output missing expected content")
	}

	csvPath := dir + "/test.csv"
	if err := writeCSV(csvPath, data); err != nil {
		t.Fatalf("writeCSV: %v", err)
	}
	csvBytes, _ := os.ReadFile(csvPath)
	csvStr := string(csvBytes)
	if !containsStr(csvStr, "name") || !containsStr(csvStr, "a") {
		t.Error("CSV output missing expected content")
	}

	// empty data
	if err := writeCSV(dir+"/empty.csv", nil); err != nil {
		t.Fatalf("writeCSV(nil): %v", err)
	}
}

func TestCurrentServiceName(t *testing.T) {
	m := model{current: viewEC2}
	got := m.currentServiceName()
	if got == "unknown" || got == "" {
		t.Errorf("currentServiceName for EC2 = %q", got)
	}

	m.current = view(9999)
	if m.currentServiceName() != "unknown" {
		t.Error("invalid view should return 'unknown'")
	}
}
