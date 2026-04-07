package aws

import (
	"errors"
	"math"
	"strings"
	"testing"
)

// ── splitLast edge cases ─────────────────────────────────────────────────────

func TestSplitLastEdgeCases(t *testing.T) {
	tests := []struct {
		s, sep, want string
	}{
		{"", ":", ""},
		{":", ":", ""},
		{"a:", ":", ""},
		{":b", ":", "b"},
		{"a:b:c:d", ":", "d"},
		{"no-sep-here", "/", "no-sep-here"},
		{"trailing/", "/", ""},
		{"/leading", "/", "leading"},
	}
	for _, tt := range tests {
		got := splitLast(tt.s, tt.sep)
		if got != tt.want {
			t.Errorf("splitLast(%q, %q) = %q, want %q", tt.s, tt.sep, got, tt.want)
		}
	}
}

// ── ParseCostAmount edge cases ───────────────────────────────────────────────

func TestParseCostAmountEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"-5.25", -5.25},
		{"1e3", 1000},
		{"  ", 0},
		{"1.23456789", 1.23456789},
	}
	for _, tt := range tests {
		got := ParseCostAmount(tt.input)
		if got != tt.want {
			t.Errorf("ParseCostAmount(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// ── SortCostsByAmount edge cases ─────────────────────────────────────────────

func TestSortCostsByAmountEdgeCases(t *testing.T) {
	// empty slice
	sorted := SortCostsByAmount(nil)
	if len(sorted) != 0 {
		t.Errorf("SortCostsByAmount(nil) len = %d, want 0", len(sorted))
	}

	// single entry
	sorted = SortCostsByAmount([]CostEntry{{Service: "S3", Amount: "1.00"}})
	if len(sorted) != 1 || sorted[0].Service != "S3" {
		t.Error("single entry should be returned as-is")
	}

	// equal amounts preserve relative order (stable-ish check)
	entries := []CostEntry{
		{Service: "A", Amount: "5.00"},
		{Service: "B", Amount: "5.00"},
		{Service: "C", Amount: "5.00"},
	}
	sorted = SortCostsByAmount(entries)
	if len(sorted) != 3 {
		t.Errorf("expected 3 entries, got %d", len(sorted))
	}

	// unparseable amounts treated as 0
	entries = []CostEntry{
		{Service: "Good", Amount: "10.00"},
		{Service: "Bad", Amount: "garbage"},
	}
	sorted = SortCostsByAmount(entries)
	if sorted[0].Service != "Good" {
		t.Errorf("expected Good first, got %s", sorted[0].Service)
	}
}

// ── CostBar edge cases ──────────────────────────────────────────────────────

func TestCostBarEdgeCases(t *testing.T) {
	tests := []struct {
		amount, max float64
		width       int
		wantLen     int
		desc        string
	}{
		{10.0, 10.0, 10, 10, "full bar"},
		{0.0, 10.0, 10, 10, "zero amount"},
		{10.0, 10.0, 1, 1, "width 1"},
		{0.001, 100.0, 20, 20, "tiny amount gets 1 filled"},
	}
	for _, tt := range tests {
		got := CostBar(tt.amount, tt.max, tt.width)
		runes := []rune(got)
		if len(runes) != tt.wantLen {
			t.Errorf("CostBar(%s): len = %d, want %d", tt.desc, len(runes), tt.wantLen)
		}
	}

	// full bar should be all filled
	bar := CostBar(10.0, 10.0, 5)
	if bar != "█████" {
		t.Errorf("full bar = %q, want all filled", bar)
	}

	// zero amount should be all empty
	bar = CostBar(0.0, 10.0, 5)
	if bar != "░░░░░" {
		t.Errorf("zero bar = %q, want all empty", bar)
	}
}

// ── parseLambdaTime edge cases ──────────────────────────────────────────────

func TestParseLambdaTimeEdgeCases(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		// RFC3339 format
		{"2024-06-15T14:30:00Z", "2024-06-15 14:30"},
		// RFC3339 with offset
		{"2024-06-15T14:30:00+05:00", "2024-06-15 14:30"},
		// nanosecond format
		{"2024-06-15T14:30:00.123456789Z", "2024-06-15 14:30"},
		// unparseable returns as-is
		{"not-a-date", "not-a-date"},
		{"2024", "2024"},
	}
	for _, tt := range tests {
		got := parseLambdaTime(tt.input)
		if got != tt.want {
			t.Errorf("parseLambdaTime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── renderSparkline edge cases ──────────────────────────────────────────────

func TestRenderSparklineEdgeCases(t *testing.T) {
	// single value
	got := renderSparkline([]float64{5.0})
	if len([]rune(got)) != 1 {
		t.Errorf("single value: len = %d, want 1", len([]rune(got)))
	}

	// all same non-zero values → all max bars
	got = renderSparkline([]float64{5, 5, 5})
	for _, r := range got {
		if r != '█' {
			t.Errorf("all-same should be max bar, got %c", r)
		}
	}

	// descending
	got = renderSparkline([]float64{10, 5, 0})
	runes := []rune(got)
	if runes[0] != '█' {
		t.Errorf("descending: first should be max, got %c", runes[0])
	}
	if runes[2] != '▁' {
		t.Errorf("descending: last should be min, got %c", runes[2])
	}

	// very large values
	got = renderSparkline([]float64{0, 1e12})
	if len([]rune(got)) != 2 {
		t.Errorf("large values: len = %d, want 2", len([]rune(got)))
	}
}

// ── extractName edge cases ──────────────────────────────────────────────────

func TestExtractNameEdgeCases(t *testing.T) {
	tests := []struct{ input, want string }{
		{"", ""},
		{"arn:aws:lambda:us-east-1:123:function:my-func", "my-func"},
		{"arn:aws:iam::123:role/path/to/MyRole", "MyRole"},
		{"just-a-name", "just-a-name"},
		{"a:b:c", "c"},
		{"trailing/", ""},
		{"no-colon-but/slash", "slash"},
	}
	for _, tt := range tests {
		got := extractName(tt.input)
		if got != tt.want {
			t.Errorf("extractName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ── contains / containsStr edge cases ───────────────────────────────────────

func TestContainsEdgeCases(t *testing.T) {
	tests := []struct {
		s, sub string
		want   bool
	}{
		{"", "", true},
		{"a", "", true},
		{"", "a", false},
		{"abc", "abc", true},
		{"abc", "abcd", false},
		{"abcabc", "cab", true},
	}
	for _, tt := range tests {
		got := contains(tt.s, tt.sub)
		if got != tt.want {
			t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.sub, got, tt.want)
		}
	}
}

// ── isNotFoundErr edge cases ────────────────────────────────────────────────

func TestIsNotFoundErrEdgeCases(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{errors.New("NoSuchEntity: user not found"), true},
		{errors.New("ResourceNotFoundException: table gone"), true},
		{errors.New("some random error"), false},
		{errors.New(""), false},
	}
	for _, tt := range tests {
		got := isNotFoundErr(tt.err)
		if got != tt.want {
			t.Errorf("isNotFoundErr(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

// ── isAccessDenied edge cases ───────────────────────────────────────────────

func TestIsAccessDeniedEdgeCases(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{errors.New("AuthorizationError: not allowed"), true},
		{errors.New("AccessDeniedException: forbidden"), true},
		{errors.New("RequestLimitExceeded"), false},
		{errors.New(""), false},
	}
	for _, tt := range tests {
		got := isAccessDenied(tt.err)
		if got != tt.want {
			t.Errorf("isAccessDenied(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

// ── parseDynamoJSON edge cases ──────────────────────────────────────────────

func TestParseDynamoJSONEdgeCases(t *testing.T) {
	// multiple pairs
	result, err := parseDynamoJSON("a=1 b=2 c=3")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}

	// value with equals sign
	result, err = parseDynamoJSON("key=a=b")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}

	// single pair with no value part (no '=') should produce no items → error
	_, err = parseDynamoJSON("noequals")
	if err == nil {
		t.Error("input without '=' should return error")
	}

	// whitespace only
	_, err = parseDynamoJSON("   ")
	if err == nil {
		t.Error("whitespace-only input should return error")
	}
}

// ── splitFields edge cases ──────────────────────────────────────────────────

func TestSplitFieldsEdgeCases(t *testing.T) {
	// multiple spaces between fields
	got := splitFields("a  b")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("splitFields('a  b') = %v", got)
	}

	// leading/trailing spaces
	got = splitFields(" a b ")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("splitFields(' a b ') = %v", got)
	}

	// single field
	got = splitFields("only")
	if len(got) != 1 || got[0] != "only" {
		t.Errorf("splitFields('only') = %v", got)
	}
}

// ── SplitKV edge cases ──────────────────────────────────────────────────────

func TestSplitKVEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", []string{""}},
		{"key=", []string{"key", ""}},
		{"=", []string{"", ""}},
		{"key=val=ue=extra", []string{"key", "val=ue=extra"}},
	}
	for _, tt := range tests {
		got := SplitKV(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("SplitKV(%q) len = %d, want %d", tt.input, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("SplitKV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

// ── FormatDeps edge cases ───────────────────────────────────────────────────

func TestFormatDepsEdgeCases(t *testing.T) {
	// empty slice (not nil)
	got := FormatDeps([]Dependency{})
	if !strings.Contains(got, "No dependencies") {
		t.Errorf("empty slice should say no deps, got %q", got)
	}

	// multiple deps
	deps := []Dependency{
		{From: "funcA", To: "tableA", Relation: "reads"},
		{From: "funcA", To: "queueB", Relation: "writes"},
	}
	got = FormatDeps(deps)
	if !strings.Contains(got, "funcA") || !strings.Contains(got, "tableA") || !strings.Contains(got, "queueB") {
		t.Errorf("FormatDeps missing content: %q", got)
	}
	// should have 2 lines (each dep on its own line)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
}

// ── fmtMetric edge cases ────────────────────────────────────────────────────

func TestFmtMetricEdgeCases(t *testing.T) {
	// zero values
	m := &MetricSummary{Name: "Invocations", Avg: 0, Max: 0, Sum: 0}
	got := fmtMetric(m, "")
	if !strings.Contains(got, "Invocations") {
		t.Errorf("fmtMetric zero values missing name: %q", got)
	}

	// large values
	m = &MetricSummary{Name: "Bytes", Avg: 1e9, Max: 5e9, Sum: 1e12}
	got = fmtMetric(m, "B")
	if !strings.Contains(got, "B") {
		t.Errorf("fmtMetric large values missing unit: %q", got)
	}
}

// ── EnrichContext.String edge cases ─────────────────────────────────────────

func TestEnrichContextStringEdgeCases(t *testing.T) {
	// only errors section
	e := &EnrichContext{Errors: []string{"timeout", "connection refused"}}
	got := e.String()
	if !strings.Contains(got, "RECENT ERRORS") {
		t.Error("should contain ERRORS header")
	}
	if strings.Contains(got, "METRICS") || strings.Contains(got, "DEPENDENCIES") {
		t.Error("should not contain other sections")
	}

	// only deps section
	e = &EnrichContext{Deps: []string{"funcA → tableB"}}
	got = e.String()
	if !strings.Contains(got, "DEPENDENCIES") {
		t.Error("should contain DEPENDENCIES header")
	}
	if strings.Contains(got, "METRICS") || strings.Contains(got, "ERRORS") {
		t.Error("should not contain other sections")
	}

	// only extra section
	e = &EnrichContext{Extra: []string{"runtime: go1.22"}}
	got = e.String()
	if !strings.Contains(got, "ADDITIONAL CONTEXT") {
		t.Error("should contain ADDITIONAL CONTEXT header")
	}
}

// ── splitOn edge cases ──────────────────────────────────────────────────────

func TestSplitOnEdgeCases(t *testing.T) {
	// consecutive separators
	got := splitOn("a,,b", ',')
	if len(got) != 3 || got[1] != "" {
		t.Errorf("splitOn('a,,b', ',') = %v, want [a  b]", got)
	}

	// separator at start and end
	got = splitOn(",a,", ',')
	if len(got) != 3 || got[0] != "" || got[2] != "" {
		t.Errorf("splitOn(',a,', ',') = %v", got)
	}

	// only separators
	got = splitOn(",,", ',')
	if len(got) != 3 {
		t.Errorf("splitOn(',,', ',') len = %d, want 3", len(got))
	}
}

// ── dynamoAttrToString edge cases ───────────────────────────────────────────

func TestDynamoAttrToStringEdgeCases(t *testing.T) {
	// nil
	got := dynamoAttrToString(nil)
	if got != "<nil>" {
		t.Errorf("dynamoAttrToString(nil) = %q, want '<nil>'", got)
	}

	// bool
	got = dynamoAttrToString(true)
	if got != "true" {
		t.Errorf("dynamoAttrToString(true) = %q", got)
	}

	// float
	got = dynamoAttrToString(3.14)
	if !strings.Contains(got, "3.14") {
		t.Errorf("dynamoAttrToString(3.14) = %q", got)
	}
}

// ── CommonRegions sanity ────────────────────────────────────────────────────

func TestCommonRegionsNotEmpty(t *testing.T) {
	if len(CommonRegions) == 0 {
		t.Error("CommonRegions should not be empty")
	}
	for _, r := range CommonRegions {
		if r == "" {
			t.Error("CommonRegions contains empty string")
		}
		if !strings.Contains(r, "-") {
			t.Errorf("region %q doesn't look like a valid AWS region", r)
		}
	}
}

// ── CostBar math precision ──────────────────────────────────────────────────

func TestCostBarPrecision(t *testing.T) {
	// amount > max should cap at full bar (not panic)
	bar := CostBar(20.0, 10.0, 5)
	if bar != "█████" {
		t.Errorf("CostBar(20, 10, 5) = %q, want full bar", bar)
	}

	// NaN max
	bar = CostBar(5.0, math.NaN(), 10)
	// just ensure no panic — output is undefined for NaN
	_ = bar
}
