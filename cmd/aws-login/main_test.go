package main

import "testing"

func TestResolveRequestedProfilePrefersExplicit(t *testing.T) {
	profile, err := resolveRequestedProfile(Args{Profile: "dev", Target: "dev"})
	if err != nil {
		t.Fatalf("resolveRequestedProfile error: %v", err)
	}
	if profile != "dev" {
		t.Fatalf("expected explicit profile, got %q", profile)
	}
}

func TestResolveRequestedProfileNoSelection(t *testing.T) {
	profile, err := resolveRequestedProfile(Args{})
	if err != nil {
		t.Fatalf("resolveRequestedProfile error: %v", err)
	}
	if profile != "" {
		t.Fatalf("expected empty profile, got %q", profile)
	}
}

func TestResolveRequestedProfileForAccountRoleSwitch(t *testing.T) {
	profile, err := resolveRequestedProfile(Args{Account: "123", Role: "Admin"})
	if err != nil {
		t.Fatalf("resolveRequestedProfile error: %v", err)
	}
	if profile != "" {
		t.Fatalf("expected no profile selection, got %q", profile)
	}
}

func TestResolveRequestedProfileMismatchError(t *testing.T) {
	_, err := resolveRequestedProfile(Args{Profile: "dev", Target: "prod"})
	if err == nil {
		t.Fatalf("expected profile mismatch error")
	}
}
