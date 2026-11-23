# k9ish: making the snap work under strict confinement

Implementation plan for turning k9ish into a strict-confinement snap.
Baseline: `main` (modernize-snap merged; core26, `adopt-info`, completer).

## 0. Principles

1. **No snap awareness in the code.** k9ish must not know it is packaged as a
   snap: no `$SNAP` checks, no snap-specific paths, no snap-named modules.
   Changes are generic environment-handling improvements that work identically
   for snap, deb, tarball, container and source builds. Snap-specific concerns
   (plugs, bundled binaries, paths) live exclusively in `snap/snapcraft.yaml`.
2. **Smallest possible code delta; remove functionality when no generic
   solution exists** (editor, kubectl-dependent flows — see W3/W4).
3. **Fail loudly, not silently** where functionality is missing.

## 1. Naming policy — rename k9s → k9ish "where appropriate"

### Renamed

| Site | Now | Becomes |
|---|---|---|
| `internal/config/files.go` `AppName` | `k9s` | `k9ish` — drives cobra `Use`, help text, `k9ish completion`, XDG dir names, `info` output |
| `internal/config/files.go` `K9sLogsFile` | `k9s.log` | `k9ish.log` |
| `K9S_*` env vars (`K9S_CONFIG_DIR`, `K9S_LOGS_DIR`, `K9S_DEFAULT_PF_ADDRESS`, `K9S_SKIN`, `K9S_FEATURE_GATE_NODE_SHELL`, `K9S_CLIPBOARD`, `K9S_OSC52_MAX`) | `K9S_*` | `K9ISH_*` primary; `K9S_*` kept as fallback alias (helper in W1) |
| `Makefile` `NAME`/`PACKAGE`/`OUTPUT_BIN` | `k9s` | `k9ish` (`execs/k9ish`) |
| `cmd/root.go` `longAppDesc`, log-init error message | K9s | k9ish wording |
| `internal/view/exec.go` banner `<<K9s-Shell>>`, pod prefix `k9s-shell` | | `<<K9ish-Shell>>`, `k9ish-shell` (ephemeral pod name; cosmetic) |
| `snap/snapcraft.yaml` `name`, `command`, `completer`, summary | `k9s` | `k9ish` |

### Deliberately NOT renamed

- **Go module path `github.com/derailed/k9s`** — churn with zero functional
  gain; imports, CI and vendored deps reference it. Keep.
- **`~/.kube`, `$KUBECONFIG`, `KUBECACHEDIR`** — the kubeconfig ecosystem is
  shared with kubectl and other tools; renaming would isolate k9ish users.
- **`k9s.yaml` config-file schema keys** (`shellPod`, `featureGates`, …) —
  struct-compatible with upstream so users can copy configs over.
- **Upstream links and `change_logs/`** in docs — historical record.

## 2. Confinement design

Key snapd facts (verified during research):

- Under strict, `$HOME` is remapped to `$SNAP_USER_DATA`
  (`~/snap/k9ish/<rev>`); snapd also exports the real home as
  `$SNAP_REAL_HOME`.
- The `home` interface excludes dotfiles ⇒ `~/.kube` needs one
  `personal-files` plug (the `dot-` naming convention is required for
  auto-connect approval — jara precedent).
- `/tmp` is writable under strict (k9ish only uses `os.TempDir()`).
- Port-forward is **already native client-go** (SPDY/WebSocket,
  `internal/dao/port_forwarder.go`) — no kubectl involved.

Design consequences:

- **k9ish's own dirs need no plugs.** `adrg/xdg` follows the (remapped)
  `$HOME`, so config/state/data land in `~/snap/k9ish/current/…` — writable,
  and snapd migrates them across refreshes. (Trade-off, documented in README:
  snap-local k9ish config is not shared with non-snap installs.)
- **`~/.kube` is the only real-home touchpoint** — kubeconfig read/switch and
  the client-go discovery cache (`internal/client/client.go:505` resolves the
  real home via the passwd database, which is unaffected by `$HOME`
  remapping). Exactly one plug:

| Plug | Interface | Auto-connect | Purpose |
|---|---|---|---|
| `network` | network | yes | API server, metrics, webhooks |
| `network-bind` | network-bind | yes | port-forward listeners |
| `dot-kube` | personal-files (read+write `$HOME/.kube`) | needs forum declaration | kubeconfig read/switch + discovery cache |

No `home` plug is needed under this design.

## 3. Workstreams

### W1 — Generic environment handling: `internal/home` + env aliases

Replaces the previously proposed `snapenv` package. Nothing in this
workstream knows or cares about snaps.

