package cache

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"github.com/gavv/md-authors/src/logs"
)

var Refresh = false

var (
	memCache  map[string]string   = make(map[string]string)
	diskCache map[string]string   = make(map[string]string)
	reCache   map[string]struct{} = make(map[string]struct{})
	diskFile  string
	diskOnce  sync.Once
)

func diskInit() {
	diskOnce.Do(func() {
		diskDir, _ := os.UserCacheDir()
		diskFile = diskDir + "/mdauthors.json"

		b, _ := os.ReadFile(diskFile)
		if err := json.Unmarshal(b, &diskCache); err != nil {
			diskCache = make(map[string]string)
		}

		logs.Debugf("loaded %d entries from %q", len(diskCache), diskFile)
	})
}

func DiskStore(keys []string, value string) {
	diskInit()

	key := strings.Join(keys, ":")

	if val, ok := diskCache[key]; ok && val == value {
		return
	}

	logs.Debugf("cache store: %q %q", key, value)
	diskCache[key] = value

	b, _ := json.MarshalIndent(diskCache, "", " ")
	os.WriteFile(diskFile, b, 0644)
}

func DiskLoad(keys []string) (string, bool) {
	diskInit()

	key := strings.Join(keys, ":")

	if value, ok := diskCache[key]; ok {
		if Refresh {
			if _, ok := reCache[key]; !ok {
				logs.Debugf("cache reset: %q", key)
				reCache[key] = struct{}{}
				return "", false
			}
		}

		logs.Debugf("cache hit: %q %q", key, value)
		return value, true
	}

	logs.Debugf("cache miss: %q", key)
	return "", false
}

func MemStore(keys []string, value string) {
	key := strings.Join(keys, ":")

	memCache[key] = value
}

func MemLoad(keys []string) (string, bool) {
	key := strings.Join(keys, ":")

	if value, ok := memCache[key]; ok {
		return value, true
	}

	return "", false
}
