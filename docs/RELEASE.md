# Release checklist (v2 candidate)

Do **not** tag, push, or publish until the owner reviews the diff and authorizes
those steps in writing.

**Candidate:** `v2.0.0` on commit to be recorded at tag time.
**Module path:** `github.com/WilsonSayago/initModules/v2` (Go major ≥ 2 suffix).
**Base (v1):** last published tag `v1.0.6` on `github.com/WilsonSayago/initModules`
(no `/v2` — still available for consumers that have not migrated).

## Before authorization

- [ ] `go.mod` declares `module github.com/WilsonSayago/initModules/v2`.
- [ ] Examples and docs import `/v2`.
- [ ] Working tree contains only attributed changes.
- [ ] `make ci` is green locally.
- [ ] GitHub Actions on the published branch is green (matrix logs show
      `go1.24.13` and `go1.27.1` with `GOTOOLCHAIN=local`).
- [ ] Changelog `[Unreleased]` matches the candidate; README install uses `/v2`.
- [ ] License file exists (owner choice) and README links it.
- [ ] `go doc ./...` matches the public API.

## Publication (owner-authorized only)

1. Commit any remaining release-commit edits (dated changelog heading
   `## [2.0.0] - YYYY-MM-DD`).
2. Annotated tag: `git tag -a v2.0.0 -m "initModules v2.0.0"`.
3. Push branch and tag (`git push` / `git push origin v2.0.0`).
4. Create the GitHub Release from that tag.
5. Verify from a temporary module **outside** this repo:

   ```sh
   mkdir /tmp/initmodules-consume && cd /tmp/initmodules-consume
   go mod init example.com/check
   GOWORK=off go get github.com/WilsonSayago/initModules/v2@v2.0.0
   GOWORK=off go list -m -json github.com/WilsonSayago/initModules/v2@v2.0.0
   ```

## Rollback / communication

- Do **not** move or recreate `v2.0.0`. A bad publish is a new patch
  (`v2.0.1`) plus a GitHub Release note describing the issue.
- Tell consumers to pin the last known-good tag (`v1.0.6` or the new v2 patch).
- If the module proxy cached a broken zip, wait for a new version; do not
  reuse the failed tag.
