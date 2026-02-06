package main

import "time"

type Config struct {
	Defaults DefaultsConfig         `toml:"defaults"`
	Aliases  map[string]AliasConfig `toml:"aliases"`
}

type DefaultsConfig struct {
	SSOSession string `toml:"sso_session"`
}

type AliasConfig struct {
	AccountID     string            `toml:"account_id"`
	DefaultRole   string            `toml:"default_role"`
	Roles         []string          `toml:"roles"`
	Region        string            `toml:"region"`
	KubeContext   string            `toml:"kube_context"`
	ProfileByRole map[string]string `toml:"profile_by_role"`
}

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
	SSOSession string
	SSOStart   string
	SSORegion  string
	Region     string
	AccountID  string
	RoleName   string
}

type SSOCacheEntry struct {
	StartURL    string `json:"startUrl"`
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
}

type Args struct {
	Target         string
	Role           string
	Account        string
	RoleFlag       string
	Alias          string
	Profile        string
	SSOSession     string
	Region         string
	KubeContext    string
	NoKube         bool
	NonInteractive bool
	PrintEnv       bool
	Version        bool
}
