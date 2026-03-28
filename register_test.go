package ginknife4j

import (
	"strings"
	"testing"
)

func TestInjectHostSettingsScript(t *testing.T) {
	pageHTML := "<html><body><div id=\"app\"></div><script src=\"webjars/js/app.js\"></script></body></html>"
	result := injectHostSettingsScript(pageHTML, "https://example.com", "/swagger/v1")

	if !strings.Contains(result, "Knife4jGlobalSettings") {
		t.Fatalf("expected local storage key in injected script")
	}
	if !strings.Contains(result, "\"https://example.com\"") {
		t.Fatalf("expected host url in injected script")
	}
	if !strings.Contains(result, "settings.enableHost=true") {
		t.Fatalf("expected enableHost=true in injected script")
	}
	if !strings.Contains(result, "history.replaceState") {
		t.Fatalf("expected history.replaceState in injected script")
	}
	if !strings.Contains(result, `"/swagger/v1/"`) {
		t.Fatalf("expected routePrefix/ in replaceState target")
	}

	replaceIdx := strings.Index(result, "history.replaceState")
	bundleIdx := strings.Index(result, "<script src=\"webjars/js/app.js\"></script>")
	if replaceIdx == -1 || bundleIdx == -1 || replaceIdx > bundleIdx {
		t.Fatalf("expected injected script before bundle script")
	}
}

func TestInjectHostSettingsScriptEmptyHost(t *testing.T) {
	pageHTML := "<html><body><script src=\"app.js\"></script></body></html>"
	result := injectHostSettingsScript(pageHTML, "", "/swagger/v1")

	if !strings.Contains(result, "history.replaceState") {
		t.Fatalf("expected replaceState even without hostURL")
	}
	if strings.Contains(result, "enableHost") {
		t.Fatalf("should not inject enableHost when hostURL is empty")
	}
}

func TestBuildHostOnlyURL(t *testing.T) {
	tests := []struct {
		name   string
		server docServer
		want   string
	}{
		{
			name:   "完整 host 信息",
			server: docServer{Scheme: "https", Host: "example.com", BasePath: "/user/api/v1"},
			want:   "https://example.com",
		},
		{
			name:   "省略 scheme 默认 http",
			server: docServer{Host: "localhost:6668", BasePath: "/v1"},
			want:   "http://localhost:6668",
		},
		{
			name:   "空 host 返回空",
			server: docServer{Scheme: "https", BasePath: "/v1"},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildHostOnlyURL(tt.server)
			if got != tt.want {
				t.Fatalf("buildHostOnlyURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInjectBaseHref(t *testing.T) {
	pageHTML := `<html><head><title>Knife4j</title></head><body></body></html>`
	result := injectBaseHref(pageHTML, "/swagger/v1")

	if !strings.Contains(result, `<base href="/swagger/v1/">`) {
		t.Fatalf("expected base href tag, got %q", result)
	}
	baseIdx := strings.Index(result, "<base ")
	titleIdx := strings.Index(result, "<title>")
	if baseIdx == -1 || titleIdx == -1 || baseIdx > titleIdx {
		t.Fatalf("expected base href BEFORE title tag")
	}
}

func TestEnableHostTextBothModesUseOriginOnly(t *testing.T) {
	server := docServer{Scheme: "https", Host: "example.com", BasePath: "/user/api/v1"}

	// 两种模式均使用 origin-only
	hostOnly := buildHostOnlyURL(server)
	if hostOnly != "https://example.com" {
		t.Fatalf("enableHostText should be origin only, got %q", hostOnly)
	}
}
