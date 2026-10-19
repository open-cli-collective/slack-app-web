# slack-app-web

`slaw` does the Slack app steps that have no API, in a headless browser:

- generate or read an **app configuration token**, the credential Slack's
  `apps.manifest.*` API takes;
- **install or reinstall** an app to its workspace;
- read an installed app's **bot token**;
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

# Secrets go to a pipe or a file, never a terminal:
slaw config-tokens create | <command that stores the pair>
slaw config-tokens get > pair.json
slaw bot-tokens get A0123456789 | <command that stores the token>
```

`config-tokens create` and `get` write one JSON object with the field names
of Slack's `tooling.tokens.rotate`:

```json
{"token": "xoxe.xoxp-...", "refresh_token": "xoxe-...", "exp": 1700000000}
```

`exp` is unix seconds. The access token lasts 12 hours; rotate it with
`tooling.tokens.rotate` and the refresh token. Slack keeps one
configuration token per workspace: `create` refuses when one exists, and
`create --replace` deletes it first, which invalidates every copy of it.

## Keeping tokens out of logs

`bot-tokens get` and `config-tokens create|get` refuse to write to a
terminal, write the token with no trailing newline, and never log,
screenshot or save a page. Error messages are cut to one line and anything
shaped like a Slack token is redacted.

An agent or CI runner captures stdout through a pipe, so the terminal guard
does not apply there: always send the output to its destination in the same
command (`| store` or `> file`), never let it reach the runner's own output.

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
