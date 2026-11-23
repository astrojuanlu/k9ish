// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package config

import (
	_ "embed"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/derailed/k9s/internal/config/data"
	"github.com/derailed/k9s/internal/env"
	"github.com/derailed/k9s/internal/slogs"
)

const (
	// EnvConfigDir represents the k9ish configuration dir env var.
	EnvConfigDir = "K9ISH_CONFIG_DIR"

	// EnvConfigDirFallback is the legacy alias for EnvConfigDir.
	EnvConfigDirFallback = "K9S_CONFIG_DIR"

	// EnvLogsDir represents the k9ish logs dir env var.
	EnvLogsDir = "K9ISH_LOGS_DIR"

	// EnvLogsDirFallback is the legacy alias for EnvLogsDir.
	EnvLogsDirFallback = "K9S_LOGS_DIR"

	// AppName tracks the k9ish app name.
	AppName = "k9ish"

	K9sLogsFile = "k9ish.log"
)

var (
	//go:embed templates/benchmarks.yaml
	// benchmarkTpl tracks benchmark default config template
	benchmarkTpl []byte

	//go:embed templates/aliases.yaml
	// aliasesTpl tracks aliases default config template
	aliasesTpl []byte

	//go:embed templates/hotkeys.yaml
	// hotkeysTpl tracks hotkeys default config template
	hotkeysTpl []byte

	//go:embed templates/stock-skin.yaml
	// stockSkinTpl tracks stock skin template
	stockSkinTpl []byte
)

var (
	// AppConfigDir tracks main k9s config home directory.
	AppConfigDir string

	// AppSkinsDir tracks skins data directory.
	AppSkinsDir string

	// AppBenchmarksDir tracks benchmarks results directory.
	AppBenchmarksDir string

	// AppDumpsDir tracks screen dumps data directory.
	AppDumpsDir string

	// AppContextsDir tracks contexts data directory.
	AppContextsDir string

	// AppConfigFile tracks k9s config file.
	AppConfigFile string

	// AppLogFile tracks k9s logs file.
	AppLogFile string

	// AppViewsFile tracks custom views config file.
	AppViewsFile string

	// AppJumpsFile tracks custom jumps config file.
	AppJumpsFile string

	// AppAliasesFile tracks aliases config file.
	AppAliasesFile string

	// AppPluginsFile tracks plugins config file.
	AppPluginsFile string

	// AppHotKeysFile tracks hotkeys config file.
	AppHotKeysFile string
)

// InitLogLoc initializes k9ish logs location.
func InitLogLoc() error {
	var appLogDir string
	switch {
	case env.IsSet(EnvLogsDir, EnvLogsDirFallback):
		appLogDir = env.Get(EnvLogsDir, EnvLogsDirFallback)
	case env.IsSet(EnvConfigDir, EnvConfigDirFallback):
		tmpDir, err := UserTmpDir()
		if err != nil {
			return err
		}
		appLogDir = tmpDir
	default:
		var err error
		appLogDir, err = xdg.StateFile(AppName)
		if err != nil {
			return err
		}
	}
	if err := data.EnsureFullPath(appLogDir, data.DefaultDirMod); err != nil {
		return err
	}
	AppLogFile = filepath.Join(appLogDir, K9sLogsFile)

	return nil
}

// InitLocs initializes k9s artifacts locations.
func InitLocs() error {
	if env.IsSet(EnvConfigDir, EnvConfigDirFallback) {
		return initEnvLocs()
	}

	return initXDGLocs()
}

func initEnvLocs() error {
	AppConfigDir = env.Get(EnvConfigDir, EnvConfigDirFallback)
	if err := data.EnsureFullPath(AppConfigDir, data.DefaultDirMod); err != nil {
		return err
	}

	AppDumpsDir = filepath.Join(AppConfigDir, "screen-dumps")
	if err := data.EnsureFullPath(AppDumpsDir, data.DefaultDirMod); err != nil {
		slog.Warn("Unable to create screen-dumps dir", slogs.Dir, AppDumpsDir, slogs.Error, err)
	}
	AppBenchmarksDir = filepath.Join(AppConfigDir, "benchmarks")
	if err := data.EnsureFullPath(AppBenchmarksDir, data.DefaultDirMod); err != nil {
		slog.Warn("Unable to create benchmarks dir",
			slogs.Dir, AppBenchmarksDir,
			slogs.Error, err,
		)
	}
	AppSkinsDir = filepath.Join(AppConfigDir, "skins")
	if err := data.EnsureFullPath(AppSkinsDir, data.DefaultDirMod); err != nil {
		slog.Warn("Unable to create skins dir",
			slogs.Dir, AppSkinsDir,
			slogs.Error, err,
		)
	}
	AppContextsDir = filepath.Join(AppConfigDir, "clusters")
	if err := data.EnsureFullPath(AppContextsDir, data.DefaultDirMod); err != nil {
		slog.Warn("Unable to create clusters dir",
			slogs.Dir, AppContextsDir,
			slogs.Error, err,
		)
	}

	AppConfigFile = filepath.Join(AppConfigDir, data.MainConfigFile)
	AppHotKeysFile = filepath.Join(AppConfigDir, "hotkeys.yaml")
	AppAliasesFile = filepath.Join(AppConfigDir, "aliases.yaml")
	AppPluginsFile = filepath.Join(AppConfigDir, "plugins.yaml")
	AppViewsFile = filepath.Join(AppConfigDir, "views.yaml")
	AppJumpsFile = filepath.Join(AppConfigDir, "jumps.yaml")

	return nil
}

