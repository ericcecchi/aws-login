package awslogin

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

func normalizeAccountName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")
	return s
}

func resolveAccount(accounts []AccountInfo, query string, w io.Writer, nonInteractive bool) (AccountInfo, error) {
	if len(accounts) == 0 {
		return AccountInfo{}, fmt.Errorf("no AWS accounts found for this SSO session")
	}

	if query != "" {
		idQuery := stripNonDigits(query)
		var idMatches []AccountInfo
		for _, acct := range accounts {
			if acct.AccountID == idQuery {
				idMatches = append(idMatches, acct)
			}
		}
		if len(idMatches) == 1 {
			return idMatches[0], nil
		}

		var exact []AccountInfo
		for _, acct := range accounts {
			if strings.EqualFold(acct.AccountName, query) {
				exact = append(exact, acct)
			}
		}
		if len(exact) == 1 {
			return exact[0], nil
		}

		normalizedQuery := normalizeAccountName(query)
		var normalized []AccountInfo
		for _, acct := range accounts {
			if normalizeAccountName(acct.AccountName) == normalizedQuery {
				normalized = append(normalized, acct)
			}
		}
		if len(normalized) == 1 {
			return normalized[0], nil
		}

		var partial []AccountInfo
		for _, acct := range accounts {
			if strings.Contains(strings.ToLower(acct.AccountName), strings.ToLower(query)) {
				partial = append(partial, acct)
			}
		}
		if len(partial) == 1 {
			return partial[0], nil
		}
		if len(partial) > 1 {
			names := make([]string, 0, len(partial))
			for _, acct := range partial {
				names = append(names, acct.AccountName)
			}
			sort.Strings(names)
			return AccountInfo{}, fmt.Errorf("account query '%s' matched multiple accounts: %s", query, strings.Join(names, ", "))
		}
		return AccountInfo{}, fmt.Errorf("account '%s' not found", query)
	}

	if nonInteractive {
		return AccountInfo{}, fmt.Errorf("missing account in non-interactive mode")
	}

	sorted := make([]AccountInfo, len(accounts))
	copy(sorted, accounts)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].AccountName) < strings.ToLower(sorted[j].AccountName)
	})
	logLine(w, "Select an AWS account:")
	selected, err := chooseInteractive(sorted, "AWS account", func(acct AccountInfo) string {
		return fmt.Sprintf("%s (%s)", acct.AccountName, acct.AccountID)
	})
	if err != nil {
		return AccountInfo{}, err
	}
	return selected, nil
}

func resolveRole(roles []RoleInfo, query string, w io.Writer, nonInteractive bool) (RoleInfo, error) {
	if len(roles) == 0 {
		return RoleInfo{}, fmt.Errorf("no roles available for the selected account")
	}

	if query != "" {
		var exact []RoleInfo
		for _, role := range roles {
			if strings.EqualFold(role.RoleName, query) {
				exact = append(exact, role)
			}
		}
		if len(exact) == 1 {
			return exact[0], nil
		}

		var partial []RoleInfo
		for _, role := range roles {
			if strings.Contains(strings.ToLower(role.RoleName), strings.ToLower(query)) {
				partial = append(partial, role)
			}
		}
		if len(partial) == 1 {
			return partial[0], nil
		}
		if len(partial) > 1 {
			names := make([]string, 0, len(partial))
			for _, role := range partial {
				names = append(names, role.RoleName)
			}
			sort.Strings(names)
			return RoleInfo{}, fmt.Errorf("role query '%s' matched multiple roles: %s", query, strings.Join(names, ", "))
		}
		return RoleInfo{}, fmt.Errorf("role '%s' not found", query)
	}

	if nonInteractive {
		return RoleInfo{}, fmt.Errorf("missing role in non-interactive mode")
	}

	sorted := make([]RoleInfo, len(roles))
	copy(sorted, roles)
	sort.Slice(sorted, func(i, j int) bool {
		return strings.ToLower(sorted[i].RoleName) < strings.ToLower(sorted[j].RoleName)
	})
	logLine(w, "Select a role:")
	selected, err := chooseInteractive(sorted, "Role", func(role RoleInfo) string {
		return role.RoleName
	})
	if err != nil {
		return RoleInfo{}, err
	}
	return selected, nil
}
