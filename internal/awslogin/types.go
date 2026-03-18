package awslogin

import "time"

type SessionInfo struct {
	Name      string
	StartURL  string
	Region    string
	LoginArgs []string
}

type AccountInfo struct {
	AccountID   string
	AccountName string
	Email       string
}

type RoleInfo struct {
	RoleName string
}

type RoleCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      *time.Time
}

type ProfileInfo struct {
	SSOSession  string
	SSOStart    string
	SSORegion   string
	Region      string
	AccountID   string
	RoleName    string
	EKSRoleARN  string
}

type SSOCacheEntry struct {
	StartURL    string `json:"startUrl"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
}

type Args struct {
	Role           string
	Account        string
	Profile        string
	SSOSession     string
	Region         string
	KubeContext    string
	Doctor         bool
	NoKube         bool
	NonInteractive bool
	PrintEnv       bool
	SetProfile     bool
	ShellInit      bool
	Version        bool
}
