//go:build integration

package slackweb

import (
	"os"
	"testing"
	"time"
)

// These run the read-only browser flows against a real, signed-in profile:
//
//	SLAW_PROFILE_DIR=<profile> make test-integration
//
// They never print page content or tokens.
func openIntegration(t *testing.T) *Session {
	t.Helper()
	dir := os.Getenv("SLAW_PROFILE_DIR")
	if dir == "" {
		t.Skip("SLAW_PROFILE_DIR is not set")
	}
	s, err := Open(Options{ProfileDir: dir, Timeout: 30 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

func TestIntegrationListApps(t *testing.T) {
	s := openIntegration(t)
	apps, err := s.ListApps()
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) == 0 {
		t.Fatal("no apps listed; the profile's account should manage at least one")
	}
	for _, a := range apps {
		if !ValidAppID(a.ID) {
			t.Errorf("listed an invalid app id %q", a.ID)
		}
	}
}

func TestIntegrationInstallStateOfAnInstalledApp(t *testing.T) {
	appID := os.Getenv("SLAW_INSTALLED_APP_ID")
	if appID == "" {
		t.Skip("SLAW_INSTALLED_APP_ID is not set")
	}
	s := openIntegration(t)
	state, acted, err := s.Install(appID, false)
	if err != nil {
		t.Fatal(err)
	}
	if acted || !state.Installed {
		t.Fatalf("acted %v installed %v; an installed app must be left alone", acted, state.Installed)
	}
}
