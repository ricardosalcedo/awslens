package aws

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

// stubItem is a minimal type for testing aggregateRegions.
type stubItem struct{ Region string }

func TestAggregateRegions_MergesResults(t *testing.T) {
	c := &Client{Region: "us-east-1"}
	items := aggregateRegions(context.Background(), c, func(_ context.Context, rc *Client) ([]stubItem, error) {
		return []stubItem{{Region: rc.Region}}, nil
	})
	if len(items) != len(CommonRegions) {
		t.Errorf("expected %d items, got %d", len(CommonRegions), len(items))
	}
}

func TestAggregateRegions_EmptyRegionsOK(t *testing.T) {
	c := &Client{Region: "us-east-1"}
	items := aggregateRegions(context.Background(), c, func(_ context.Context, _ *Client) ([]stubItem, error) {
		return nil, nil
	})
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestAggregateRegions_ErrorsLoggedAndSkipped(t *testing.T) {
	// Capture slog output.
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(orig)

	c := &Client{Region: "us-east-1"}
	items := aggregateRegions(context.Background(), c, func(_ context.Context, _ *Client) ([]stubItem, error) {
		return nil, errors.New("access denied")
	})
	if len(items) != 0 {
		t.Errorf("expected 0 items from all-error regions, got %d", len(items))
	}
	logged := buf.String()
	if !strings.Contains(logged, "region fetch failed") {
		t.Error("expected 'region fetch failed' in debug log output")
	}
	if !strings.Contains(logged, "access denied") {
		t.Error("expected error message in debug log output")
	}
}

func TestAggregateRegions_PartialFailure(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(orig)

	c := &Client{Region: "us-east-1"}
	items := aggregateRegions(context.Background(), c, func(_ context.Context, rc *Client) ([]stubItem, error) {
		if rc.Region == "us-east-1" {
			return nil, errors.New("timeout")
		}
		return []stubItem{{Region: rc.Region}}, nil
	})
	// us-east-1 should fail, rest should succeed
	expectedSuccess := len(CommonRegions) - 1
	if len(items) != expectedSuccess {
		t.Errorf("expected %d items, got %d", expectedSuccess, len(items))
	}
	logged := buf.String()
	if !strings.Contains(logged, "us-east-1") {
		t.Error("expected failed region 'us-east-1' in debug log")
	}
	if !strings.Contains(logged, "timeout") {
		t.Error("expected 'timeout' error in debug log")
	}
}
