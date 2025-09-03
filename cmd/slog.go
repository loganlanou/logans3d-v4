package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/lmittmann/tint"
)

var once sync.Once

func init() {
	once.Do(func() {
		// Get module name dynamically from runtime build info
		modulePrefix := getModulePrefix()

		logLevel := slog.LevelInfo
		if logLevelStr := os.Getenv("LOG_LEVEL"); logLevelStr != "" {
			if err := logLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
				panic(fmt.Sprintf("invalid log level: %s", logLevelStr))
			}
		}

		if logLevel == slog.LevelDebug {
			replacer := func(_ []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					if source, ok := a.Value.Any().(*slog.Source); ok {
						// Clean up the file path using the module prefix
						source.File = cleanSourcePath(source.File, modulePrefix)
					}
				}
				if err, ok := a.Value.Any().(error); ok {
					aErr := tint.Err(err)
					aErr.Key = a.Key
					return aErr
				}
				return a
			}

			handler := tint.NewHandler(os.Stdout, &tint.Options{
				Level:       slog.LevelDebug,
				TimeFormat:  time.TimeOnly,
				ReplaceAttr: replacer,
				AddSource:   true,
			})

			slog.SetDefault(slog.New(handler))
			slog.Info("debug logging enabled")
			return
		}

		// Set up the logger to be json output
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
		slog.Info("json logging enabled")
	})
}

// getModulePrefix extracts the module path from runtime build info
// and returns a prefix that can be used to clean source paths
func getModulePrefix() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Path == "" {
		// Fallback to current working directory approach
		if wd, err := os.Getwd(); err == nil {
			return "/" + filepath.Base(wd) + "/"
		}
		// Ultimate fallback
		return "/logans3d-v4/"
	}

	// Use the last component of the module path
	// e.g., "github.com/loganlanou/logans3d-v4" -> "/logans3d-v4/"
	parts := strings.Split(info.Main.Path, "/")
	if len(parts) > 0 {
		return "/" + parts[len(parts)-1] + "/"
	}

	return "/" + info.Main.Path + "/"
}

// cleanSourcePath removes the module prefix from the file path to make logs more readable
func cleanSourcePath(filePath, modulePrefix string) string {
	// Split the file path on the module name, and keep the last half
	// This makes the logs more readable by showing just the relative path
	parts := strings.Split(filePath, modulePrefix)
	if len(parts) == 2 {
		return parts[1]
	}

	// If we can't split on the module prefix, try to clean it up another way
	// Remove common Go path prefixes that aren't useful
	cleaned := filePath
	if idx := strings.LastIndex(cleaned, "/go/src/"); idx != -1 {
		cleaned = cleaned[idx+8:]
	} else if idx := strings.LastIndex(cleaned, "/src/"); idx != -1 {
		cleaned = cleaned[idx+5:]
	}

	return cleaned
}