package awslogin

import "testing"

func TestProfileFlagIsUsed(t *testing.T) {
	args, err := parseArgs([]string{"--profile", "dev"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Profile != "dev" {
		t.Fatalf("expected profile dev, got %q", args.Profile)
	}
}

func TestAccountRoleFlagsNoProfile(t *testing.T) {
	args, err := parseArgs([]string{"--account", "123", "--role", "Admin"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "123" || args.Role != "Admin" || args.Profile != "" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestPositionalAccountAndRoleArgs(t *testing.T) {
	args, err := parseArgs([]string{"myaccount", "admin"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "myaccount" || args.Role != "admin" {
		t.Fatalf("expected account=myaccount role=admin, got %+v", args)
	}
}
