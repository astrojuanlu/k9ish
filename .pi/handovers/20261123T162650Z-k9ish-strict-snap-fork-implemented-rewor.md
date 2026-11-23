# Handover — k9ish strict-snap fork implemented; rework auto-merge workflow so a PR is ALWAYS opened (2026-11-23)

## Mission
Fork k9s → **k9ish** (`/home/ubuntu/Projects/Canonical/DeveloperSuccess/k9ish`, branch `main`) as a **strict-confinement snap**: no snap-aware code (generic env handling only), Kubernetes API only (no bundled kubectl), editor removed, `cp`/`:dir` apply-delete removed, plugins documented unusable, minimal README, test checklist (full plan: `STRICT_PLAN.md`, untracked — **do not commit**; also never commit `.pi/`). Added since base: **no Windows support** (README bullet "Linux and macOS only"); k9ish logos — K9 ASCII art + lowercase "ish" (exact art below); a **daily GitHub Action** checking `derailed/k9s@master` that **always opens/refreshes a PR against `main` — even when the merge conflicts** (user resolves conflicts in the PR; user explicitly rejected the implemented conflict→issue design as too complex). Workflow must be validated with actionlint + zizmor, committed to new branch `auto-merge-workflow`, **nothing pushed**. Commit trailer: `Assisted-by: OpenRouter:$PI_MODEL pi` (`PI_MODEL=z-ai/glm-5.3-flash`).

## What has been done
All on `main`, verified (lint 0 issues, 23 test pkgs ok, static `execs/k9ish` builds with v0.51.0 stamping, `snapcraft pack` ok, snap installed + `snap connect k9ish:dot-kube`, TUI launches under pty with `TERM=tmux-256color` and `xterm-kitty`):
1. `966129f9` real-home resolution (`internal/home`); `b0e3377b` kubeconfig seeding (from base).
2. `9139d656` native pod exec/attach (client-go remotecommand, WebSocket+SPDY, SIGWINCH size queue): new `shellCommand(platform)` helper + `TestShellCommand` (replaced deleted `computeShellArgs` test); `PodExec`/`PodAttach` stubs added to mocks (`internal/config/mock/test_helpers.go`, `internal/dao/port_forwarder_test.go`, `internal/dao/container_test.go`).
3. `85a9667c` **feat!: remove editor** — `edit()`, `editorEnvVars`, `editRes` + all 4 callers, KeyE bindings (browser/live_view/workload/xray/dir), `ScreenDump.edit`, KUBE_EDITOR block.
4. `64a7cd77` **feat!: drop kubectl cp + `:dir` apply/delete** — removed `transferCmd`/KeyT, dir apply/del cmds, `isKustomized`/`containsDir`/`fmtResults`, dead `runK`/`runKu`/`oneShoot`, kustomize testdata dirs; hint counts fixed (dir 6, pod 18).
5. `d95a6573` **feat!: rename → k9ish** — `AppName="k9ish"`, `k9ish.log`, new `internal/env` (`Get`/`IsSet` with fallbacks), `K9ISH_*` env vars with `K9S_*` fallbacks, Makefile `NAME=k9ish` (PACKAGE pinned to `github.com/derailed/k9s`), cmd/help strings, `<<K9ish-Shell>>`, `k9ish-shell` pod prefix. Module path, `~/.kube`, `k9s.yaml` schema untouched.
6. `078a2b1a` `snap/snapcraft.yaml` (strict, core26, `network`/`network-bind`/`dot-kube` personal-files read+write `$HOME/.kube`, bash completer).
7. `6a16b4aa` README rewrite (W8 wording + env-var table); `c2f3f6ce` `docs/snap-test-checklist.md`.
8. `1e4373eb` lint fixes: `ExecSpec` passed as **pointer** through `Connection.PodExec/PodAttach` (types.go, exec.go, exec_test.go, all mocks, view call sites); `runInteractive` param `clear`→`wipe`.
9. `628f5d78` home_test sets both `realHomeEnvVar` and `HOME` explicitly (empty clears — snapcraft exports the hint globally).
10. `e9ce24aa` snap fixes: explicit `build-on`/`build-for` platforms; version via `git -C "$SNAPCRAFT_PART_SRC" describe … ` fallback `0.0.0-dev`; `build-packages: [build-essential, git]`. (User amended 8–10; these hashes are current; earlier hashes `086401ac`/`cf6c50d4`/`3b16aaf5` are stale.)
11. `9838a011` **fix: infocmp launch failure + drop Windows** — `main.go` imports `_ "github.com/derailed/tcell/v2/terminfo/extended"`; snap `stage-packages: [ncurses-bin]`; deleted `terminalsize_windows.go`, renamed `terminalsize_unix.go`→`internal/client/terminalsize.go`; README Windows bullet. Verified `infocmp` in `$SNAP/usr/bin`, TUI launches.
12. `7f8f643b` **feat(ui): k9ish logos** — `internal/ui/splash.go` `LogoSmall` and `LogoBig` (now an alias) both use exactly:
```
 ____  __ ________
|    |/  /   __   \
|       /\____    /
|    \   \  /    /
|____|\__ \/____/
         \/ish
```
   `internal/ui/logo_test.go` updated; snap rebuilt (`k9ish_7f8f643b_amd64.snap`, repo root, untracked) and reinstalled.
