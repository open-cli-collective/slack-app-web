# slack-app-web

`slaw` does the Slack app steps that have no API, in a headless browser:

- generate or read an **app configuration token**, the credential Slack's
  `apps.manifest.*` API takes;
- **install or reinstall** an app to its workspace;
- read an installed app's **bot token**;
- generate an **app-level token** (`xapp-...`), such as the one Socket Mode
  needs, and revoke one by name;
- upload an app's **icon**;
- **list** the apps an account manages.

Everything else about an app (its name, scopes, events, settings) is in its
manifest: manage that through Slack's `apps.manifest.*` API with the
configuration token this tool produces.

## Install

```bash
brew install --cask open-cli-collective/tap/slaw
slaw browser install   # downloads the Playwright driver and Chromium
slaw login             # sign in once in a browser window
```

On Linux, add the Open CLI Collective package repository
([instructions](https://github.com/open-cli-collective/linux-packages)), then
install the `slaw` package with apt, dnf, or pacman. From source: `go install github.com/open-cli-collective/slack-app-web/cmd/slaw@latest`.

## Usage

```bash
slaw me                                  # is the profile signed in? exits 3 if not
slaw apps list
slaw apps install A0123456789            # leaves an installed app alone
slaw apps install A0123456789 --reinstall
slaw icons update A0123456789 --file icon.png

# Secrets go only where you name them, never to slaw's output:
slaw config-tokens create --to-command '<command that reads the pair on stdin>'
slaw config-tokens get --to-file ./pair.json
slaw bot-tokens get A0123456789 --to-command '<command that reads the token on stdin>'
slaw app-tokens create A0123456789 --name socket-mode --scope connections:write \
  --to-command '<command that reads the token on stdin>'
slaw app-tokens revoke A0123456789 --name socket-mode
```

`app-tokens create` takes a token name and one `--scope` per scope (Socket
Mode needs `connections:write`). Slack shows an app-level token's value only
when it is generated, so `create` refuses a name the app already lists
instead of making a second token under it; revoke the old one with
`app-tokens revoke` or choose another name. If the token is generated but
cannot be read or delivered, the command fails with a hint to revoke it: the
token exists and its value is lost, so a rerun under the same name is refused
until it is revoked. A destination that failed keeps its usual exit code.
`app-tokens revoke` needs only the token's name, so it clears a token whose
value was lost; every copy of the token stops working at once.

`config-tokens create` and `get` deliver one JSON object with the field names
of Slack's `tooling.tokens.rotate`:

```json
{"token": "xoxe.xoxp-...", "refresh_token": "xoxe-...", "exp": 1700000000}
```

`exp` is unix seconds. The access token lasts 12 hours; rotate it with
`tooling.tokens.rotate` and the refresh token. Slack keeps one
configuration token per workspace: `create` refuses when one exists, and
`create --replace` deletes it first, which invalidates every copy of it.

## Keeping tokens out of logs

`bot-tokens get`, `app-tokens create` and `config-tokens create|get` never write a token to
slaw's own stdout or stderr, whether that is a terminal, an agent
transcript or a CI log. Each takes exactly one destination:

- `--to-file <path>` creates the file with mode 0600, writing it in full
  before it appears. An existing file is refused unless `--overwrite` is set.
- `--to-command <command>` runs the command through the shell with the token
  on its stdin. The command's own output passes through; a nonzero exit fails
  the run with its status, never the token.

With neither, the command fails before opening the browser. Tokens are
written with no trailing newline, and slaw never logs, screenshots or saves
a page. Error messages are cut to one line and anything shaped like a Slack
token is redacted.

Do not put a token on a command line inside `--to-command` (for example
`"$(cat)"` as an argument): process listings show arguments. Use a command
that reads stdin.

## Browser profiles

Each profile is a persistent Chromium profile holding one Slack sign-in,
under the user cache directory (`slack-app-web/profiles/<name>`). Use
`--profile <name>` (or `SLAW_PROFILE`) for several accounts, or
`--profile-dir <path>` (or `SLAW_PROFILE_DIR`) for a directory of your own.
Only one browser can use a profile at a time.

`login` is the only command that opens a window; every other command runs
headless and exits 3 with a hint to run `slaw login` when the session is gone.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | failure, including an install that awaits admin approval |
| 2 | usage error (bad app id, missing file, token already exists) |
| 3 | the profile is not signed in |
| 4 | app not found, or not manageable by this account |
| 5 | Slack did not respond as expected |

Set `SLAW_FULL_ERRORS=1` to see the browser's full call log on failure (still
redacted).
