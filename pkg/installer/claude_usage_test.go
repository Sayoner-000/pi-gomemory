package installer

import (
	"path/filepath"
	"testing"
)

func TestClaudeUsageAggregator(t *testing.T) {
	opts := sandbox(t)
	claudeDir := filepath.Join(opts.PiAgentDir, "fake_claude")

	cacheData := `{
  "version": 3,
  "files": {
    "/fake/path/projA/session1.jsonl": {
      "size": 100,
      "mtime": 12345.0,
      "project": "proyecto-alpha",
      "rows": {
        "msg1|req1": ["2026-11-11T12:00:00Z", "claude-sonnet-5.5", 1000, 200, 500, 8000],
        "msg2|req2": ["2026-11-10T12:00:00Z", "claude-sonnet-5.5", 2000, 300, 1000, 12000]
      }
    }
  }
}`
	writeFile(t, filepath.Join(claudeDir, "token-usage-cache.json"), cacheData)

	res, err := FetchClaudeUsage(claudeDir)
	if err != nil {
		t.Fatalf("FetchClaudeUsage fallo: %v", err)
	}

	if res.Sessions != 1 {
		t.Errorf("Sessions = %d, esperado 1", res.Sessions)
	}
	if res.Total.Input != 3000 || res.Total.Output != 500 {
		t.Errorf("Total = %+v, esperado in:3000 out:500", res.Total)
	}
	if len(res.Projects) != 1 || res.Projects[0].Name != "proyecto-alpha" {
		t.Errorf("Projects = %+v", res.Projects)
	}
}
