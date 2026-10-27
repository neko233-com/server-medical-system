package config

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// =============================================================================
//  .env 支持
// =============================================================================
//  生产落地时，运维同学更习惯把配置写进 .env 文件而不是到处 export。
//  这里提供一个零依赖的轻量解析器（刻意不引第三方库，保持单二进制纯净）。
//
//  规则：
//   1. 真实环境变量优先级最高 —— .env 中的同名键**不会**覆盖已有环境变量，
//      保证容器 / systemd 的 Environment 始终能压过文件配置；
//   2. 支持 # 注释、空行、export 前缀、单双引号包裹；
//   3. 文件不存在属正常情况（全默认部署），不打日志、不报错。
// =============================================================================

// DefaultEnvFile is the file Load() attempts to read at startup.
const DefaultEnvFile = ".env"

// LoadDotEnv parses path and applies each key=value pair to the process
// environment unless that key is already set.
func LoadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		// Absent .env is a valid deployment mode (everything defaulted/injected).
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(sc.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		// Tolerate `export KEY=VALUE` which people paste from shell profiles.
		raw = strings.TrimPrefix(raw, "export ")
		eq := strings.Index(raw, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(raw[:eq])
		val := strings.TrimSpace(raw[eq+1:])
		if key == "" {
			continue
		}
		// Strip matching surrounding quotes.
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') ||
				(val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		// Existing environment always wins.
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		if err := os.Setenv(key, val); err != nil {
			log.Printf("config: set %s failed: %v", key, err)
		}
	}
}

// EnvFiles returns the ordered list of env files Load() will attempt. Later
// entries can be added to support per-environment overlays (e.g. .env.local).
//
// Search order:
//  1. the explicit ENV_FILE override (if set) — used verbatim;
//  2. "./.env" relative to the process working directory;
//  3. ".env" next to the executable — this is what makes Windows services and
//     systemd units work, because neither reliably sets a working directory.
//
// Missing files are silently skipped: running with pure environment variables
// is a perfectly valid deployment mode.
func EnvFiles(override string) []string {
	if override != "" {
		return []string{override}
	}
	files := []string{DefaultEnvFile}
	// Executable-adjacent .env — resolved once, best effort.
	if exe, err := os.Executable(); err == nil {
		adjacent := filepath.Join(filepath.Dir(exe), DefaultEnvFile)
		// Avoid listing the same path twice when CWD == executable dir.
		if abs, aerr := filepath.Abs(adjacent); aerr == nil {
			if cwd, cerr := os.Getwd(); cerr == nil {
				if absCwd, _ := filepath.Abs(cwd); absCwd != abs {
					files = append(files, adjacent)
				}
			}
		}
	}
	return files
}
