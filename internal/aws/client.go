package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type Client struct {
	Config  aws.Config
	Profile string
	Region  string
}

// cliCacheEntry matches the AWS CLI credential cache format.
type cliCacheEntry struct {
	Credentials struct {
		AccessKeyID     string    `json:"AccessKeyId"`
		SecretAccessKey string    `json:"SecretAccessKey"`
		SessionToken    string    `json:"SessionToken"`
		Expiration      time.Time `json:"Expiration"`
	} `json:"Credentials"`
}

// loadFromCLICache tries to load valid credentials from ~/.aws/cli/cache/*.json
func loadFromCLICache() *cliCacheEntry {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".aws", "cli", "cache")
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return nil
	}
	now := time.Now().UTC().Add(5 * time.Minute) // 5 min buffer
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(cacheDir, e.Name()))
		if err != nil {
			continue
		}
		var entry cliCacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.Credentials.AccessKeyID != "" && entry.Credentials.Expiration.After(now) {
			return &entry
		}
	}
	return nil
}

// CommonRegions is the list of regions awslens scans by default.
var CommonRegions = []string{
	"us-east-1", "us-east-2", "us-west-1", "us-west-2",
	"eu-west-1", "eu-west-2", "eu-central-1",
	"ap-southeast-1", "ap-southeast-2", "ap-northeast-1",
	"sa-east-1", "ca-central-1",
}

func New(profile, region string) (*Client, error) {
	opts := []func(*config.LoadOptions) error{
		config.WithRegion("us-east-1"), // default; overridden below if set
	}
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	if region != "" {
		opts = append(opts, config.WithRegion(region))
	}

	// Use CLI cache only for profiles that don't have a role_arn
	// (role-based profiles must assume fresh to respect their policies)
	profiles := LoadProfiles()
	var profileHasRole bool
	for _, p := range profiles {
		if p.Name == profile && p.RoleARN != "" {
			profileHasRole = true
			break
		}
	}

	if !profileHasRole {
		if cached := loadFromCLICache(); cached != nil {
			opts = append(opts, config.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					cached.Credentials.AccessKeyID,
					cached.Credentials.SecretAccessKey,
					cached.Credentials.SessionToken,
				),
			))
		}
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &Client{Config: cfg, Profile: profile, Region: cfg.Region}, nil
}

// NewForRegion creates a client for a specific region using the same credentials.
func (c *Client) NewForRegion(region string) *Client {
	cfg := c.Config.Copy()
	cfg.Region = region
	return &Client{Config: cfg, Profile: c.Profile, Region: region}
}
