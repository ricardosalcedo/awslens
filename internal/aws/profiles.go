package aws

import (
	"bufio"
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
