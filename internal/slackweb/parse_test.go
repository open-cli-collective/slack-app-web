package slackweb

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestParseAppTable(t *testing.T) {
	var rows [][]string
	raw, err := os.ReadFile("../../testdata/apps-table.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		t.Fatal(err)
	}
	want := []App{
		{Name: "Deploy Notices", ID: "A0EXAMPLE01", Workspace: "Example Co"},
		{Name: "Build Bot", ID: "A0EXAMPLE02", Workspace: "Example Co"},
	}
	if got := parseAppTable(rows); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseAppTable = %+v, want %+v", got, want)
	}
}

func TestParseAppTableWithoutAnIDColumn(t *testing.T) {
	if got := parseAppTable([][]string{{"Name"}, {"x"}}); got != nil {
		t.Fatalf("got %+v", got)
	}
}

func TestParseExpiry(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	cases := []struct {
		text string
		want time.Duration
	}{
		{"Example Co\nCopy\nExpires in 12 hours\nCopy", 12 * time.Hour},
		{"Expires in 1 hour", time.Hour},
		{"Expires in 45 minutes", 45 * time.Minute},
		{"no expiry shown", configTokenLifetime},
	}
	for _, c := range cases {
		if got := parseExpiry(c.text, now); got != now.Add(c.want).Unix() {
			t.Errorf("parseExpiry(%q) = %d, want %d", c.text, got, now.Add(c.want).Unix())
		}
	}
}

func TestConfigTokenContract(t *testing.T) {
	raw, err := json.Marshal(ConfigToken{Token: "a", RefreshToken: "r", Exp: 42})
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../testdata/config-token.json")
	if err != nil {
		t.Fatal(err)
	}
	var got, expected map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &expected); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("contract = %s, want %s", raw, want)
	}
}

func TestTokenPatterns(t *testing.T) {
	values := []string{"", "Bot User OAuth Token", "xoxb-1-2-abc"}
	if got := firstMatch(botTokenPattern, values); got != "xoxb-1-2-abc" {
		t.Errorf("bot token = %q", got)
	}
	copied := []string{"xoxe-1-My0tRefresh", "xoxe.xoxp-1-Mi0Access"}
	if got := firstMatch(configAccessPattern, copied); got != "xoxe.xoxp-1-Mi0Access" {
		t.Errorf("access = %q", got)
	}
	if got := firstMatch(configRefreshPattern, copied); got != "xoxe-1-My0tRefresh" {
		t.Errorf("refresh = %q", got)
	}
}

func TestTokenNameListed(t *testing.T) {
	raw, err := os.ReadFile("../../testdata/app-level-tokens-section.txt")
	if err != nil {
		t.Fatal(err)
	}
	section := string(raw)
	cases := map[string]bool{
		"socket-mode":       true,
		"  socket-mode ":    true,
		"Deploy  Listener":  true,
		"socket":            false,
		"Socket-Mode":       true,
		"connections:write": true,
		"new-token":         false,
	}
	for name, want := range cases {
		if got := tokenNameListed(section, name); got != want {
			t.Errorf("tokenNameListed(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestAppTokenPattern(t *testing.T) {
	text := "Token\nxapp-1-A0EXAMPLE01-1234567890123-0123abcdef\nCopy"
	if got := firstMatch(appTokenPattern, []string{"Generate", text}); got != "xapp-1-A0EXAMPLE01-1234567890123-0123abcdef" {
		t.Errorf("app token = %q", got)
	}
}

func TestValidAppID(t *testing.T) {
	for id, want := range map[string]bool{"A0EXAMPLE01": true, "a0example": false, "C0123456": false, "": false} {
		if got := ValidAppID(id); got != want {
			t.Errorf("ValidAppID(%q) = %v", id, got)
		}
	}
}
