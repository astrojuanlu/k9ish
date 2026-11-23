# k9ish snap test checklist

Manual verification to run per release, on top of the automated unit suites
(`make test`). Layers below mirror the project testing strategy.

## Layer 2 — Build gates

- [ ] `make lint && make test && make build` pass (static binary,
  `CGO_ENABLED=0`).
- [ ] `snapcraft pack` succeeds with `confinement: strict` (amd64; arm64 via
  `snapcraft remote-build` when available).

## Layer 3 — Install & confinement smoke

- [ ] `sudo snap install ./k9ish_*.snap --dangerous`
- [ ] `snap interfaces k9ish` shows exactly `network`, `network-bind`,
  `dot-kube` (until the store declaration exists, connect manually via
  `sudo snap connect k9ish:dot-kube`).
- [ ] `snap run --shell k9ish.k9ish -c 'env'`: `HOME` remapped to
  `$SNAP_USER_DATA`, real-home hint variable present, `$SNAP/bin` on `PATH`,
  completer file installed.
- [ ] AppArmor denial sweep while exercising all features:
  `sudo journalctl -k --since -10min | sudo snappy-debug.security scanlog`
  → zero unexpected denials.

## Layer 4 — Functional matrix

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
| 13 | Missing kubeconfig (`~/.kube/config` absent, no env/flag) | clear error naming the expected path |
| 14 | kubeconfig outside `~/.kube` via `--kubeconfig` | explicit failure message (documented limitation) |
| 15 | Plugin with host binary | fails with a clear message (documented) |
| 16 | `:edit`, ctrl-e, `kubectl cp`-equivalent actions | absent from help; no dead bindings |
| 17 | Read-only user (CanI) | exec/dump still fine; mutation attempts show RBAC errors |

## Layer 5 — Non-snap regression (generic-code proof)

- [ ] Source build (`make build`): full behavior parity with k9s minus the
  removed features — host environment, no snap variables set.
- [ ] `HOME` overridden + real-home hint exported (simulating any sandbox,
  no snapd involved): kubeconfig still discovered from the real home.
- [ ] tarball/deb-style layout: k9ish in `/usr/local/bin` → identical
  behavior.
