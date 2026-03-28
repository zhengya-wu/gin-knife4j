package ginknife4j

import (
	"encoding/json"
	"testing"
)

func TestPatchDocJSONSwagger2(t *testing.T) {
	raw := []byte(`{
		"swagger":"2.0",
		"info":{"title":"demo","version":"v1"},
		"paths":{
			"/mcp/square/list":{
				"get":{
					"responses":{"200":{"description":"ok"}}
				}
			}
		}
	}`)

	patched := patchDocJSON(raw, Config{}, docServer{
		Scheme:   "https",
		Host:     "example.com",
		BasePath: "/user/api/v1",
	})

	var doc map[string]interface{}
	if err := json.Unmarshal(patched, &doc); err != nil {
		t.Fatalf("unmarshal swagger2 doc failed: %v", err)
	}
	if doc["host"] != "example.com" {
		t.Fatalf("expected host to be patched, got %v", doc["host"])
	}
	if doc["basePath"] != "/user/api/v1" {
		t.Fatalf("expected basePath to be patched, got %v", doc["basePath"])
	}
	schemes, _ := doc["schemes"].([]interface{})
	if len(schemes) != 1 || schemes[0] != "https" {
		t.Fatalf("expected schemes to contain https, got %v", schemes)
	}
	paths, _ := doc["paths"].(map[string]interface{})
	if _, ok := paths["/mcp/square/list"]; !ok {
		t.Fatalf("expected /mcp/square/list in paths")
	}
}

func TestPatchDocJSONOpenAPI3FoldsBasePathIntoPaths(t *testing.T) {
	raw := []byte(`{
		"openapi":"3.0.3",
		"info":{"title":"demo","version":"v1"},
		"servers":[{"url":"http://localhost:8080/api/v1"}],
		"paths":{
			"/users":{
				"get":{
					"tags":["user"],
					"responses":{"200":{"description":"ok"}}
				}
			}
		}
	}`)

	patched := patchDocJSON(raw, Config{
		OpenAPI3:        true,
		TagNames:        map[string]string{"user": "用户管理"},
		SecuritySchemes: []string{"JWT"},
	}, docServer{
		Scheme:   "https",
		Host:     "example.com",
		BasePath: "/api/v1",
	})

	var doc map[string]interface{}
	if err := json.Unmarshal(patched, &doc); err != nil {
		t.Fatalf("unmarshal openapi3 doc failed: %v", err)
	}

	servers, _ := doc["servers"].([]interface{})
	if len(servers) == 0 {
		t.Fatalf("expected servers after patching")
	}
	server, _ := servers[0].(map[string]interface{})
	if server["url"] != "https://example.com" {
		t.Fatalf("expected server url to be origin-only, got %v", server["url"])
	}

	paths, _ := doc["paths"].(map[string]interface{})
	if _, ok := paths["/api/v1/users"]; !ok {
		t.Fatalf("expected /api/v1/users in paths after folding, got %v", paths)
	}
	if _, ok := paths["/users"]; ok {
		t.Fatalf("original /users should be replaced by /api/v1/users")
	}
}

func TestPatchDocJSONModeIndependence(t *testing.T) {
	raw := []byte(`{
		"swagger":"2.0",
		"info":{"title":"demo","version":"v1"},
		"basePath":"/v1",
		"paths":{"/ping":{"get":{"responses":{"200":{"description":"ok"}}}}}
	}`)

	server := docServer{Scheme: "https", Host: "example.com", BasePath: "/user/api/v1"}

	s2 := patchDocJSON(raw, Config{OpenAPI3: false}, server)
	var s2Doc map[string]interface{}
	if err := json.Unmarshal(s2, &s2Doc); err != nil {
		t.Fatalf("swagger2 unmarshal: %v", err)
	}
	if s2Doc["host"] != "example.com" {
		t.Fatalf("swagger2: expected host = example.com, got %v", s2Doc["host"])
	}
	if s2Doc["basePath"] != "/user/api/v1" {
		t.Fatalf("swagger2: expected basePath = /user/api/v1, got %v", s2Doc["basePath"])
	}

	// Swagger 2 文档在 OAS3 config 下不会被转换，按 Swagger 2 逻辑 patch
	o3 := patchDocJSON(raw, Config{OpenAPI3: true}, server)
	var o3Doc map[string]interface{}
	if err := json.Unmarshal(o3, &o3Doc); err != nil {
		t.Fatalf("openapi3 unmarshal: %v", err)
	}
	if o3Doc["host"] != "example.com" {
		t.Fatalf("swagger2 doc in openapi3 mode: expected host patched, got %v", o3Doc["host"])
	}
}
