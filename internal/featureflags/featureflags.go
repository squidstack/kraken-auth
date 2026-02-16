package featureflags

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rollout/rox-go/v5/server"
)

const defaultFMPath = "/app/config/fm.json"

// Flags holds all feature flags this service cares about.
type Flags struct {
	// Boolean example: enable tutorial or some demo behavior
	EnableTutorial server.RoxFlag

	// String example: log level ("info","debug","warn")
	LogLevel server.RoxString

	// Boolean “kill-switch” to put the API in offline mode
	Offline server.RoxFlag
}

var (
	flags = &Flags{
		LogLevel: server.NewRoxString("info", []string{"debug", "info", "warn", "error"}),
		Offline:  server.NewRoxFlag(false),
	}

	rox *server.Rox
)

// readEnvKey tries to read either a raw key or {"envKey":"..."} from path.
func readEnvKey(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	s := strings.TrimSpace(string(b))
	if s == "" {
		return "", fmt.Errorf("file %s empty", path)
	}
	// If it looks like JSON, try to parse {"envKey":"..."}.
	if strings.HasPrefix(s, "{") {
		var obj struct {
			EnvKey string `json:"envKey"`
			Key    string `json:"key"` // tolerate other field names
		}
		if err := json.Unmarshal([]byte(s), &obj); err != nil {
			return "", fmt.Errorf("parse JSON in %s: %w", path, err)
		}
		if obj.EnvKey != "" {
			return obj.EnvKey, nil
		}
		if obj.Key != "" {
			return obj.Key, nil
		}
		return "", fmt.Errorf("no envKey in JSON %s", path)
	}
	// Otherwise treat the whole file as the key.
	return s, nil
}

// Init sets up the CloudBees Feature Management SDK.
// fmPath can be "" to use the default (/app/config/fm.json).
func Init(ctx context.Context, fmPath string) error {
	if fmPath == "" {
		fmPath = defaultFMPath
	}
	key, err := readEnvKey(fmPath)
	if err != nil {
		// Non-fatal: just log and continue without FM
		fmt.Printf("[featureflags] no FM key: %v (flags will use defaults)\n", err)
		return nil
	}

	// Build options (tweak as desired)
	opts := server.NewRoxOptions(server.RoxOptionsBuilder{
		// Poll every 60s by default; you can set Version, Logger etc.
		FetchInterval: 60 * time.Second,
	})

	rox = server.NewRox()
	ns := os.Getenv("FM_NAMESPACE")
	if ns == "" {
		ns = "default" // or leave empty if you want no prefix
	}
	rox.Register(ns, flags)

	// Block until initial fetch completes (or returns error)
	if setupErr := <-rox.Setup(key, opts); setupErr != nil {
		// Non-fatal: fall back to defaults
		fmt.Printf("[featureflags] setup error: %v (flags will use defaults)\n", setupErr)
	}
	return nil
}

// Values exposes a read-only view of the flags.
func Values() *Flags { return flags }

// Shutdown gracefully stops the SDK (optional).
func Shutdown() {
	if rox == nil {
		return
	}
	<-rox.Shutdown()
}

//
