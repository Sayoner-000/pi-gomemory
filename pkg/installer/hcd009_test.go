package installer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLatestTagFromRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			http.Redirect(w, r, "/owner/repo/releases/tag/v9.8.7", http.StatusFound)
		case "/norelease":
			http.Redirect(w, r, "/owner/repo/releases", http.StatusFound)
		default:
			t.Errorf("no debe seguir la redirección: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	tag, err := fetchLatestTag(srv.URL+"/releases/latest", 2*time.Second)
	if err != nil || tag != "v9.8.7" {
		t.Fatalf("tag = %q, %v", tag, err)
	}
	if _, err := fetchLatestTag(srv.URL+"/norelease", 2*time.Second); err == nil {
		t.Fatal("una redirección sin /releases/tag/ debe dar error")
	}
	var hits int32
	api := newReleaseServer("v1.0.1", &hits)
	defer api.Close()
	if tag, err := fetchLatestTag(api.URL, 2*time.Second); err != nil || tag != "v1.0.1" {
		t.Fatalf("JSON: %q, %v", tag, err)
	}
}

func TestNoGitHubAPI(t *testing.T) {
	if strings.Contains(DefaultReleasesAPI, "api.github.com") {
		t.Fatalf("DefaultReleasesAPI = %s", DefaultReleasesAPI)
	}
	src, err := os.ReadFile("deps.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "api.github.com") {
		t.Fatal("deps.go no debe consultar api.github.com")
	}
}
