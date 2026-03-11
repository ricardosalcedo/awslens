package aws

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Profile struct {
	Name          string
	Region        string
	RoleARN       string
	SourceProfile string
	SSOAccount    string
	SSORole       string
}

// LoadProfiles reads ~/.aws/config and ~/.aws/credentials to build a profile list.
func LoadProfiles() []Profile {
	profiles := map[string]*Profile{}

	// parse ~/.aws/config
	parseConfig(filepath.Join(os.Getenv("HOME"), ".aws", "config"), profiles, true)
	// parse ~/.aws/credentials (adds profiles not in config)
	parseConfig(filepath.Join(os.Getenv("HOME"), ".aws", "credentials"), profiles, false)

	var result []Profile
	// default first
	if p, ok := profiles["default"]; ok {
		result = append(result, *p)
	}
	for name, p := range profiles {
		if name != "default" {
			result = append(result, *p)
		}
	}
	return result
}

func parseConfig(path string, profiles map[string]*Profile, isConfig bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var current *Profile
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			name := strings.Trim(line, "[]")
			name = strings.TrimPrefix(name, "profile ")
			if _, ok := profiles[name]; !ok {
				profiles[name] = &Profile{Name: name}
			}
			current = profiles[name]
			continue
		}
		if current == nil {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "region":
			current.Region = v
		case "role_arn":
			current.RoleARN = v
		case "source_profile":
			current.SourceProfile = v
		case "sso_account_id":
			current.SSOAccount = v
		case "sso_role_name":
			current.SSORole = v
		}
	}
}

// SaveProfile appends a new profile to ~/.aws/config and optionally ~/.aws/credentials.
func SaveProfile(name, region, accessKey, secretKey, roleARN, sourceProfile, ssoStartURL, ssoRegion, ssoAccount, ssoRole string) error {
	home := os.Getenv("HOME")
	awsDir := filepath.Join(home, ".aws")
	if err := os.MkdirAll(awsDir, 0700); err != nil {
		return err
	}

	// write config section
	configPath := filepath.Join(awsDir, "config")
	section := "\n"
	if name == "default" {
		section += "[default]\n"
	} else {
		section += fmt.Sprintf("[profile %s]\n", name)
	}
	if region != "" {
		section += fmt.Sprintf("region = %s\n", region)
	}
	if roleARN != "" {
		section += fmt.Sprintf("role_arn = %s\n", roleARN)
		if sourceProfile != "" {
			section += fmt.Sprintf("source_profile = %s\n", sourceProfile)
		}
	}
	if ssoStartURL != "" {
		section += fmt.Sprintf("sso_start_url = %s\n", ssoStartURL)
		section += fmt.Sprintf("sso_region = %s\n", ssoRegion)
		section += fmt.Sprintf("sso_account_id = %s\n", ssoAccount)
		section += fmt.Sprintf("sso_role_name = %s\n", ssoRole)
	}

	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(section); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// write credentials if access key provided
	if accessKey != "" && secretKey != "" {
		credPath := filepath.Join(awsDir, "credentials")
		credSection := "\n"
		if name == "default" {
			credSection += "[default]\n"
		} else {
			credSection += fmt.Sprintf("[%s]\n", name)
		}
		credSection += fmt.Sprintf("aws_access_key_id = %s\n", accessKey)
		credSection += fmt.Sprintf("aws_secret_access_key = %s\n", secretKey)

		cf, err := os.OpenFile(credPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		if _, err := cf.WriteString(credSection); err != nil {
			cf.Close()
			return err
		}
		cf.Close()
	}
	return nil
}
