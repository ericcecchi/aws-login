package awslogin

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const awsLoginCacheDir = "~/.aws-login/cache"

type accountsCache struct {
	CachedAt time.Time     `json:"cached_at"`
	Items    []AccountInfo `json:"items"`
}

type rolesCache struct {
	CachedAt time.Time  `json:"cached_at"`
	Items    []RoleInfo `json:"items"`
}

func sessionCacheKey(session SessionInfo) string {
	if strings.TrimSpace(session.Name) != "" {
		return "session:" + session.Name
	}
	return fmt.Sprintf("start:%s|region:%s", session.StartURL, session.Region)
}

func listAccountsCached(accessToken, region, sessionKey string) ([]AccountInfo, error) {
	cachePath := cacheFilePath("accounts", sessionKey)
	cached, ok := loadAccountsCache(cachePath)
	if ok {
		go func() {
			fresh, err := listAccounts(accessToken, region)
			if err != nil {
				return
			}
			_ = saveAccountsCache(cachePath, fresh)
		}()
		return cached, nil
	}

	fresh, err := listAccounts(accessToken, region)
	if err != nil {
		return nil, err
	}
	_ = saveAccountsCache(cachePath, fresh)
	return fresh, nil
}

func listRolesCached(accessToken, region, accountID, sessionKey string) ([]RoleInfo, error) {
	cachePath := cacheFilePath("roles", sessionKey+"|account:"+accountID)
	cached, ok := loadRolesCache(cachePath)
	if ok {
		go func() {
			fresh, err := listRoles(accessToken, region, accountID)
			if err != nil {
				return
			}
			_ = saveRolesCache(cachePath, fresh)
		}()
		return cached, nil
	}

	fresh, err := listRoles(accessToken, region, accountID)
	if err != nil {
		return nil, err
	}
	_ = saveRolesCache(cachePath, fresh)
	return fresh, nil
}

func cacheFilePath(kind, key string) string {
	hash := sha1.Sum([]byte(key))
	return filepath.Join(expandPath(awsLoginCacheDir), fmt.Sprintf("%s-%x.json", kind, hash[:]))
}

func loadAccountsCache(path string) ([]AccountInfo, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var payload accountsCache
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}
	if payload.Items == nil {
		return []AccountInfo{}, true
	}
	return payload.Items, true
}

func saveAccountsCache(path string, accounts []AccountInfo) error {
	payload := accountsCache{CachedAt: time.Now().UTC(), Items: accounts}
	return writeCacheFile(path, payload)
}

func loadRolesCache(path string) ([]RoleInfo, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var payload rolesCache
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false
	}
	if payload.Items == nil {
		return []RoleInfo{}, true
	}
	return payload.Items, true
}

func saveRolesCache(path string, roles []RoleInfo) error {
	payload := rolesCache{CachedAt: time.Now().UTC(), Items: roles}
	return writeCacheFile(path, payload)
}

func writeCacheFile(path string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