func initXDGLocs() error {
	var err error

	AppConfigDir, err = xdg.ConfigFile(AppName)
	if err != nil {
		return err
	}

	AppConfigFile, err = xdg.ConfigFile(filepath.Join(AppName, data.MainConfigFile))
	if err != nil {
		return err
	}

	AppHotKeysFile = filepath.Join(AppConfigDir, "hotkeys.yaml")
	AppAliasesFile = filepath.Join(AppConfigDir, "aliases.yaml")
	AppPluginsFile = filepath.Join(AppConfigDir, "plugins.yaml")
	AppViewsFile = filepath.Join(AppConfigDir, "views.yaml")
	AppJumpsFile = filepath.Join(AppConfigDir, "jumps.yaml")

	AppSkinsDir = filepath.Join(AppConfigDir, "skins")
	if e := data.EnsureFullPath(AppSkinsDir, data.DefaultDirMod); e != nil {
		slog.Warn("No skins dir detected", slogs.Error, e)
	}

	AppDumpsDir, err = xdg.StateFile(filepath.Join(AppName, "screen-dumps"))
	if err != nil {
		return err
	}

	AppBenchmarksDir, err = xdg.StateFile(filepath.Join(AppName, "benchmarks"))
	if err != nil {
		slog.Warn("No benchmarks dir detected",
			slogs.Dir, AppBenchmarksDir,
			slogs.Error, err,
		)
	}

	dataDir, err := xdg.DataFile(AppName)
	if err != nil {
		return err
	}
	AppContextsDir = filepath.Join(dataDir, "clusters")
	if err := data.EnsureFullPath(AppContextsDir, data.DefaultDirMod); err != nil {
		slog.Warn("No context dir detected",
			slogs.Dir, AppContextsDir,
			slogs.Error, err,
		)
	}

	return nil
}

// AppContextDir generates a valid context config dir.
func AppContextDir(cluster, context string) string {
	return filepath.Join(AppContextsDir, data.SanitizeContextSubpath(cluster, context))
}

// AppContextAliasesFile generates a valid context specific aliases file path.
func AppContextAliasesFile(cluster, context string) string {
	return filepath.Join(AppContextsDir, data.SanitizeContextSubpath(cluster, context), "aliases.yaml")
}

// AppContextPluginsFile generates a valid context specific plugins file path.
func AppContextPluginsFile(cluster, context string) string {
	return filepath.Join(AppContextsDir, data.SanitizeContextSubpath(cluster, context), "plugins.yaml")
}

// AppContextHotkeysFile generates a valid context specific hotkeys file path.
func AppContextHotkeysFile(cluster, context string) string {
	return filepath.Join(AppContextsDir, data.SanitizeContextSubpath(cluster, context), "hotkeys.yaml")
}

// AppContextConfig generates a valid context config file path.
func AppContextConfig(cluster, context string) string {
	return filepath.Join(AppContextDir(cluster, context), data.MainConfigFile)
}

// DumpsDir generates a valid context dump directory.
func DumpsDir(cluster, context string) (string, error) {
	dir := filepath.Join(AppDumpsDir, data.SanitizeContextSubpath(cluster, context))

	return dir, data.EnsureDirPath(dir, data.DefaultDirMod)
}

// EnsureBenchmarksDir generates a valid benchmark results directory.
func EnsureBenchmarksDir(cluster, context string) (string, error) {
	dir := filepath.Join(AppBenchmarksDir, data.SanitizeContextSubpath(cluster, context))

	return dir, data.EnsureDirPath(dir, data.DefaultDirMod)
}

// EnsureBenchmarksCfgFile generates a valid benchmark file.
func EnsureBenchmarksCfgFile(cluster, context string) (string, error) {
	f := filepath.Join(AppContextDir(cluster, context), "benchmarks.yaml")
	if err := data.EnsureDirPath(f, data.DefaultDirMod); err != nil {
		return "", err
	}
	if _, err := os.Stat(f); errors.Is(err, fs.ErrNotExist) {
		return f, os.WriteFile(f, benchmarkTpl, data.DefaultFileMod)
	}

	return f, nil
}

// EnsureAliasesCfgFile generates a valid aliases file.
func EnsureAliasesCfgFile() (string, error) {
	f := filepath.Join(AppConfigDir, "aliases.yaml")
	if err := data.EnsureDirPath(f, data.DefaultDirMod); err != nil {
		return "", err
	}
	if _, err := os.Stat(f); errors.Is(err, fs.ErrNotExist) {
		return f, os.WriteFile(f, aliasesTpl, data.DefaultFileMod)
	}

	return f, nil
}

// EnsureHotkeysCfgFile generates a valid hotkeys file.
func EnsureHotkeysCfgFile() (string, error) {
	f := filepath.Join(AppConfigDir, "hotkeys.yaml")
	if err := data.EnsureDirPath(f, data.DefaultDirMod); err != nil {
		return "", err
	}
	if _, err := os.Stat(f); errors.Is(err, fs.ErrNotExist) {
		return f, os.WriteFile(f, hotkeysTpl, data.DefaultFileMod)
	}

	return f, nil
}

// SkinFileFromName generate skin file path from spec.
func SkinFileFromName(n string) string {
	if n == "" {
		n = "stock"
	}

	return filepath.Join(AppSkinsDir, n+".yaml")
}
