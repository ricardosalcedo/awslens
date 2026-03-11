package aws

import (
	"errors"
	"testing"
)

func TestSplitKV(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"key=value", []string{"key", "value"}},
		{"name=hello world", []string{"name", "hello world"}},
		{"noequals", []string{"noequals"}},
		{"a=b=c", []string{"a", "b=c"}},
		{"=empty", []string{"", "empty"}},
	}
	for _, tt := range tests {
		got := SplitKV(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("SplitKV(%q) = %v, want %v", tt.input, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("SplitKV(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
			}
		}
	}
}

func TestSplitFields(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a=1 b=2", []string{"a=1", "b=2"}},
		{"single", []string{"single"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitFields(tt.input)
		if len(got) != len(tt.want) {
			t.Errorf("splitFields(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestSplitLast(t *testing.T) {
	tests := []struct {
		s, sep, want string
	}{
		{"arn:aws:sqs:us-east-1:123:my-queue", ":", "my-queue"},
		{"no-sep", ":", "no-sep"},
		{"a/b/c", "/", "c"},
	}
	for _, tt := range tests {
		got := splitLast(tt.s, tt.sep)
		if got != tt.want {
			t.Errorf("splitLast(%q, %q) = %q, want %q", tt.s, tt.sep, got, tt.want)
		}
	}
}

func TestSortCostsByAmount(t *testing.T) {
	entries := []CostEntry{
		{Service: "S3", Amount: "1.50"},
		{Service: "EC2", Amount: "10.00"},
		{Service: "Lambda", Amount: "0.25"},
	}
	sorted := SortCostsByAmount(entries)
	if sorted[0].Service != "EC2" {
		t.Errorf("expected EC2 first, got %s", sorted[0].Service)
	}
	if sorted[2].Service != "Lambda" {
		t.Errorf("expected Lambda last, got %s", sorted[2].Service)
	}
	// original should be unchanged
	if entries[0].Service != "S3" {
		t.Error("SortCostsByAmount mutated original slice")
	}
}

func TestCostBar(t *testing.T) {
	bar := CostBar(5.0, 10.0, 10)
	runes := []rune(bar)
	if len(runes) != 10 {
		t.Errorf("CostBar rune length = %d, want 10", len(runes))
	}
	// zero max
	if CostBar(5.0, 0, 10) != "" {
		t.Error("CostBar with max=0 should return empty")
	}
	// small amount should get at least 1 filled
	bar = CostBar(0.01, 10.0, 10)
	runes = []rune(bar)
	if runes[0] != '█' {
		t.Error("CostBar should have at least 1 filled block for non-zero amount")
	}
}

func TestParseLambdaTime(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", ""},
		{"2024-01-15T10:30:00.000+0000", "2024-01-15 10:30"},
		{"garbage", "garbage"},
	}
	for _, tt := range tests {
		got := parseLambdaTime(tt.input)
		if got != tt.want {
			t.Errorf("parseLambdaTime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsNotFoundErr(t *testing.T) {
	if !isNotFoundErr(nil) {
		t.Error("nil should return true")
	}
	if !isNotFoundErr(errors.New("NotFoundException: thing not found")) {
		t.Error("NotFoundException should return true")
	}
	if isNotFoundErr(errors.New("AccessDenied")) {
		t.Error("AccessDenied should return false")
	}
}

func TestIsAccessDenied(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("AccessDenied"), true},
		{errors.New("UnauthorizedOperation"), true},
		{errors.New("StatusCode: 403"), true},
		{errors.New("ThrottlingException"), false},
		{errors.New("connection refused"), false},
		{errors.New("context deadline exceeded"), false},
	}
	for _, tt := range tests {
		got := isAccessDenied(tt.err)
		if got != tt.want {
			t.Errorf("isAccessDenied(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestContains(t *testing.T) {
	if !contains("hello world", "world") {
		t.Error("should find 'world' in 'hello world'")
	}
	if contains("hello", "world") {
		t.Error("should not find 'world' in 'hello'")
	}
	if !contains("same", "same") {
		t.Error("exact match should return true")
	}
}

func TestParseDynamoJSON(t *testing.T) {
	result, err := parseDynamoJSON("id=123 name=foo")
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}

	_, err = parseDynamoJSON("")
	if err == nil {
		t.Error("empty input should return error")
	}
}

func TestRenderSparkline(t *testing.T) {
	// all zeros
	got := renderSparkline([]float64{0, 0, 0, 0})
	for _, r := range got {
		if r != '▁' {
			t.Errorf("all zeros should be ▁, got %c", r)
		}
	}
	// ascending
	got = renderSparkline([]float64{0, 5, 10})
	if len([]rune(got)) != 3 {
		t.Errorf("ascending length = %d", len([]rune(got)))
	}
	// single peak
	got = renderSparkline([]float64{0, 0, 10, 0, 0})
	runes := []rune(got)
	if runes[2] != '█' {
		t.Errorf("peak should be █, got %c", runes[2])
	}
}

func TestExtractName(t *testing.T) {
	tests := []struct{ input, want string }{
		{"arn:aws:sqs:us-east-1:123:my-queue", "my-queue"},
		{"arn:aws:iam::123:role/MyRole", "MyRole"},
		{"simple-name", "simple-name"},
	}
	for _, tt := range tests {
		got := extractName(tt.input)
		if got != tt.want {
			t.Errorf("extractName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDeps(t *testing.T) {
	deps := []Dependency{
		{From: "myFunc", To: "myTable", Relation: "reads/writes DynamoDB"},
	}
	got := FormatDeps(deps)
	if !containsStr(got, "myFunc") || !containsStr(got, "myTable") {
		t.Errorf("FormatDeps missing content: %q", got)
	}
	// empty
	if FormatDeps(nil) == "" {
		t.Error("nil deps should return 'No dependencies' message")
	}
}
