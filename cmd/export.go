package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"

	awsclient "github.com/awslens/awslens/internal/aws"
)

// maxExportConcurrency limits parallel service fetches to avoid API throttling.
const maxExportConcurrency = 5

// validServices lists all exportable service names.
var validServices = []string{
	"ec2", "lambda", "s3", "rds", "dynamodb", "apigateway", "ecs", "ecr",
	"stepfunctions", "elb", "route53", "secretsmanager", "ssm", "sqs", "sns",
	"cloudwatch", "cloudformation", "costs", "elasticache", "opensearch",
	"msk", "glue", "athena", "codecommit", "codepipeline", "codebuild",
	"eventbridge", "waf",
}

// runHeadlessExport fetches service data and writes it to stdout in the given format.
func runHeadlessExport(format, service string) error {
	if profile == "" {
		return fmt.Errorf("--profile is required when using --output")
	}
	format = strings.ToLower(format)
	if format != "json" && format != "csv" {
		return fmt.Errorf("--output must be 'json' or 'csv', got %q", format)
	}

	services := validServices
	if service != "" {
		s := strings.ToLower(service)
		found := false
		for _, v := range validServices {
			if v == s {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown service %q; valid: %s", service, strings.Join(validServices, ", "))
		}
		services = []string{s}
	}

	client, err := awsclient.New(profile, region)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	ctx := context.Background()

	data := fetchServices(ctx, client, services)

	switch format {
	case "json":
		return writeJSONOut(os.Stdout, data)
	case "csv":
		return writeCSVOut(os.Stdout, data)
	}
	return nil
}

type serviceData struct {
	Service string             `json:"service"`
	Items   []map[string]interface{} `json:"items"`
}

func fetchServices(ctx context.Context, client *awsclient.Client, services []string) []serviceData {
	type indexedResult struct {
		index int
		data  serviceData
	}

	sem := make(chan struct{}, maxExportConcurrency)
	var mu sync.Mutex
	var collected []indexedResult
	var wg sync.WaitGroup

	for i, svc := range services {
		wg.Add(1)
		go func(idx int, s string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			items := fetchService(ctx, client, s)
			if len(items) > 0 {
				mu.Lock()
				collected = append(collected, indexedResult{index: idx, data: serviceData{Service: s, Items: items}})
				mu.Unlock()
			}
		}(i, svc)
	}
	wg.Wait()

	// Sort by original order to ensure deterministic output
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].index < collected[j].index
	})
	results := make([]serviceData, 0, len(collected))
	for _, r := range collected {
		results = append(results, r.data)
	}
	return results
}

func fetchService(ctx context.Context, client *awsclient.Client, svc string) []map[string]interface{} {
	var raw interface{}
	switch svc {
	case "ec2":
		raw = awsclient.AllRegionsInstances(ctx, client)
	case "lambda":
		raw = awsclient.AllRegionsFunctions(ctx, client)
	case "s3":
		buckets, err := client.ListBuckets(ctx)
		if err != nil {
			return nil
		}
		raw = buckets
	case "rds":
		raw = awsclient.AllRegionsDBInstances(ctx, client)
	case "dynamodb":
		raw = awsclient.AllRegionsDynamoTables(ctx, client)
	case "apigateway":
		raw = awsclient.AllRegionsRestAPIs(ctx, client)
	case "ecs":
		raw = awsclient.AllRegionsClusters(ctx, client)
	case "ecr":
		raw = awsclient.AllRegionsECRRepos(ctx, client)
	case "stepfunctions":
		raw = awsclient.AllRegionsStateMachines(ctx, client)
	case "elb":
		raw = awsclient.AllRegionsLoadBalancers(ctx, client)
	case "route53":
		zones, err := client.ListHostedZones(ctx)
		if err != nil {
			return nil
		}
		raw = zones
	case "secretsmanager":
		raw = awsclient.AllRegionsSecrets(ctx, client)
	case "ssm":
		raw = awsclient.AllRegionsSSMParams(ctx, client)
	case "sqs":
		raw = awsclient.AllRegionsQueues(ctx, client)
	case "sns":
		raw = awsclient.AllRegionsTopics(ctx, client)
	case "cloudwatch":
		raw = awsclient.AllRegionsAlarms(ctx, client)
	case "cloudformation":
		raw = awsclient.AllRegionsStacks(ctx, client)
	case "costs":
		entries, total, err := client.GetMonthlyCosts(ctx, monthsBack)
		if err != nil {
			return nil
		}
		// Append total as metadata in the items
		items := toMaps(entries)
		items = append(items, map[string]interface{}{"_total": total})
		return items
	case "elasticache":
		raw = awsclient.AllRegionsCacheClusters(ctx, client)
	case "opensearch":
		raw = awsclient.AllRegionsOSDomains(ctx, client)
	case "msk":
		raw = awsclient.AllRegionsMSKClusters(ctx, client)
	case "glue":
		raw = awsclient.AllRegionsGlueDatabases(ctx, client)
	case "athena":
		raw = awsclient.AllRegionsAthenaWorkgroups(ctx, client)
	case "codecommit":
		raw = awsclient.AllRegionsCodeRepos(ctx, client)
	case "codepipeline":
		raw = awsclient.AllRegionsPipelines(ctx, client)
	case "codebuild":
		raw = awsclient.AllRegionsBuildProjects(ctx, client)
	case "eventbridge":
		raw = awsclient.AllRegionsEBRules(ctx, client)
	case "waf":
		raw = awsclient.AllRegionsWAFWebACLs(ctx, client)
	default:
		return nil
	}
	return toMaps(raw)
}

// toMaps converts a slice of structs to []map[string]interface{} via JSON round-trip.
func toMaps(v interface{}) []map[string]interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(b, &items); err != nil {
		return nil
	}
	return items
}

func writeJSONOut(w io.Writer, data []serviceData) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func writeCSVOut(w io.Writer, data []serviceData) error {
	if len(data) == 0 {
		return nil
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Collect all unique keys across all items for headers
	headerSet := map[string]bool{"service": true}
	for _, sd := range data {
		for _, item := range sd.Items {
			for k := range item {
				headerSet[k] = true
			}
		}
	}
	headers := make([]string, 0, len(headerSet))
	for k := range headerSet {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	if err := cw.Write(headers); err != nil {
		return err
	}
	for _, sd := range data {
		for _, item := range sd.Items {
			row := make([]string, len(headers))
			for i, h := range headers {
				if h == "service" {
					row[i] = sd.Service
				} else if v, ok := item[h]; ok {
					row[i] = fmt.Sprintf("%v", v)
				}
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
	}
	return cw.Error()
}
