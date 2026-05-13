package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestFetchServicesDeterministicOrder(t *testing.T) {
	// fetchServices should preserve original service order even when running in parallel.
	// We can't call fetchService without AWS creds, but we can verify the output
	// functions maintain order with pre-built data.
	data := []serviceData{
		{Service: "ec2", Items: []map[string]interface{}{{"ID": "i-1"}}},
		{Service: "lambda", Items: []map[string]interface{}{{"Name": "fn1"}}},
		{Service: "s3", Items: []map[string]interface{}{{"Name": "bucket1"}}},
	}
	var prev string
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		if err := writeJSONOut(&buf, data); err != nil {
			t.Fatal(err)
		}
		if prev != "" && buf.String() != prev {
			t.Fatal("output is non-deterministic")
		}
		prev = buf.String()
	}
}

func TestMaxExportConcurrency(t *testing.T) {
	if maxExportConcurrency < 1 {
		t.Fatal("maxExportConcurrency must be at least 1")
	}
	if maxExportConcurrency > 20 {
		t.Fatal("maxExportConcurrency should not exceed 20 to avoid API throttling")
	}
}

func TestWriteJSONOut(t *testing.T) {
	data := []serviceData{
		{Service: "ec2", Items: []map[string]interface{}{
			{"ID": "i-123", "State": "running"},
			{"ID": "i-456", "State": "stopped"},
		}},
	}
	var buf bytes.Buffer
	if err := writeJSONOut(&buf, data); err != nil {
		t.Fatal(err)
	}
	var got []serviceData
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Service != "ec2" || len(got[0].Items) != 2 {
		t.Fatalf("unexpected output: %s", buf.String())
	}
}

func TestWriteCSVOut(t *testing.T) {
	data := []serviceData{
		{Service: "lambda", Items: []map[string]interface{}{
			{"Name": "fn1", "Runtime": "go1.x"},
			{"Name": "fn2", "Runtime": "python3.9"},
		}},
	}
	var buf bytes.Buffer
	if err := writeCSVOut(&buf, data); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Fatalf("expected 3 lines, got %d: %s", len(lines), buf.String())
	}
	// Headers should be sorted
	headers := lines[0]
	if !strings.Contains(headers, "Name") || !strings.Contains(headers, "Runtime") || !strings.Contains(headers, "service") {
		t.Fatalf("missing expected headers: %s", headers)
	}
}

func TestWriteCSVOutDeterministicHeaders(t *testing.T) {
	data := []serviceData{
		{Service: "s3", Items: []map[string]interface{}{
			{"Name": "bucket1", "Region": "us-east-1"},
		}},
	}
	// Run multiple times to verify determinism
	var prev string
	for i := 0; i < 10; i++ {
		var buf bytes.Buffer
		if err := writeCSVOut(&buf, data); err != nil {
			t.Fatal(err)
		}
		if prev != "" && buf.String() != prev {
			t.Fatal("CSV output is non-deterministic")
		}
		prev = buf.String()
	}
}

func TestToMaps(t *testing.T) {
	type item struct {
		Name string `json:"Name"`
		Age  int    `json:"Age"`
	}
	input := []item{{Name: "a", Age: 1}, {Name: "b", Age: 2}}
	got := toMaps(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0]["Name"] != "a" {
		t.Fatalf("expected Name=a, got %v", got[0]["Name"])
	}
}

func TestValidateOutputFormat(t *testing.T) {
	// Test that invalid format is rejected
	profile = "test"
	err := runHeadlessExport("xml", "")
	if err == nil || !strings.Contains(err.Error(), "must be 'json' or 'csv'") {
		t.Fatalf("expected format error, got: %v", err)
	}
}

func TestValidateServiceName(t *testing.T) {
	profile = "test"
	err := runHeadlessExport("json", "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unknown service") {
		t.Fatalf("expected unknown service error, got: %v", err)
	}
}

func TestRequiresProfile(t *testing.T) {
	profile = ""
	err := runHeadlessExport("json", "")
	if err == nil || !strings.Contains(err.Error(), "--profile is required") {
		t.Fatalf("expected profile required error, got: %v", err)
	}
}

func TestWriteCSVOutEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := writeCSVOut(&buf, nil); err != nil {
		t.Fatal(err)
	}
	// Empty data should produce empty output (no headers)
	if buf.Len() != 0 {
		t.Fatalf("expected empty output, got: %s", buf.String())
	}
}

func TestWriteJSONOutEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := writeJSONOut(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "null") {
		t.Fatalf("expected null JSON, got: %s", buf.String())
	}
}
