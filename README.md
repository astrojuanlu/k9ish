# k9ish

**k9ish** is a fork of [K9s](https://github.com/derailed/k9s) — same TUI,
modified so it can be packaged and distributed as a **strict-confinement
snap** and to talk to clusters exclusively through the Kubernetes API.

Not yet supported / removed compared to K9s:

- resource & manifest editing (`:edit`, ctrl-e) and the external-editor
  integration
- `kubectl cp` (pod file upload/download)
- apply/delete of local manifests/kustomizations from the `:dir` view
- plugins (a content-interface mechanism is planned)
- native clipboard helpers — OSC52 terminal clipboard only
- kubeconfig/log/config paths outside `~/.kube` and the snap's own dirs
- Windows builds — Linux and macOS only

For everything else (usage, configuration, key bindings, plugins for
non-snap builds), see the [K9s README](https://github.com/derailed/k9s/blob/master/README.md).

## Building and installing the snap

```sh
snapcraft pack
sudo snap install ./k9ish_*.snap --dangerous
```

Until the store grants auto-connect for the `dot-kube` plug, run once:

```sh
sudo snap connect k9ish:dot-kube
```

## Environment variables

Current variable names come first; the legacy `K9S_*` names are honored as
fallback aliases for backward compatibility.

| Primary | Legacy alias | Purpose |
|---|---|---|
| `K9ISH_CONFIG_DIR` | `K9S_CONFIG_DIR` | Override the k9ish configuration directory |
| `K9ISH_LOGS_DIR` | `K9S_LOGS_DIR` | Override the k9ish logs directory |
| `K9ISH_DEFAULT_PF_ADDRESS` | `K9S_DEFAULT_PF_ADDRESS` | Port-forward bind address (default `localhost`) |
| `K9ISH_SKIN` | `K9S_SKIN` | Skin to load from the environment |
| `K9ISH_FEATURE_GATE_NODE_SHELL` | `K9S_FEATURE_GATE_NODE_SHELL` | Enable/disable node shell (`true`/`false`) |
| `K9ISH_CLIPBOARD` | `K9S_CLIPBOARD` | Clipboard mode: `auto`, `native`, `osc52` |
| `K9ISH_OSC52_MAX` | `K9S_OSC52_MAX` | Max OSC52 payload length |

`KUBECONFIG`/`--kubeconfig` behave as usual and take precedence over the
default `~/.kube/config` discovery.
