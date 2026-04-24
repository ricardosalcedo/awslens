package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtypes "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// ── SSM mock ─────────────────────────────────────────────────────────────────

type mockSSM struct {
	out *ssm.DescribeParametersOutput
	err error
}

func (m *mockSSM) DescribeParameters(ctx context.Context, params *ssm.DescribeParametersInput, optFns ...func(*ssm.Options)) (*ssm.DescribeParametersOutput, error) {
	return m.out, m.err
}

// ── Secrets Manager mock ─────────────────────────────────────────────────────

type mockSecretsManager struct {
	out *secretsmanager.ListSecretsOutput
	err error
}

func (m *mockSecretsManager) ListSecrets(ctx context.Context, params *secretsmanager.ListSecretsInput, optFns ...func(*secretsmanager.Options)) (*secretsmanager.ListSecretsOutput, error) {
	return m.out, m.err
}

// ── SSM tests ────────────────────────────────────────────────────────────────

func TestListSSMParamsWithAPI(t *testing.T) {
	ts := time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockSSM
		want    int
		wantErr bool
		check   func(t *testing.T, params []SSMParam)
	}{
		{
			name: "full field mapping",
			mock: &mockSSM{out: &ssm.DescribeParametersOutput{
				Parameters: []ssmtypes.ParameterMetadata{{
					Name:             aws.String("/app/db-host"),
					Type:             ssmtypes.ParameterTypeString,
					Version:          3,
					LastModifiedDate: &ts,
				}},
			}},
			want: 1,
			check: func(t *testing.T, params []SSMParam) {
				p := params[0]
				if p.Name != "/app/db-host" {
					t.Errorf("Name = %q, want %q", p.Name, "/app/db-host")
				}
				if p.Type != "String" {
					t.Errorf("Type = %q, want %q", p.Type, "String")
				}
				if p.Version != 3 {
					t.Errorf("Version = %d, want 3", p.Version)
				}
				if p.LastModified != "2025-07-10" {
					t.Errorf("LastModified = %q, want %q", p.LastModified, "2025-07-10")
				}
			},
		},
		{
			name: "nil LastModifiedDate",
			mock: &mockSSM{out: &ssm.DescribeParametersOutput{
				Parameters: []ssmtypes.ParameterMetadata{{
					Name: aws.String("/app/key"),
					Type: ssmtypes.ParameterTypeSecureString,
				}},
			}},
			want: 1,
			check: func(t *testing.T, params []SSMParam) {
				if params[0].LastModified != "" {
					t.Errorf("LastModified = %q, want empty", params[0].LastModified)
				}
			},
		},
		{
			name: "multiple parameters",
			mock: &mockSSM{out: &ssm.DescribeParametersOutput{
				Parameters: []ssmtypes.ParameterMetadata{
					{Name: aws.String("/a"), Type: ssmtypes.ParameterTypeString},
					{Name: aws.String("/b"), Type: ssmtypes.ParameterTypeStringList},
					{Name: aws.String("/c"), Type: ssmtypes.ParameterTypeSecureString},
				},
			}},
			want: 3,
		},
		{
			name: "empty results",
			mock: &mockSSM{out: &ssm.DescribeParametersOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockSSM{err: errors.New("access denied")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := ListSSMParamsWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(params) != tt.want {
				t.Fatalf("got %d params, want %d", len(params), tt.want)
			}
			if tt.check != nil {
				tt.check(t, params)
			}
		})
	}
}

// ── Secrets Manager tests ────────────────────────────────────────────────────

func TestListSecretsWithAPI(t *testing.T) {
	changed := time.Date(2025, 5, 20, 0, 0, 0, 0, time.UTC)
	accessed := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name    string
		mock    *mockSecretsManager
		want    int
		wantErr bool
		check   func(t *testing.T, secrets []Secret)
	}{
		{
			name: "full field mapping",
			mock: &mockSecretsManager{out: &secretsmanager.ListSecretsOutput{
				SecretList: []smtypes.SecretListEntry{{
					Name:             aws.String("prod/db-password"),
					ARN:              aws.String("arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/db-password-AbCdEf"),
					Description:      aws.String("Production DB password"),
					LastChangedDate:  &changed,
					LastAccessedDate: &accessed,
				}},
			}},
			want: 1,
			check: func(t *testing.T, secrets []Secret) {
				s := secrets[0]
				if s.Name != "prod/db-password" {
					t.Errorf("Name = %q, want %q", s.Name, "prod/db-password")
				}
				if s.ARN != "arn:aws:secretsmanager:us-east-1:123456789012:secret:prod/db-password-AbCdEf" {
					t.Errorf("ARN = %q", s.ARN)
				}
				if s.Description != "Production DB password" {
					t.Errorf("Description = %q", s.Description)
				}
				if s.LastChanged != "2025-05-20" {
					t.Errorf("LastChanged = %q, want %q", s.LastChanged, "2025-05-20")
				}
				if s.LastAccessed != "2025-06-01" {
					t.Errorf("LastAccessed = %q, want %q", s.LastAccessed, "2025-06-01")
				}
			},
		},
		{
			name: "nil dates",
			mock: &mockSecretsManager{out: &secretsmanager.ListSecretsOutput{
				SecretList: []smtypes.SecretListEntry{{
					Name: aws.String("test/secret"),
				}},
			}},
			want: 1,
			check: func(t *testing.T, secrets []Secret) {
				if secrets[0].LastChanged != "" {
					t.Errorf("LastChanged = %q, want empty", secrets[0].LastChanged)
				}
				if secrets[0].LastAccessed != "" {
					t.Errorf("LastAccessed = %q, want empty", secrets[0].LastAccessed)
				}
			},
		},
		{
			name: "nil LastAccessedDate only",
			mock: &mockSecretsManager{out: &secretsmanager.ListSecretsOutput{
				SecretList: []smtypes.SecretListEntry{{
					Name:            aws.String("new-secret"),
					LastChangedDate: &changed,
				}},
			}},
			want: 1,
			check: func(t *testing.T, secrets []Secret) {
				if secrets[0].LastChanged != "2025-05-20" {
					t.Errorf("LastChanged = %q, want %q", secrets[0].LastChanged, "2025-05-20")
				}
				if secrets[0].LastAccessed != "" {
					t.Errorf("LastAccessed = %q, want empty", secrets[0].LastAccessed)
				}
			},
		},
		{
			name: "multiple secrets",
			mock: &mockSecretsManager{out: &secretsmanager.ListSecretsOutput{
				SecretList: []smtypes.SecretListEntry{
					{Name: aws.String("s1")},
					{Name: aws.String("s2")},
					{Name: aws.String("s3")},
				},
			}},
			want: 3,
		},
		{
			name: "empty results",
			mock: &mockSecretsManager{out: &secretsmanager.ListSecretsOutput{}},
			want: 0,
		},
		{
			name:    "API error",
			mock:    &mockSecretsManager{err: errors.New("throttled")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secrets, err := ListSecretsWithAPI(context.Background(), tt.mock)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if len(secrets) != tt.want {
				t.Fatalf("got %d secrets, want %d", len(secrets), tt.want)
			}
			if tt.check != nil {
				tt.check(t, secrets)
			}
		})
	}
}
