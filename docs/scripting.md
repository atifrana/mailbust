# Scripting & CI with MailBust

MailBust's scripting mode prints exactly what you ask for on `stdout` (everything
else goes to `stderr`), so it drops straight into shell pipelines and CI.

## Commands

```bash
mailbust --new                      # create a random mailbox, print its address
mailbust --new --json               # {address, token, expires_at}
mailbust testing --code             # wait for the next code, print it, exit
mailbust testing --link             # wait for the next verify link, print it, exit
mailbust testing --code --json      # full JSON result (code, from, subject, …)
mailbust testing --code --timeout 60    # wait up to 60s (default 120)
mailbust testing --list             # current messages (tab-separated)
mailbust testing --list --json      # current messages as JSON
```

## Exit codes

| Code | Meaning |
| ---- | ------- |
| 0 | success |
| 1 | error (network, server, etc.) |
| 2 | timeout - no matching message arrived in time |
| 3 | invalid input (bad flags/arguments) |

Scripts can branch on these directly:

```bash
code=$(mailbust testing --code --timeout 90)
case $? in
  0) echo "got code: $code" ;;
  2) echo "no code arrived in time" >&2; exit 1 ;;
  *) echo "mailbust failed" >&2; exit 1 ;;
esac
```

## Environment variables

| Variable | Purpose |
| -------- | ------- |
| `MAILBUST_API` | Backend base URL (default `https://api.mailbust.com`) |
| `MAILBUST_DOMAIN` | Mail domain for bare names (default `mailbust.com`) |
| `MAILBUST_TOKEN` | Bearer token for a private mailbox (handy in CI, which has no OS keychain) |

## Two ways to use it in CI

**Named public mailbox (simplest).** Named mailboxes are public, so anyone who
knows the name can read them - fine for throwaway CI codes if you pick an
unguessable name (e.g. include the run id):

```bash
addr="ci-$GITHUB_RUN_ID@mailbust.com"
./signup.sh "$addr"
code=$(mailbust "$addr" --code --timeout 120)
```

**Private mailbox (no public reads).** Create one, carry the token via
`MAILBUST_TOKEN`:

```bash
json=$(mailbust --new --json)
addr=$(echo "$json" | jq -r .address)
export MAILBUST_TOKEN=$(echo "$json" | jq -r .token)
./signup.sh "$addr"
code=$(mailbust "$addr" --code --timeout 120)
```

A ready-to-adapt GitHub Actions workflow is in
[`ci-github-actions.yml`](ci-github-actions.yml).
