package root

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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
	appToken  string
	tokenName string
	scopes    []string
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
func (f *fakeBrowser) CreateAppToken(_, name string, scopes []string) (string, error) {
	f.tokenName, f.scopes = name, scopes
	return f.appToken, nil
}
func (f *fakeBrowser) Install(_ string, reinstall bool) (slackweb.InstallState, bool, error) {
	f.reinstall = reinstall
	return f.state, f.acted, nil
}

func run(t *testing.T, fake *fakeBrowser, args ...string) (string, error) {
	t.Helper()
	out, _, err := runAll(t, fake, args...)
	return out, err
}

// runAll runs slaw and returns what it wrote to stdout and stderr.
func runAll(t *testing.T, fake *fakeBrowser, args ...string) (string, string, error) {
	t.Helper()
	t.Setenv("SLAW_PROFILE", "")
	t.Setenv("SLAW_PROFILE_DIR", "")
	opened := 0
	fake.opened = &opened
	var out, errOut bytes.Buffer
	opts := cmdutil.Default()
	opts.Stdout, opts.Stderr = &out, &errOut
	opts.Now = func() time.Time { return time.Unix(1_800_000_000, 0) }
	opts.ProfileDir = t.TempDir()
	opts.Open = func(slackweb.Options) (cmdutil.Browser, error) { opened++; return fake, nil }
	cmd := NewCmd(opts)
	cmd.SetArgs(args)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

const (
	botToken     = "xoxb-1-2-distinctive"
	accessToken  = "xoxe.xoxp-1-distinctive"
	refreshToken = "xoxe-1-distinctive"
	appToken     = "xapp-1-A0EXAMPLE01-1-distinctive"
)

func tokenFake() *fakeBrowser {
	return &fakeBrowser{botToken: botToken, appToken: appToken, pair: slackweb.ConfigToken{Token: accessToken, RefreshToken: refreshToken, Exp: 1_800_043_200}}
}

var secretCommands = [][]string{
	{"bot-tokens", "get", "A0EXAMPLE01"},
	{"config-tokens", "create"},
	{"config-tokens", "get"},
	{"app-tokens", "create", "A0EXAMPLE01", "--name", "socket-mode", "--scope", "connections:write"},
}

func assertNoToken(t *testing.T, args []string, outputs ...string) {
	t.Helper()
	for _, o := range outputs {
		for _, tok := range []string{botToken, accessToken, refreshToken, appToken} {
			if strings.Contains(o, tok) {
				t.Fatalf("%v: a token reached slaw's output: %q", args, o)
			}
		}
	}
}

func TestSecretsNeedASinkBeforeTheBrowserOpens(t *testing.T) {
	for _, args := range secretCommands {
		fake := tokenFake()
		out, errOut, err := runAll(t, fake, args...)
		if !errors.Is(err, secretout.ErrNoSink) || exitcode.Of(err) != exitcode.Usage {
			t.Fatalf("%v: err = %v, want a usage error naming the sinks", args, err)
		}
		if *fake.opened != 0 {
			t.Fatalf("%v: opened the browser without a sink", args)
		}
		assertNoToken(t, args, out, errOut, err.Error())
	}
}

func TestSecretsRefuseTwoSinks(t *testing.T) {
	for _, args := range secretCommands {
		fake := tokenFake()
		_, _, err := runAll(t, fake, append(args, "--to-file", filepath.Join(t.TempDir(), "x"), "--to-command", "cat")...)
		if !errors.Is(err, secretout.ErrTwoSinks) || *fake.opened != 0 {
			t.Fatalf("%v: err = %v, opened %d", args, err, *fake.opened)
		}
	}
}

func TestSecretsGoToTheFileAndNeverToOutput(t *testing.T) {
	want := map[string]string{
		"bot-tokens":    botToken,
		"app-tokens":    appToken,
		"config-tokens": `{"token":"xoxe.xoxp-1-distinctive","refresh_token":"xoxe-1-distinctive","exp":1800043200}`,
	}
	for _, args := range secretCommands {
		path := filepath.Join(t.TempDir(), "secret")
		out, errOut, err := runAll(t, tokenFake(), append(args, "--to-file", path)...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != want[args[0]] {
			t.Fatalf("%v: file holds %q", args, got)
		}
		assertNoToken(t, args, out, errOut)
	}
}

func TestSecretsGoToTheCommandsStdin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	for _, args := range secretCommands {
		got := filepath.Join(t.TempDir(), "got")
		out, errOut, err := runAll(t, tokenFake(), append(args, "--to-command", "cat > "+got+"; echo stored")...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if b, _ := os.ReadFile(got); len(b) == 0 {
			t.Fatalf("%v: the command read nothing", args)
		}
		if out != "stored\n" {
			t.Fatalf("%v: stdout %q, want the command's own output", args, out)
		}
		assertNoToken(t, args, out, errOut)
	}
}

func TestAFailingSinkLeaksNothing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses sh")
	}
	for _, args := range secretCommands {
		out, errOut, err := runAll(t, tokenFake(), append(args, "--to-command", "cat >/dev/null; exit 3")...)
		if err == nil {
			t.Fatalf("%v: a failing command succeeded", args)
		}
		assertNoToken(t, args, out, errOut, err.Error())
		existing := filepath.Join(t.TempDir(), "secret")
		if err := os.WriteFile(existing, []byte("old"), 0o600); err != nil {
			t.Fatal(err)
		}
		fake := tokenFake()
		out, errOut, err = runAll(t, fake, append(args, "--to-file", existing)...)
		if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
			t.Fatalf("%v: existing file: err = %v, opened %d", args, err, *fake.opened)
		}
		assertNoToken(t, args, out, errOut, err.Error())
	}
}