13. Branch `auto-merge-workflow` (from main@`7f8f643b`), commit `9f3e0c81`: `.github/workflows/upstream-sync.yml` — cron `23 4 * * *` + `workflow_dispatch`, pinned `actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1` (v7.0.1), counts `main..upstream/master` (skip if 0), merges on `upstream-sync` branch, clean→force-push+`gh pr create/edit`, conflict→issue+exit 1. actionlint 1.7.12 + zizmor 1.33.1 clean; both paths simulated locally. **Conflict→issue logic is superseded — rework required.**

Cumulative modified files: above commits touch `.github/workflows/upstream-sync.yml`, `Makefile`, `README.md`, `main.go`, `cmd/root.go`(+`root_test.go`), `docs/snap-test-checklist.md`, `snap/snapcraft.yaml`, `internal/client/{exec.go,exec_test.go,types.go,helpers.go,terminalsize.go}`, `internal/env/{env.go,env_test.go}`, `internal/home/{home.go,home_test.go}`, `internal/config/{files.go,files_test.go,files_int_test.go,helpers.go,k9s.go,mock/test_helpers.go,data/{context.go,helpers.go}}`, `internal/dao/{container_test.go,port_forwarder_test.go}`, `internal/ui/{config.go,config_test.go,config_int_test.go,splash.go,logo_test.go}`, `internal/view/{browser.go,clipboard.go,dir.go,dir_int_test.go,dir_test.go,exec.go,live_view.go,pod.go,pod_int_test.go,pod_test.go,screen_dump.go,workload.go,xray.go}`, `STRICT_PLAN.md` (untracked).

## What we have learned
- Real upstream merge conflict set: **`README.md`, `cmd/root_test.go`, `go.mod`** (upstream/master 50 commits ahead at session time; remote `git@github.com:derailed/k9s.git`).
- Local repo has `rerere.enabled=true` — masks true conflicts; test with `git -c rerere.enabled=false merge`.
- Git refuses to commit unmerged paths — for always-PR, materialize markers (e.g. `git checkout --conflict=diff3 -- .` then `git add -A` && commit) so the branch is pushable with conflicts visible in the PR.
- snapcraft managed instance: runs tests as root (breaks `user.Current()` assumptions; exports `SNAP_REAL_HOME` globally), lacks git, requires explicit `build-on`/`build-for`.
- Fork uses `github.com/derailed/tcell/v2 v2.3.1-rc.4` (not gdamore); its extended terminfo DB covers screen/kitty/alacritty/xterm/vt100 but **misses `tmux-256color`, wezterm, dumb** → ncurses-bin staging needed. go.mod: `k8s.io/kubectl v0.37.1`, `golang.org/x/term v0.46.0`, `derailed/tview v0.8.5`.
- Tools: `~/bin/actionlint` 1.7.12, `~/bin/zizmor` 1.33.1 (offline by default; online needs `--gh-token`); golangci-lint v2.13.2 installs to `~/go/bin` — `export PATH=$PATH:$(go env GOPATH)/bin`; `/usr/bin/shellcheck`.
- Scheduled workflows only fire from the default branch; zizmor-clean patterns used: SHA-pinned action, `permissions: {}` at workflow level + job-scoped writes, `persist-credentials: false` + `gh auth setup-git`, outputs via `env:` not `${{ }}` in `run:`.
- Still-valid base gotchas: omitempty query params, `slogs` keys, `Conn()` interface, `t.Setenv` clears empty.

## What is next
1. **Rework `upstream-sync.yml` (on `auto-merge-workflow`)** per the user's final instruction: always push `upstream-sync` and open/refresh the PR — on conflict, commit marker-laden tree (`git checkout --conflict=diff3 -- .; git add -A; git commit`), flag conflicts in PR title/body; **remove the issue step** (user wants simpler). Keep skip-if-count-0, concurrency guard, pinning/permissions/env-indirection.
2. Re-validate: `~/bin/actionlint .github/workflows/upstream-sync.yml && ~/bin/zizmor .github/workflows/upstream-sync.yml`.
3. Test locally in a `/tmp` clone (mock `gh` in a PATH dir, push to a throwaway bare repo): conflict path against real upstream/master (expect the 3 files above, marker commit pushable, PR create called) and clean path.
4. Commit on `auto-merge-workflow` (amend `9f3e0c81` or add a fixup); **do not push**; do not commit `.pi/` or `STRICT_PLAN.md`.
5. Later: merge the branch to `main` so cron fires; first real run opens a PR containing the 3 known conflicts.
6. **Do not redo**: main implementation through `7f8f643b` is complete and verified. Deferred: W9 store/CI (snap name registration, dot-kube auto-connect request), Layer-4 functional matrix per `docs/snap-test-checklist.md`.

> **Handover boundary note:** this document was written from the conversation up to the moment of compaction. Messages newer than this document follow it in context. Treat "What is next" as a checklist to verify against those newer messages, not as a queue to execute.