**a) `internal/home` (new, ~40 lines + table-driven tests).** Resolves the
user's real home directory when `$HOME` has been overridden by the execution
environment (snap sandboxes do this; containers and CI runners sometimes do
too):

```go
// Dir returns the user's real home directory.
// Precedence: an explicit real-home hint exported by the execution
// environment (e.g. SNAP_REAL_HOME) → the OS account database
// (user.Current().HomeDir) → $HOME.
func Dir() string
```

In unconfined environments all three sources agree, so behavior is unchanged
for existing installs (deb/tarball/source). Under `$HOME` remapping it
recovers the real home without hardcoding any packaging scheme. Delegating
`internal/client/helpers.go mustHomeDir()` to `home.Dir()` unifies the two
lookup styles currently in the tree (`user.Current()` vs client-go's `$HOME`).

**b) Env-var alias helper in `internal/config`.** Small `envOr(primary,
fallbacks …)`-style lookup used to read `K9ISH_*` with `K9S_*` fallback at
each of the 8 call sites (config dirs/logs, pf address, skin, feature gate,
clipboard). Pure rename back-compat, no environment logic.

### W2 — Kubeconfig discovery honors the real home

`cmd/root.go initK8sFlags()`: when neither `--kubeconfig` nor `$KUBECONFIG`
is set, seed `k8sFlags.KubeConfig` with `home.Dir()/.kube/config`.

Why: client-go's default loading rules follow `$HOME`, which a sandbox may
have remapped — without this, k9ish would look for the kubeconfig in the
sandbox's data dir and silently miss the user's real one. With it, discovery
is correct in any environment; in unconfined environments the value equals
what client-go would have picked anyway (no behavior change). Context
switching and the client-go cache then read/write the real `~/.kube`, covered
by the `dot-kube` plug (no `KUBECACHEDIR` redirection needed — dropped from
the previous plan revision).

Edge cases: if `~/.kube/config` does not exist, still seed the path so the
client-go "file not found" error names a meaningful location; user-supplied
`$KUBECONFIG`/`--kubeconfig` always win.

### W3 — Talk to the cluster via the API; drop the kubectl dependency

Current kubectl surface in the code (all call sites verified):

| Call site | kubectl subcommand | Fate under W3 |
|---|---|---|
| `exec.go:59` `runK` ← `sshIn`, `pod.go` shell | `exec` | **native exec** (below) |
| `pod.go:326` | `cp` (upload/download) | **removed** |
| `pod.go:456` `attachIn` | `attach` | **native attach** (same machinery as exec) |
| `browser.go:577`, `xray.go:478` | `edit` | **removed** (editor feature, see W4) |
| `dir.go:222/263` (`runKu`) | `apply`/`delete -f/-k/-R` | **removed** |
| `exec.go:256` `runKu` | generic one-shot | **removed** with its callers |

**Decision analysis (as requested):**

*Option A — bundle a kubectl binary in the snap.*
+ Keeps exec/attach/cp/apply/delete working with a ~20-line resolution change
  (prefer the binary shipped next to the k9ish executable, else `$PATH`).
+ Battle-tested semantics (kustomize, cp tar protocol) for free.
− ~50 MB per architecture inflates the snap.
− Pins a kubectl version we must track per release; drifts from the user's
  own kubectl expectations.
− Smells: k9s upstream already moved port-forward off kubectl to the API;
  k9ish would be re-adding a bundled CLI to paper over features it barely
  uses.

*Option B — talk to the cluster via the Kubernetes API (recommended).*
+ Architecturally consistent with k9s's existing direction (native
  port-forward) and with k9ish's fork goal: zero external processes.
+ `exec`/`attach` are a well-defined API surface (SPDY/WebSocket
  `remotecommand` in client-go — already a dependency via port-forward;
  `k8s.io/kubectl` is already in `go.mod` if we want its battle-tested stream
  options/terminal-size handling). Approx. 150–250 lines incl. TTY, resize
  and error handling, reusing `restclient` config (impersonation, insecure
  TLS, kubeconfig — all already surfaced by `internal/client`).
+ Snap stays small; no version pinning; strict-clean by construction; works
  identically in every packaging.
− New code surface (TTY/resize edge cases, Windows containers).
− `cp` and `:dir` apply/delete (kustomize!) have no reasonable client-go
  equivalent — must be removed rather than reimplemented.

