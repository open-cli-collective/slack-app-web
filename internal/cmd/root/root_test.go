package root

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/open-cli-collective/slack-app-web/internal/cmd/cmdutil"
	"github.com/open-cli-collective/slack-app-web/internal/exitcode"
	"github.com/open-cli-collective/slack-app-web/internal/secretout"
	"github.com/open-cli-collective/slack-app-web/internal/slackweb"
)

type fakeBrowser struct {
	opened    *int
	signedIn  bool
	apps      []slackweb.App
	state     slackweb.InstallState
	acted     bool
	botToken  string
	pair      slackweb.ConfigToken
	iconPath  string
	reinstall bool
}

func (f *fakeBrowser) SignedIn() (bool, error)           { return f.signedIn, nil }
func (f *fakeBrowser) Login(string, time.Duration) error { return nil }
func (f *fakeBrowser) ListApps() ([]slackweb.App, error) { return f.apps, nil }
func (f *fakeBrowser) BotToken(string) (string, error)   { return f.botToken, nil }
func (f *fakeBrowser) UpdateIcon(_, png string) error    { f.iconPath = png; return nil }
func (f *fakeBrowser) Close()                            {}
func (f *fakeBrowser) CreateConfigToken(string, bool, time.Time) (slackweb.ConfigToken, error) {
	return f.pair, nil
}
func (f *fakeBrowser) GetConfigToken(string, time.Time) (slackweb.ConfigToken, error) {
	return f.pair, nil
}
func (f *fakeBrowser) Install(_ string, reinstall bool) (slackweb.InstallState, bool, error) {
	f.reinstall = reinstall
	return f.state, f.acted, nil
}

func run(t *testing.T, fake *fakeBrowser, terminal bool, args ...string) (string, error) {
	t.Helper()
	opened := 0
	fake.opened = &opened
	var out, errOut bytes.Buffer
	opts := cmdutil.Default()
	opts.Stdout, opts.Stderr = &out, &errOut
	opts.StdoutIsTerminal = func() bool { return terminal }
	opts.Now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	opts.ProfileDir = t.TempDir()
	opts.Open = func(slackweb.Options) (cmdutil.Browser, error) { opened++; return fake, nil }
	cmd := NewCmd(opts)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err := cmd.Execute()
	return out.String(), err
}

func TestSecretsAreRefusedOnATerminalBeforeTheBrowserOpens(t *testing.T) {
	for _, args := range [][]string{{"bot-tokens", "get", "A0EXAMPLE01"}, {"config-tokens", "create"}, {"config-tokens", "get"}} {
		fake := &fakeBrowser{botToken: "xoxb-1", pair: slackweb.ConfigToken{Token: "t"}}
		out, err := run(t, fake, true, args...)
		if !errors.Is(err, secretout.ErrTerminal) {
			t.Fatalf("%v: err = %v, want ErrTerminal", args, err)
		}
		if out != "" || *fake.opened != 0 {
			t.Fatalf("%v: printed %q, opened browser %d times", args, out, *fake.opened)
		}
	}
}

func TestBotTokenGoesToAPipeExactly(t *testing.T) {
	out, err := run(t, &fakeBrowser{botToken: "xoxb-1-2-abc"}, false, "bot-tokens", "get", "A0EXAMPLE01")
	if err != nil || out != "xoxb-1-2-abc" {
		t.Fatalf("out %q err %v", out, err)
	}
}

func TestConfigTokenEmitsTheContract(t *testing.T) {
	fake := &fakeBrowser{pair: slackweb.ConfigToken{Token: "xoxe.xoxp-1-a", RefreshToken: "xoxe-1-r", Exp: 1_800_043_200}}
	out, err := run(t, fake, false, "config-tokens", "create")
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"token":"xoxe.xoxp-1-a","refresh_token":"xoxe-1-r","exp":1800043200}`; out != want {
		t.Fatalf("out %q, want %q", out, want)
	}
}

func TestAppIDIsValidated(t *testing.T) {
	fake := &fakeBrowser{}
	_, err := run(t, fake, false, "apps", "install", "not-an-app")
	if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
		t.Fatalf("err %v (code %d), opened %d", err, exitcode.Of(err), *fake.opened)
	}
}

func TestAppsList(t *testing.T) {
	fake := &fakeBrowser{apps: []slackweb.App{{Name: "Build Bot", ID: "A0EXAMPLE02", Workspace: "Example Co"}}}
	out, err := run(t, fake, true, "apps", "list")
	if err != nil {
		t.Fatal(err)
	}
	if want := "ID | NAME | WORKSPACE\nA0EXAMPLE02 | Build Bot | Example Co\n"; out != want {
		t.Fatalf("out %q", out)
	}
}

func TestInstallLeavesAnInstalledAppAlone(t *testing.T) {
	fake := &fakeBrowser{state: slackweb.InstallState{Installed: true}}
	out, err := run(t, fake, true, "apps", "install", "A0EXAMPLE01")
	if err != nil || out != "A0EXAMPLE01 is already installed\n" || fake.reinstall {
		t.Fatalf("out %q err %v reinstall %v", out, err, fake.reinstall)
	}
	fake = &fakeBrowser{state: slackweb.InstallState{Installed: true}, acted: true}
	out, err = run(t, fake, true, "apps", "install", "A0EXAMPLE01", "--reinstall")
	if err != nil || out != "Reinstalled A0EXAMPLE01\n" || !fake.reinstall {
		t.Fatalf("out %q err %v reinstall %v", out, err, fake.reinstall)
	}
}

func TestInstallPendingApprovalFails(t *testing.T) {
	fake := &fakeBrowser{state: slackweb.InstallState{PendingApproval: true}, acted: true}
	_, err := run(t, fake, true, "apps", "install", "A0EXAMPLE01")
	if err == nil || !strings.Contains(err.Error(), "admin must approve") {
		t.Fatalf("err %v", err)
	}
}

func TestMeExitsAuthWhenSignedOut(t *testing.T) {
	out, err := run(t, &fakeBrowser{}, true, "me")
	if exitcode.Of(err) != exitcode.Auth || !strings.Contains(out, "Signed in: no") {
		t.Fatalf("out %q err %v", out, err)
	}
}

func TestIconsRejectsANonPNG(t *testing.T) {
	fake := &fakeBrowser{}
	_, err := run(t, fake, true, "icons", "update", "A0EXAMPLE01", "--file", "root.go")
	if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
		t.Fatalf("err %v opened %d", err, *fake.opened)
	}
}