func TestAppTokensPassNameAndScopes(t *testing.T) {
	fake := tokenFake()
	_, err := run(t, fake, "app-tokens", "create", "A0EXAMPLE01", "--name", " socket-mode ",
		"--scope", "connections:write", "--scope", "authorizations:read", "--to-file", filepath.Join(t.TempDir(), "t"))
	if err != nil {
		t.Fatal(err)
	}
	if fake.tokenName != "socket-mode" || strings.Join(fake.scopes, ",") != "connections:write,authorizations:read" {
		t.Fatalf("name %q scopes %v", fake.tokenName, fake.scopes)
	}
}

func TestAppTokensValidateBeforeTheBrowserOpens(t *testing.T) {
	sink := []string{"--to-file", "unused"}
	for _, args := range [][]string{
		{"app-tokens", "create", "not-an-app", "--name", "n", "--scope", "connections:write"},
		{"app-tokens", "create", "A0EXAMPLE01", "--scope", "connections:write"},
		{"app-tokens", "create", "A0EXAMPLE01", "--name", "  ", "--scope", "connections:write"},
		{"app-tokens", "create", "A0EXAMPLE01", "--name", "n"},
		{"app-tokens", "create", "A0EXAMPLE01", "--name", "n", "--scope", "connections write"},
	} {
		fake := tokenFake()
		_, err := run(t, fake, append(args, sink...)...)
		if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
			t.Errorf("%v: err %v (code %d), opened %d", args, err, exitcode.Of(err), *fake.opened)
		}
	}
}

func TestAppIDIsValidated(t *testing.T) {
	fake := &fakeBrowser{}
	_, err := run(t, fake, "apps", "install", "not-an-app")
	if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
		t.Fatalf("err %v (code %d), opened %d", err, exitcode.Of(err), *fake.opened)
	}
}

func TestAppsList(t *testing.T) {
	fake := &fakeBrowser{apps: []slackweb.App{{Name: "Build Bot", ID: "A0EXAMPLE02", Workspace: "Example Co"}}}
	out, err := run(t, fake, "apps", "list")
	if err != nil {
		t.Fatal(err)
	}
	if want := "ID | NAME | WORKSPACE\nA0EXAMPLE02 | Build Bot | Example Co\n"; out != want {
		t.Fatalf("out %q", out)
	}
}

func TestInstallLeavesAnInstalledAppAlone(t *testing.T) {
	fake := &fakeBrowser{state: slackweb.InstallState{Installed: true}}
	out, err := run(t, fake, "apps", "install", "A0EXAMPLE01")
	if err != nil || out != "A0EXAMPLE01 is already installed\n" || fake.reinstall {
		t.Fatalf("out %q err %v reinstall %v", out, err, fake.reinstall)
	}
	fake = &fakeBrowser{state: slackweb.InstallState{Installed: true}, acted: true}
	out, err = run(t, fake, "apps", "install", "A0EXAMPLE01", "--reinstall")
	if err != nil || out != "Reinstalled A0EXAMPLE01\n" || !fake.reinstall {
		t.Fatalf("out %q err %v reinstall %v", out, err, fake.reinstall)
	}
}

func TestInstallPendingApprovalFails(t *testing.T) {
	fake := &fakeBrowser{state: slackweb.InstallState{PendingApproval: true}, acted: true}
	_, err := run(t, fake, "apps", "install", "A0EXAMPLE01")
	if !errors.Is(err, slackweb.ErrPendingApproval) {
		t.Fatalf("err %v", err)
	}
}

func TestMeExitsAuthWhenSignedOut(t *testing.T) {
	out, err := run(t, &fakeBrowser{}, "me")
	if code, _ := cmdutil.Classify(err); code != exitcode.Auth || !strings.Contains(out, "Signed in: no") {
		t.Fatalf("out %q err %v", out, err)
	}
}

func TestIconsRejectsANonPNG(t *testing.T) {
	fake := &fakeBrowser{}
	_, err := run(t, fake, "icons", "update", "A0EXAMPLE01", "--file", "root.go")
	if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
		t.Fatalf("err %v opened %d", err, *fake.opened)
	}
}

func TestLoginRejectsAWorkspaceThatIsNotASubdomain(t *testing.T) {
	for _, ws := range []string{"evil.example@acme", "acme/x", "acme#", "-acme"} {
		fake := &fakeBrowser{}
		_, err := run(t, fake, "login", "--workspace", ws)
		if exitcode.Of(err) != exitcode.Usage || *fake.opened != 0 {
			t.Errorf("%q: err %v opened %d", ws, err, *fake.opened)
		}
	}
}

func TestProfileDefaultsComeFromTheEnvironment(t *testing.T) {
	t.Setenv("SLAW_PROFILE", "work")
	t.Setenv("SLAW_PROFILE_DIR", "/tmp/example-profile")
	opts := cmdutil.Default()
	if opts.Profile != "work" || opts.ProfileDir != "/tmp/example-profile" {
		t.Fatalf("profile %q dir %q", opts.Profile, opts.ProfileDir)
	}
}