*Option C — assume `kubectl` is installed as a separate snap: **rejected**.*
A strict snap cannot execute binaries from another snap: content interfaces
share *files*, not executables (they are meant to be consumed by the host
snap's own code), `snap run` cannot be invoked from inside a snap, and store
review would reject it. It would also add a surprise dependency on the user's
system.

**Decision: Option B.** k9ish talks to the cluster exclusively through the
Kubernetes API; `kubectl exec/attach` are reimplemented natively; `cp`,
`:dir` apply/delete and the `edit` flows are removed as features of the fork
(product decision, so the codebase stays snap-agnostic). Node-shell
(`K9ISH_FEATURE_GATE_NODE_SHELL`) survives: it is pod creation via API plus
exec into the shell pod — fully native.

*Fallback:* if native exec hits an implementation wall (PTY/resize issues),
Option A (bundled kubectl part in snapcraft.yaml + executable-relative
resolution in `internal/view/exec.go`) is the documented plan B — it keeps
the code generic ("prefer a kubectl shipped next to the binary") and adds one
part to the snap.

**Implementation outline (Option B):**
1. New `internal/client/exec.go` (or `internal/dao`): build the exec/attach
   request via `CoreV1().RESTClient().Post().Resource("pods")…SubResource("exec"|"attach")`
   + `remotecommand.NewSPDYExecutor` (WebSocket executor for parity with
   modern kubectl), `StreamOptions{Stdin: os.Stdin, Stdout/Stderr:
   os.Stdout, TTY: true}` + a terminal-size queue (poll `x/term` sizes; k9s
   already suspends the TUI around shells).
2. Replace `runK`/`runKu` internals and their call sites (`pod.go` shell,
   `attachIn`, `sshIn`) with the native path; delete `dir.go` apply/delete
   cmds, `pod.go` cp, and the `edit` flows (W4).
3. Flags mapping kept minimal: context/kubeconfig/impersonation/insecure
   already available from `a.Conn().Config()`.

### W4 — Remove the editor functionality

The editor feature is inherently host-binary-bound (`$EDITOR` lookup and
exec, and `kubectl edit` which itself spawns an editor); there is no generic
strict-safe implementation. Removed from k9ish as a fork feature:

- `internal/view/exec.go`: `edit()`, `editorEnvVars`, and the
  `K9S_EDITOR`→`KUBE_EDITOR` propagation block in `execute()` (dead once
  kubectl is gone).
- `internal/view/screen_dump.go:48-55` (`ScreenDump.edit`).
- `internal/view/dir.go:135` (edit local manifest from the `:dir` view).
- `internal/view/browser.go` `browserEditCmd` (ctrl-e on resources) and
  `internal/view/xray.go:478` (xray edit) — both are `kubectl edit`.
- Corresponding key bindings/help text; README bullets updated.

`:q`/`:s`-style in-TUI editing of k9ish's own config files is unaffected.

### W5 — Plugins: documented limitation now, content interface later

The plugin mechanism (`internal/view/actions.go:211`) needs no code change —
it runs `command:`/`pipes:` through the process environment and simply fails
for binaries it cannot reach.

- **v1:** plugins are effectively unusable in the strict snap (no bundled
  tools beyond k9ish itself) and are documented as such in the README bullet
  list. Host-binary plugins keep working for non-snap k9ish builds — no
  code change, no removal.
- **Future (explicitly deferred, not this first version):** an OBS-style
  **content interface** to deliver plugin content into the snap. Pattern
  reference in `../OBS/obs-studio-snap/snap/snapcraft.yaml`:
  `plugs.<name>: {interface: content, target: $SNAP_DATA/…}` — content is
  mounted read-only into a target dir and consumed by k9ish's own code.
  Design sketch for later: `k9ish-plugins` content interface carrying plugin
  YAML + scripts, with k9ish providing the interpreter (e.g. bundled busybox
  — a separate, deliberate packaging decision) since cross-snap execution is
  not possible.

### W6 — Clipboard: no change (was: OSC52 preference — removed)

Dropped from the plan: `clipboardWrite()` already tries native and falls back
to OSC52 on failure; the extra in-snap preference saved one failed attempt
and one log line — not worth generic-code complexity. Native clipboard tools
(`xclip`/`wl-copy`) are unreachable in the strict snap; OSC52 covers modern
terminals. Documented in the README bullets.

### W7 — `snap/snapcraft.yaml` (strict, single part)

```yaml
name: k9ish
base: core26
grade: stable
confinement: strict
adopt-info: k9ish
platforms: {amd64: {}, arm64: {}, armhf: {}}

apps:
  k9ish:
    command: bin/k9ish
    completer: k9ish-completion.bash
    plugs: [network, network-bind, dot-kube]

plugs:
  dot-kube:
    interface: personal-files
    read:  [$HOME/.kube]
    write: [$HOME/.kube]

parts:
  k9ish:
    plugin: go
    source: .
    override-pull: |
      craftctl default
      craftctl set version=$(git describe --long --tags --always --match=v*.*.* | sed 's/v//')
    override-build: |
      make test
      make build
      install -D "$SNAPCRAFT_PART_BUILD/execs/k9ish" "$SNAPCRAFT_PART_INSTALL/bin/k9ish"
      "$SNAPCRAFT_PART_INSTALL/bin/k9ish" completion bash > "$SNAPCRAFT_PART_INSTALL/k9ish-completion.bash"
    build-packages: [build-essential]
    build-snaps: [go]
```

Changes vs. the previous revision: no kubectl/micro parts (W3/W4); `source: .`
(builds this checkout). Local install: `snapcraft pack && sudo snap install
./k9ish*.snap --dangerous`; until the store declaration exists, run once:
`sudo snap connect k9ish:dot-kube`.

### W8 — README (minimal)

Rewrite `README.md` to a short page:

> **k9ish** is a fork of [K9s](https://github.com/derailed/k9s) — same TUI,
> modified so it can be packaged and distributed as a **strict-confinement
> snap** and to talk to clusters exclusively through the Kubernetes API.
>
> Not yet supported / removed compared to K9s:
> - resource & manifest editing (`:edit`, ctrl-e) and the external-editor
>   integration
> - `kubectl cp` (pod file upload/download)
> - apply/delete of local manifests/kustomizations from the `:dir` view
> - plugins (a content-interface mechanism is planned)
> - native clipboard helpers — OSC52 terminal clipboard only
> - kubeconfig/log/config paths outside `~/.kube` and the snap's own dirs
>
> For everything else (usage, configuration, key bindings, plugins for
> non-snap builds), see the [K9s README](…).

Plus a short "Building/installing the snap" block (`snapcraft pack`,
`--dangerous`, `snap connect k9ish:dot-kube`) and the `K9ISH_*` env-var table
with `K9S_*` fallbacks. All other README content links upstream instead of
being duplicated.

### W9 — Store / CI: deferred

Not in this iteration, per decision. (When the time comes: register the
`k9ish` snap name, forum auto-connect request for `dot-kube`, GitHub Actions
upload to `edge`.)

### W10 — Testing strategy (comprehensive)

**Layer 1 — Unit tests (run in `make test`, CI-able):**
- `internal/home`: table-driven precedence matrix
  (hint+HOME+passwd / no hint+remapped HOME / no hint+HOME unset / plain
  environment). Includes a no-sandbox simulation: set `HOME=/tmp/overridden`
  with a real-home hint → `Dir()` returns the real home (proves the generic
  behavior without snapd).
- `internal/config` env-alias helper: primary wins, fallback used, unset
  returns "".
- W2 flag seeding: `--kubeconfig` set → untouched; `$KUBECONFIG` set →
  untouched; neither → seeded with `home.Dir()/.kube/config`; seeded even if
  the file is missing (error-message check).
- W3 exec/attach: request-URL construction, flag/impersonation mapping,
  stream-options wiring — via fake REST client / recorded fixtures (no live
  cluster in unit tests).

**Layer 2 — Build gates:**
- `make lint && make test && make build` (static binary,
  `CGO_ENABLED=0`).
- `snapcraft pack` in managed mode for amd64 (and arm64 via
  `snapcraft remote-build` when available) — must succeed with
  `confinement: strict`.

**Layer 3 — Install & confinement smoke:**
- `sudo snap install ./k9ish*.snap --dangerous`
- `snap interfaces k9ish` shows exactly `network`, `network-bind`,
  `dot-kube` (until declaration: dot-kube connected manually via
  `snap connect`).
- `snap run --shell k9ish.k9ish -c 'env'`: `HOME` remapped to
  `$SNAP_USER_DATA`, hint var present, `$SNAP/bin` on PATH, completer file
  installed.
- AppArmor denial sweep while exercising all features:
  `sudo journalctl -k --since -10min | sudo snappy-debug.security scanlog`
  → zero unexpected denials.

**Layer 4 — Functional matrix (manual checklist, per release):**

| # | Scenario | Expected |
|---|---|---|
| 1 | Launch TUI with real `~/.kube/config` | connects; `k9ish info` shows snap-dir config paths |
| 2 | Switch context in TUI | **real** `~/.kube/config` modified (check mtime/content); no `~/snap/…/kube` copies appear |
| 3 | Browse/list resources across namespaces | works (network only) |
| 4 | `:shell` / ctrl-b exec into pod container | PTY works; resize propagates (resize terminal, rerun command); exit returns to TUI cleanly |
| 5 | `:attach` to a running container | streams logs/IO |
| 6 | Node shell (feature gate on, privileged pod) | shell pod created via API, native exec into it |
| 7 | Port-forward a service, `curl localhost:PORT` | listener binds (network-bind), traffic flows |
| 8 | `:pf` auto-forward annotation | same as 7 |
| 9 | Screen dump, benchmarks, logs | written under `~/snap/k9ish/current/…`; `K9ISH_CONFIG_DIR`/`K9ISH_LOGS_DIR` overrides honored when inside reachable roots |
| 10 | Completion | `k9ish completion bash` + snap completer sourced in bash |
| 11 | Skin/hotkey/alias/views files | loaded from snap config dir |
| 12 | Clipboard copy in OSC52-capable terminal | paste works; no crash on unsupported terminal |
| 13 | Missing kubeconfig (`home.Dir()/.kube/config` absent, no env/flag) | clear error naming the expected path |
| 14 | kubeconfig outside `~/.kube` via `--kubeconfig` | explicit failure message (documented limitation) |
| 15 | Plugin with host binary | fails with a clear message (documented) |
| 16 | `:edit`, ctrl-e, `kubectl cp`-equivalent actions | absent from help; no dead bindings |
| 17 | Read-only user (CanI) | exec/dump still fine; mutation attempts show RBAC errors |

**Layer 5 — Non-snap regression (generic-code proof):**
- Source build (`make build`): full behavior parity with k9s minus the
  removed features — host environment, no snap vars set.
- With `HOME` overridden + real-home hint exported (simulating any sandbox,
  no snapd involved): kubeconfig still discovered from the real home
  (validates W1/W2 generically).
- tarball/deb-style layout: k9ish in `/usr/local/bin` → identical behavior.

**Regression automation:** add the Layer-1 suites to CI; keep Layer 3/4 as a
checked-in checklist (`docs/snap-test-checklist.md`) to run per release.

## 4. Functionality intentionally changed for k9ish (all builds)

| Feature | Status |
|---|---|
| Resource/manifest editing (`:edit`, ctrl-e, screen-dump edit, xray edit) | **removed** (W4) |
| `kubectl cp` pod file transfer | **removed** (W3) |
| `:dir` apply/delete of manifests & kustomizations | **removed** (W3) |
| kubectl as a dependency for any flow | **removed** — exec/attach native (W3) |
| Plugins | mechanism kept; unusable in strict snap v1; content interface planned (W5) |
| Native clipboard helpers (xclip/wl-copy) | unreachable in snap; OSC52 fallback (W6 — no code change) |
| kubeconfig/log/config paths outside `~/.kube`/snap dirs | constrained by confinement (documented) |

Everything else — browsing, context switching, exec/shell, node-shell,
attach, port-forward, benchmarks, dumps, skins, hotkeys, completion — works
in every packaging.

## 5. Risks / open questions

- **Native exec fidelity** (W3): TTY, resize and signal handling are the
  riskiest part; mitigate by reusing `k8s.io/kubectl`'s stream/terminal
  helpers (already in `go.mod`) and keeping the bundled-kubectl fallback
  documented.
- **`dot-kube` auto-connect denial** — worst case: one documented
  `snap connect k9ish:dot-kube` per install; still functional.
- **`home.Dir()` precedence** — if a sandbox sets the real-home hint but
  user.Current() disagrees (root-in-container), ordering (hint → passwd →
  `$HOME`) must be locked down by tests (W10 L1).
- **Feature removals** are fork-level product decisions so the code stays
  snap-agnostic; called out explicitly in the README to set expectations vs
  upstream k9s.
- **arm64/armhf** — verify tcell/term and the exec path there; port-forward
  address default (`localhost`) works unchanged.

## 6. Suggested commit sequence

1. `feat(home): add internal/home real-home resolution` (+ unit tests)
2. `feat(client): discover default kubeconfig from the real home when $HOME is overridden`
3. `feat(view): native pod exec/attach via client-go remotecommand`
4. `feat!: remove editor functionality (edit, ctrl-e, screen-dump/xray/dir edit)`
5. `feat!: drop kubectl cp and :dir apply/delete`
6. `refactor: k9s → k9ish branding (AppName, env aliases with K9S_ fallback, Makefile, strings)`
7. `build(snap): strict confinement, dot-kube plug, single-part build`
8. `docs: minimal k9ish README + STRICT_SNAP implementation status`
9. `test: snap functional checklist + unit suites wired into CI`
