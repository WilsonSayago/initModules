# Release checklist (v1 candidate)

Do **not** tag, push, or publish until the owner reviews the diff and authorizes
those steps in writing.

**Candidate:** `v1.6.0` on commit to be recorded at tag time.
**Base:** last published tag `v1.0.6`.
**API check:** `golang.org/x/exp/cmd/apidiff@v0.0.0-20260908205506-85c1c2202aba`
against `v1.0.6` reported compatible additions only (2026-09-16).

## Before authorization

- [ ] Plans 001–006 are `DONE`.
- [ ] Working tree contains only attributed changes (no stray owner-local files).
- [ ] `make ci` is green locally.
- [ ] GitHub Actions on the published branch is green (matrix logs show
      `go1.24.13` and `go1.27.1` with `GOTOOLCHAIN=local`).
- [ ] Changelog `[Unreleased]` matches the candidate; README install line will
      be switched from `@v1.0.6` to `@v1.6.0` in the release commit.
- [ ] License file exists (owner choice) and README links it.
- [ ] `go doc ./...` matches the public API.

## Publication (owner-authorized only)

1. Commit any remaining release-commit edits (version pin in README, dated
   changelog heading `## [1.6.0] - YYYY-MM-DD`).
2. Annotated tag: `git tag -a v1.6.0 -m "initModules v1.6.0"`.
3. Push branch and tag (`git push` / `git push origin v1.6.0`).
4. Create the GitHub Release from that tag.
5. Verify from a temporary module **outside** this repo:

   ```sh
   mkdir /tmp/initmodules-consume && cd /tmp/initmodules-consume
   go mod init example.com/check
   GOWORK=off go get github.com/WilsonSayago/initModules@v1.6.0
   GOWORK=off go list -m -json github.com/WilsonSayago/initModules@v1.6.0
   ```

## Rollback / communication

- Do **not** move or recreate `v1.6.0`. A bad publish is a new patch
  (`v1.6.1`) plus a GitHub Release note describing the issue.
- Tell consumers to pin the last known-good tag (`v1.0.6` or the new patch).
- If the module proxy cached a broken zip, wait for a new version; do not
  reuse the failed tag.
