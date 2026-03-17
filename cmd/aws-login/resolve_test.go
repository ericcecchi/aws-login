package main

import (
	"bytes"
	"testing"
)

func TestResolveAccountByID(t *testing.T) {
	accounts := []AccountInfo{{AccountID: "123", AccountName: "Prod"}}
	acct, err := resolveAccount(accounts, "123-", &bytes.Buffer{}, true)
	if err != nil {
		t.Fatalf("resolveAccount error: %v", err)
	}
	if acct.AccountID != "123" {
		t.Fatalf("unexpected account: %+v", acct)
	}
}

func TestResolveAccountAmbiguous(t *testing.T) {
	accounts := []AccountInfo{
		{AccountID: "123", AccountName: "Prod"},
		{AccountID: "124", AccountName: "Prod-Secondary"},
	}
	_, err := resolveAccount(accounts, "pro", &bytes.Buffer{}, true)
	if err == nil {
		t.Fatalf("expected error for ambiguous account query")
	}
}

func TestResolveRoleByName(t *testing.T) {
	roles := []RoleInfo{{RoleName: "Admin"}, {RoleName: "Read"}}
	role, err := resolveRole(roles, "admin", &bytes.Buffer{}, true)
	if err != nil {
		t.Fatalf("resolveRole error: %v", err)
	}
	if role.RoleName != "Admin" {
		t.Fatalf("unexpected role: %+v", role)
	}
}

func TestResolveRoleMissingInNonInteractive(t *testing.T) {
	roles := []RoleInfo{{RoleName: "Admin"}}
	_, err := resolveRole(roles, "", &bytes.Buffer{}, true)
	if err == nil {
		t.Fatalf("expected error for missing role in non-interactive mode")
	}
}
