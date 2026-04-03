package awslogin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const kubePrefsPath = "~/.aws-login/kube-prefs.json"

func loadKubePref(accountID string) (string, error) {
	path := expandPath(kubePrefsPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var prefs map[string]string
	if err := json.Unmarshal(data, &prefs); err != nil {
		return "", err
	}
	return prefs[accountID], nil
}

func saveKubePref(accountID, contextName string) error {
	path := expandPath(kubePrefsPath)
	prefs := map[string]string{}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &prefs)
	}
	prefs[accountID] = contextName
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func maybeSwitchKubeAuto(accountID, region, explicitContext, profileName, eksRoleARN string, nonInteractive bool, w io.Writer) {
	if !commandExists("kubectl") {
		logLine(w, "⚠️  kubectl not found; skipping context switch")
		return
	}
	if explicitContext != "" {
		if err := switchContextWithKubectl(explicitContext, w); err != nil {
			logLine(w, fmt.Sprintf("⚠️  Failed to switch context: %v", err))
			return
		}
		// Remember this choice for the account so future logins default to it.
		if accountID != "" {
			_ = saveKubePref(accountID, explicitContext)
		}
		return
	}
	if !commandExists("aws") {
		logLine(w, "⚠️  AWS CLI not found; skipping kube context discovery")
		return
	}

	clusters, err := listEKSClusters(profileName, region)
	if err != nil {
		logLine(w, fmt.Sprintf("⚠️  Unable to list EKS clusters: %v", err))
		return
	}
	if len(clusters) == 0 {
		return
	}

	for _, cluster := range clusters {
		if err := updateKubeconfig(cluster, profileName, region, eksRoleARN); err != nil {
			logLine(w, fmt.Sprintf("⚠️  Failed to update kubeconfig for %s: %v", cluster, err))
		}
	}

	contexts, err := listKubeContexts()
	if err != nil {
		logLine(w, fmt.Sprintf("⚠️  Unable to list kubectl contexts: %v", err))
		return
	}

	matches := filterContexts(contexts, accountID, clusters)
	if len(matches) == 0 {
		logLine(w, "⚠️  No Kubernetes contexts matched this account")
		return
	}

	// Single match: switch automatically.
	if len(matches) == 1 {
		if err := switchContextWithKubectl(matches[0], w); err != nil {
			logLine(w, fmt.Sprintf("⚠️  Failed to switch context: %v", err))
		}
		return
	}

	// Multiple matches: check for a saved preference first.
	if accountID != "" {
		if pref, err := loadKubePref(accountID); err == nil && pref != "" {
			for _, ctx := range matches {
				if ctx == pref {
					logLine(w, fmt.Sprintf("ℹ️  Using saved Kubernetes context: %s", pref))
					if err := switchContextWithKubectl(pref, w); err != nil {
						logLine(w, fmt.Sprintf("⚠️  Failed to switch context: %v", err))
					}
					return
				}
			}
			// Saved preference no longer matches any available context; fall through.
			logLine(w, fmt.Sprintf("ℹ️  Saved context %q is no longer available; please choose a new one", pref))
		}
	}

	// Non-interactive: skip rather than block.
	if nonInteractive {
		logLine(w, "⚠️  Multiple Kubernetes contexts found; skipping auto-switch (use --kube-context to specify one)")
		return
	}

	// Interactive: let the user pick and remember the choice.
	logLine(w, "Multiple Kubernetes contexts are available for this account:")
	for _, ctx := range matches {
		logLine(w, fmt.Sprintf("  %s", ctx))
	}
	chosen, err := chooseInteractive(matches, "Kubernetes context", func(s string) string { return s })
	if err != nil {
		logLine(w, fmt.Sprintf("⚠️  Context selection cancelled: %v", err))
		return
	}
	if err := switchContextWithKubectl(chosen, w); err != nil {
		logLine(w, fmt.Sprintf("⚠️  Failed to switch context: %v", err))
		return
	}
	if accountID != "" {
		if err := saveKubePref(accountID, chosen); err != nil {
			logLine(w, fmt.Sprintf("⚠️  Failed to save context preference: %v", err))
		} else {
			logLine(w, fmt.Sprintf("ℹ️  Saved %s as your default context for this account", chosen))
		}
	}
}

func listEKSClusters(profileName, region string) ([]string, error) {
	args := []string{"eks", "list-clusters", "--region", region, "--output", "json", "--no-cli-pager"}
	if profileName != "" {
		args = append(args, "--profile", profileName)
	}
	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("aws eks list-clusters failed")
	}
	var response struct {
		Clusters []string `json:"clusters"`
	}
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse cluster list")
	}
	return response.Clusters, nil
}

func updateKubeconfig(cluster, profileName, region, roleARN string) error {
	args := []string{"eks", "update-kubeconfig", "--name", cluster, "--region", region, "--alias", cluster}
	if profileName != "" {
		args = append(args, "--profile", profileName)
	}
	if roleARN != "" {
		args = append(args, "--role-arn", roleARN)
	}
	cmd := exec.Command("aws", args...)
	return cmd.Run()
}

func listKubeContexts() ([]string, error) {
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("kubectl config get-contexts failed")
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	contexts := []string{}
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			contexts = append(contexts, strings.TrimSpace(line))
		}
	}
	sort.Strings(contexts)
	return contexts, nil
}

func filterContexts(contexts []string, accountID string, clusters []string) []string {
	matches := []string{}
	clusterSet := map[string]struct{}{}
	for _, cluster := range clusters {
		clusterSet[cluster] = struct{}{}
	}
	for _, ctx := range contexts {
		if accountID != "" && strings.Contains(ctx, accountID) {
			matches = append(matches, ctx)
			continue
		}
		for cluster := range clusterSet {
			if strings.Contains(ctx, cluster) {
				matches = append(matches, ctx)
				break
			}
		}
	}
	return matches
}

func switchContextWithKubectl(context string, w io.Writer) error {
	cmd := exec.Command("kubectl", "config", "use-context", context)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("could not switch context automatically")
	}
	logLine(w, fmt.Sprintf("✓ Switched to %s", context))
	return nil
}
