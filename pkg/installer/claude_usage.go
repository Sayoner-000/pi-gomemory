package installer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ClaudeTokenBucket struct {
	Input       int64 `json:"input"`
	Output      int64 `json:"output"`
	CacheCreate int64 `json:"cache_create"`
	CacheRead   int64 `json:"cache_read"`
	Messages    int   `json:"messages"`
}

func (b *ClaudeTokenBucket) Total() int64 {
	return b.Input + b.Output + b.CacheCreate + b.CacheRead
}

type ClaudeProjectUsage struct {
	Name string `json:"name"`
	ClaudeTokenBucket
}

type ClaudeUsageSummary struct {
	ConfigDir string               `json:"configDir"`
	Sessions  int                  `json:"sessions"`
	Today     ClaudeTokenBucket    `json:"today"`
	Week      ClaudeTokenBucket    `json:"week"`
	Total     ClaudeTokenBucket    `json:"total"`
	Projects  []ClaudeProjectUsage `json:"projects"`
}

type tokenCacheFile struct {
	Version int                        `json:"version"`
	Files   map[string]cachedFileEntry `json:"files"`
}

type cachedFileEntry struct {
	Size    int64            `json:"size"`
	Mtime   float64          `json:"mtime"`
	Project string           `json:"project"`
	Rows    map[string][]any `json:"rows"` // [timestamp, model, in, out, cache_create, cache_read]
}

// FetchClaudeUsage lee la caché de Claude Code (~/.claude/token-usage-cache.json)
// y agrega los totales de hoy, últimos 7 días y acumulado total.
func FetchClaudeUsage(claudeDir string) (*ClaudeUsageSummary, error) {
	if claudeDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		claudeDir = filepath.Join(home, ".claude")
	}

	cachePath := filepath.Join(claudeDir, "token-usage-cache.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("no se encontró telemetría de Claude Code en %s", cachePath)
	}

	var cache tokenCacheFile
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsear caché de tokens: %w", err)
	}

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	weekAgoStr := now.AddDate(0, 0, -6).Format("2006-01-02")

	summary := &ClaudeUsageSummary{
		ConfigDir: claudeDir,
	}

	seenKeys := make(map[string]bool)
	sessionsSet := make(map[string]bool)
	projectMap := make(map[string]*ClaudeTokenBucket)

	sortedPaths := make([]string, 0, len(cache.Files))
	for p := range cache.Files {
		sortedPaths = append(sortedPaths, p)
	}
	sort.Strings(sortedPaths)

	for _, path := range sortedPaths {
		entry := cache.Files[path]
		sessName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		sessionsSet[sessName] = true

		projName := entry.Project
		if projName == "" {
			projName = filepath.Base(filepath.Dir(path))
		}
		projBucket, ok := projectMap[projName]
		if !ok {
			projBucket = &ClaudeTokenBucket{}
			projectMap[projName] = projBucket
		}

		// Ordenar las claves de rows dentro del archivo para consistencia
		rowKeys := make([]string, 0, len(entry.Rows))
		for k := range entry.Rows {
			rowKeys = append(rowKeys, k)
		}
		sort.Strings(rowKeys)

		for _, key := range rowKeys {
			row := entry.Rows[key]
			if seenKeys[key] {
				continue
			}
			seenKeys[key] = true

			if len(row) < 6 {
				continue
			}

			tsStr, _ := row[0].(string)
			inTok := toInt64(row[2])
			outTok := toInt64(row[3])
			cCreate := toInt64(row[4])
			cRead := toInt64(row[5])

			// Acumular Total
			addTokens(&summary.Total, inTok, outTok, cCreate, cRead)
			addTokens(projBucket, inTok, outTok, cCreate, cRead)

			// Filtrar por fecha en zona horaria local
			dayStr := ""
			if ts, err := time.Parse(time.RFC3339, tsStr); err == nil {
				dayStr = ts.In(time.Local).Format("2006-01-02")
			} else if len(tsStr) >= 10 {
				dayStr = tsStr[:10]
			}

			if dayStr == todayStr {
				addTokens(&summary.Today, inTok, outTok, cCreate, cRead)
			}
			if dayStr >= weekAgoStr && dayStr <= todayStr {
				addTokens(&summary.Week, inTok, outTok, cCreate, cRead)
			}
		}
	}

	summary.Sessions = len(sessionsSet)

	for pName, pBucket := range projectMap {
		summary.Projects = append(summary.Projects, ClaudeProjectUsage{
			Name:              pName,
			ClaudeTokenBucket: *pBucket,
		})
	}

	sort.Slice(summary.Projects, func(i, j int) bool {
		return summary.Projects[i].Total() > summary.Projects[j].Total()
	})

	return summary, nil
}

func addTokens(b *ClaudeTokenBucket, inTok, outTok, cCreate, cRead int64) {
	b.Input += inTok
	b.Output += outTok
	b.CacheCreate += cCreate
	b.CacheRead += cRead
	b.Messages++
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}

// FormatHumanNumber convierte cantidades grandes en sufijos k, M, B
func FormatHumanNumber(n int64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
