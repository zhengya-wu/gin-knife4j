package ginknife4j

import (
	"encoding/json"
	"sort"
	"strings"
)

type docServer struct {
	Scheme   string
	Host     string
	BasePath string
}

func patchDocJSON(raw []byte, cfg Config, server docServer) []byte {
	var doc map[string]interface{}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return raw
	}

	patchBasePath(doc, server)
	patchSecurity(doc, cfg.SecuritySchemes)
	renameTags(doc, cfg.TagNames)

	out, err := json.Marshal(doc)
	if err != nil {
		return raw
	}
	return out
}

func patchBasePath(doc map[string]interface{}, server docServer) {
	if _, ok := doc["openapi"]; ok {
		patchOpenAPI3BasePath(doc, server)
		return
	}
	if strings.TrimSpace(server.BasePath) != "" {
		doc["basePath"] = server.BasePath
	}
	if strings.TrimSpace(server.Host) != "" {
		doc["host"] = server.Host
	}
	if strings.TrimSpace(server.Scheme) != "" {
		doc["schemes"] = []interface{}{server.Scheme}
	}
}

// patchOpenAPI3BasePath 对 OAS3 文档做 Knife4j 兼容处理：
// Knife4j 前端只从 servers[0].url 提取 origin（scheme://host），
// 忽略其中的路径部分。为保证调试请求地址正确，将 basePath 折叠进每个 path 键中，
// servers[0].url 仅保留 origin。
func patchOpenAPI3BasePath(doc map[string]interface{}, server docServer) {
	originURL := buildHostOnlyURL(server)
	if strings.TrimSpace(originURL) != "" {
		doc["servers"] = []interface{}{
			map[string]interface{}{"url": originURL},
		}
	}

	basePath := normalizeBasePath(server.BasePath)
	if basePath == "" {
		return
	}

	// 先提取原文档 servers 中可能包含的路径前缀，需要替换掉
	existingPrefix := extractServerBasePath(doc)

	paths, ok := doc["paths"].(map[string]interface{})
	if !ok || len(paths) == 0 {
		return
	}
	newPaths := make(map[string]interface{}, len(paths))
	for p, spec := range paths {
		// 去除原有的 server path 前缀（如果 path 已包含）
		cleanPath := p
		if existingPrefix != "" && strings.HasPrefix(p, existingPrefix) {
			cleanPath = strings.TrimPrefix(p, existingPrefix)
			if cleanPath == "" {
				cleanPath = "/"
			}
		}
		newPaths[basePath+cleanPath] = spec
	}
	doc["paths"] = newPaths
}

// extractServerBasePath 从 OAS3 文档的 servers[0].url 中提取路径部分。
func extractServerBasePath(doc map[string]interface{}) string {
	servers, _ := doc["servers"].([]interface{})
	if len(servers) == 0 {
		return ""
	}
	first, _ := servers[0].(map[string]interface{})
	if first == nil {
		return ""
	}
	url, _ := first["url"].(string)
	// 去除 scheme://host 部分，保留路径
	if idx := strings.Index(url, "://"); idx >= 0 {
		rest := url[idx+3:]
		if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
			return normalizeBasePath(rest[slashIdx:])
		}
	}
	return ""
}

func patchSecurity(doc map[string]interface{}, keep []string) {
	if len(keep) == 0 {
		return
	}
	delete(doc, "security")
	if _, ok := doc["openapi"]; ok {
		components, _ := doc["components"].(map[string]interface{})
		if components == nil {
			components = make(map[string]interface{})
		}
		components["securitySchemes"] = filterSecurityDefinitions(components["securitySchemes"], keep)
		doc["components"] = components
		return
	}
	doc["securityDefinitions"] = filterSecurityDefinitions(doc["securityDefinitions"], keep)
}

func filterSecurityDefinitions(raw interface{}, keep []string) map[string]interface{} {
	securityDefinitions, _ := raw.(map[string]interface{})
	if securityDefinitions == nil {
		securityDefinitions = make(map[string]interface{})
	}
	if len(keep) == 0 {
		return securityDefinitions
	}
	filtered := make(map[string]interface{}, len(keep))
	for _, name := range keep {
		if definition, ok := securityDefinitions[name]; ok {
			filtered[name] = definition
		}
	}
	return filtered
}

func renameTags(doc map[string]interface{}, tagNames map[string]string) {
	paths, _ := doc["paths"].(map[string]interface{})
	tagMeta := make(map[string]map[string]interface{})
	existingTags, _ := doc["tags"].([]interface{})
	for _, item := range existingTags {
		tag, _ := item.(map[string]interface{})
		if tag == nil {
			continue
		}
		name, _ := tag["name"].(string)
		if name == "" {
			continue
		}
		displayName := resolveTagName(name, tagNames)
		cloned := cloneInterfaceMap(tag)
		cloned["name"] = displayName
		tagMeta[displayName] = cloned
	}

	tagSet := make(map[string]bool)
	for _, methods := range paths {
		methodMap, _ := methods.(map[string]interface{})
		for _, spec := range methodMap {
			specMap, _ := spec.(map[string]interface{})
			tags, _ := specMap["tags"].([]interface{})
			for i, tag := range tags {
				name, _ := tag.(string)
				if name == "" {
					continue
				}
				displayName := resolveTagName(name, tagNames)
				tags[i] = displayName
				tagSet[displayName] = true
			}
		}
	}

	sortedNames := make([]string, 0, len(tagSet))
	for name := range tagSet {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	if len(sortedNames) == 0 {
		return
	}
	tags := make([]interface{}, 0, len(sortedNames))
	for _, name := range sortedNames {
		if meta, ok := tagMeta[name]; ok {
			tags = append(tags, meta)
			continue
		}
		tags = append(tags, map[string]interface{}{"name": name})
	}
	doc["tags"] = tags
}

func resolveTagName(name string, tagNames map[string]string) string {
	if renamed, ok := tagNames[name]; ok && renamed != "" {
		return renamed
	}
	return name
}

func cloneInterfaceMap(src map[string]interface{}) map[string]interface{} {
	if len(src) == 0 {
		return map[string]interface{}{}
	}
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func buildServerURL(server docServer) string {
	basePath := strings.TrimSpace(server.BasePath)
	if basePath == "" {
		return ""
	}
	basePath = normalizeBasePath(basePath)
	host := strings.TrimSpace(server.Host)
	scheme := strings.TrimSpace(server.Scheme)
	if host == "" {
		return basePath
	}
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + host + basePath
}

// buildHostOnlyURL 仅返回 scheme://host，不包含 basePath。
// 用于 Swagger 2 (OAS2) 场景：因为 showUrl 已包含 basePath，
// enableHostText 只需提供 origin 即可避免路径重复。
func buildHostOnlyURL(server docServer) string {
	host := strings.TrimSpace(server.Host)
	if host == "" {
		return ""
	}
	scheme := strings.TrimSpace(server.Scheme)
	if scheme == "" {
		scheme = "http"
	}
	return scheme + "://" + host
}

func normalizeBasePath(basePath string) string {
	trimmed := strings.TrimSpace(basePath)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}
