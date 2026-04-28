package aws

import (
	"context"
	"errors"
	"testing"
)

func TestAggregateRegions_AllSucceed(t *testing.T) {
	c := &Client{Region: "us-east-1"}
	items, failed := aggregateRegions(ctx(), c, func(_ context.Context, rc *Client) ([]string, error) {
		return []string{rc.Region + "-item"}, nil
	})
	if len(failed) != 0 {
		t.Errorf("expected no failures, got %v", failed)
	}
	if len(items) != len(CommonRegions) {
		t.Errorf("expected %d items, got %d", len(CommonRegions), len(items))
	}
}

func TestAggregateRegions_SomeFail(t *testing.T) {
	c := &Client{Region: "us-east-1"}
	items, failed := aggregateRegions(ctx(), c, func(_ context.Context, rc *Client) ([]string, error) {
		if rc.Region == "us-east-1" || rc.Region == "eu-west-1" {
			return nil, errors.New("access denied")
		}
		return []string{rc.Region}, nil
	})
	if len(failed) != 2 {
		t.Fatalf("expected 2 failures, got %d: %v", len(failed), failed)
	}
	// failed should be sorted
	if failed[0] != "eu-west-1" || failed[1] != "us-east-1" {
		t.Errorf("expected sorted [eu-west-1 us-east-1], got %v", failed)
	}
	if len(items) != len(CommonRegions)-2 {
		t.Errorf("expected %d items, got %d", len(CommonRegions)-2, len(items))
	}
}

func TestAggregateRegions_AllFail(t *testing.T) {
	c := &Client{Region: "us-east-1"}
	items, failed := aggregateRegions(ctx(), c, func(_ context.Context, _ *Client) ([]string, error) {
		return nil, errors.New("timeout")
	})
	if len(failed) != len(CommonRegions) {
		t.Errorf("expected %d failures, got %d", len(CommonRegions), len(failed))
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestAggregateRegions_EmptyResultsNotFailed(t *testing.T) {
	c := &Client{Region: "us-east-1"}
	items, failed := aggregateRegions(ctx(), c, func(_ context.Context, _ *Client) ([]string, error) {
		return nil, nil // no items, no error
	})
	if len(failed) != 0 {
		t.Errorf("expected no failures for empty results, got %v", failed)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func ctx() context.Context { return context.Background() }
