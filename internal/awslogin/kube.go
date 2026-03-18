package awslogin

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

func maybeSwitchKubeAuto(accountID, region, explicitContext, profileName, eksRoleARN string, w io.Writer) {
	if !commandExists("kubectl") {
		logLine(w, "⚠️  kubectl not found; skipping context switch")
		return
	}
	if explicitContext != "" {
		_ = switchContextWithKubectl(explicitContext, w)
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
		logLine(w, "⚠️  No EKS clusters found for this account")
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

	logLine(w, "Available Kubernetes contexts for this account:")
	for _, ctx := range matches {
		logLine(w, fmt.Sprintf("- %s", ctx))
	}

	chosen := matches[0]
	if err := switchContextWithKubectl(chosen, w); err != nil {
		logLine(w, fmt.Sprintf("⚠️  Failed to switch context: %v", err))
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
