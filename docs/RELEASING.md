# Releasing MailBust

Releases are cut by GoReleaser from a version tag, via the `release` GitHub Action.

## One-time setup

1. **Homebrew tap & Scoop bucket repos** (only needed for `brew`/`scoop` installs):
   - Create `atifrana/homebrew-tap` (public, empty).
   - Create `atifrana/scoop-bucket` (public, empty).
2. **PAT secret**: create a GitHub Personal Access Token with `contents:write` on
   those two repos, and add it to `atifrana/mailbust` as the repository secret
   **`TAP_GITHUB_TOKEN`**.
   - If you skip this, comment out the `brews:` and `scoops:` blocks in
     `.goreleaser.yaml` for now - the binaries and GitHub Release still publish.

## Cut a release

```bash
# make sure main is green, then:
git tag v0.1.0
git push origin v0.1.0
```

The `release` workflow builds all platforms, publishes a GitHub Release with
archives + `checksums.txt`, and (if configured) updates the tap and bucket.

## Record the demo GIF

```bash
go install github.com/charmbracelet/vhs@latest
# ensure `mailbust` is on PATH (go install ./cli, or use a built binary)
vhs docs/demo.tape      # writes docs/demo.gif
```

## Dry run / validate locally

```bash
goreleaser check                 # validate .goreleaser.yaml
goreleaser release --snapshot --clean   # build everything without publishing
```
