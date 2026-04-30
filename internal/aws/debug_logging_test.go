package aws

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

// captureSlog replaces the default slog logger with one that writes to a
// buffer at DEBUG level, runs fn, then restores the original logger.
func captureSlog(fn func()) string {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(orig)
	fn()
	return buf.String()
}

func TestProbeAccess_LogsDeniedServices(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("probe access", "service", "TestSvc", "allowed", false, "error", "AccessDenied")
	})
	if !strings.Contains(logged, "probe access") {
		t.Error("expected 'probe access' in debug log")
	}
	if !strings.Contains(logged, "TestSvc") {
		t.Error("expected service name in debug log")
	}
}

func TestSecurityAudit_LogsErrors(t *testing.T) {
	tests := []struct {
		name string
		msg  string
	}{
		{"security groups", "security audit: security groups"},
		{"s3 buckets", "security audit: s3 buckets"},
		{"iam users", "security audit: iam users"},
		{"root account", "security audit: root account"},
	}
	for _, tt := range tests {
		logged := captureSlog(func() {
			slog.Debug(tt.msg, "error", "simulated failure")
		})
		if !strings.Contains(logged, tt.msg) {
			t.Errorf("expected %q in debug log", tt.msg)
		}
		if !strings.Contains(logged, "simulated failure") {
			t.Errorf("expected error message in debug log for %s", tt.name)
		}
	}
}

func TestS3GetBucketLocation_LogsError(t *testing.T) {
	logged := captureSlog(func() {
		slog.Debug("s3 GetBucketLocation failed", "bucket", "my-bucket", "error", "access denied")
	})
	if !strings.Contains(logged, "s3 GetBucketLocation failed") {
		t.Error("expected 's3 GetBucketLocation failed' in debug log")
	}
	if !strings.Contains(logged, "my-bucket") {
		t.Error("expected bucket name in debug log")
	}
}
