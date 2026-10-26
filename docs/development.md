# slack-app-web development notes

Repo-local facts for `slack-app-web` (binary `slaw`). Shared standards:
https://github.com/open-cli-collective/cli-common/tree/main/docs

## Quick commands

```bash
make build              # ./bin/slaw
make test               # go test -race ./...
make lint               # golangci-lint
make smoke              # build, then run --help on every command group
make check              # tidy, fmt, lint, test, build
SLAW_PROFILE_DIR=<dir> make test-integration   # read-only browser flows against a signed-in profile
git config core.hooksPath scripts/hooks          # reject AI-tool mentions in commit messages
```

## Layout

- `cmd/slaw/`: entry point.
- `internal/cmd/`: one package per command group; `cmdutil` holds the global
  options and the `Browser` interface commands run against, which tests
  replace with a fake.
- `internal/slackweb/`: the browser flows (Playwright through playwright-go)
  and the pure parsing they rely on (`parse.go`).
- `internal/secretout/`: the only path a secret takes out of slaw (a 0600
  file or a command's stdin, never slaw's own output), and the error
  redaction.
- `internal/output/`: pipe-delimited tables.
- `internal/exitcode/`: errors that carry the exit code.

## Rules for changes

- **Browser only for what has no API.** Manifest changes, scopes, events and
  settings belong to Slack's `apps.manifest.*` API, used through other tools;
  this CLI adds a command only for a step Slack offers in the browser alone.
- **Secrets never reach a log.** A token leaves only through a
  `secretout.Sink` the caller named (`--to-file` or `--to-command`), never
  through stdout or stderr, because a terminal check cannot tell an agent
  transcript or a CI log from a pipe. Do not print page text, take screenshots, or wrap a
  browser error without `secretout.ErrorText` in the path to stderr. Token
  pages (OAuth & Permissions, install, Your Apps' token table, the app-level
  token dialog) are never captured in any form. A new token prefix needs a
  matching pattern in `secretout/scrub.go`.
- **Read state from the page, then act.** Wait for an element that proves the
  page rendered (the token label or the install button) before deciding what
  to click; the settings pages render after `domcontentloaded`.
- **Fixtures are invented.** Tests use example app ids (`A0EXAMPLE01`) and
  workspace names (`Example Co`).

## Slack page facts the flows depend on

- Your Apps (`api.slack.com/apps`) has a **Generate Token** button when signed
  in, which is the signed-in check. It is disabled while the workspace
  already has a configuration token; Slack keeps one per workspace.
- Configuration and bot tokens are not in the page text. The Copy buttons
  write them to the clipboard; an init script captures those writes in the
  page instead of the system clipboard.
- An app's Basic Information page redirects to `app.slack.com/app-settings/`
  and saves a new icon as soon as the file input receives it; the flow
  confirms by the avatar URL changing.
- The install page shows "Bot User OAuth Token" once installed and an
  "Install to <workspace>" or "Reinstall to <workspace>" button.

- Basic Information has an **App-Level Tokens** section listing each token
  by name with its scopes, and a **Generate Token and Scopes** button that
  opens a dialog with a token name field, an **Add Scope** picker and
  **Generate**. The token (`xapp-...`) then shows in a dialog closed with
  **Done**. `app-tokens create` refuses a name the section already lists.
  These selectors were written without a live page to check against; the
  fixture `testdata/app-level-tokens-section.txt` is invented.

## Release

`version.txt` holds MAJOR.MINOR; `auto-release.yml` tags
`v<MAJOR.MINOR>.<run number>` on a qualifying merge and `release.yml` publishes
through the shared workflow (cli-common `release.md`, `distribution.md`).
