package tui

import (
	"encoding/csv"
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

func TestWriteCSVDeterministicHeaders(t *testing.T) {
	data := []map[string]string{
		{"Zebra": "1", "Apple": "2", "Mango": "3"},
		{"Zebra": "4", "Apple": "5", "Banana": "6"},
	}

	dir := t.TempDir()
	path := dir + "/test.csv"
	if err := writeCSV(path, data); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	// header row must be sorted
	want := []string{"Apple", "Banana", "Mango", "Zebra"}
	got := records[0]
	if len(got) != len(want) {
		t.Fatalf("header len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("header[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	// verify data rows use the sorted header order
	if records[1][0] != "2" || records[1][3] != "1" {
		t.Errorf("row 1 = %v, expected Apple=2 ... Zebra=1", records[1])
	}
}

func TestWriteCSVEmpty(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/empty.csv"
	if err := writeCSV(path, nil); err != nil {
		t.Fatal(err)
	}
	// file should not be created for empty data
	if _, err := os.Stat(path); err == nil {
		t.Error("writeCSV with nil data should not create a file")
	}
}
