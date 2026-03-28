package ginknife4j

import (
	"encoding/json"
	"html"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type service struct {
	cfg      Config
	docCache sync.Map
}

type swaggerResource struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	SwaggerVersion string `json:"swaggerVersion"`
	Location       string `json:"location"`
	ContextPath    string `json:"contextPath"`
}

// Register 在 gin.Engine 上注册 knife4j 文档路由。
//
// 使用示例（Swagger 2）：
//
//	ginknife4j.Register(r,
//	    ginknife4j.WithTitle("My API"),
//	    ginknife4j.WithDocJSONPath("./swagger.json"),
//	    ginknife4j.WithDefaultBasePath("/api/v1"),
//	)
//
// 使用示例（OpenAPI 3，源文档必须是原生 OAS3 格式）：
//
//	ginknife4j.Register(r,
//	    ginknife4j.WithTitle("My API"),
//	    ginknife4j.WithDocJSONPath("./openapi3.json"),
//	    ginknife4j.WithDefaultBasePath("/api/v1"),
//	    ginknife4j.WithOpenAPI3(true),
//	    ginknife4j.WithRoutePrefix("/swagger/v1/openapi3"),
//	)
func Register(r *gin.Engine, opts ...Option) {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}
	cfg.RoutePrefix = normalizeRoutePrefix(cfg.RoutePrefix)
	if cfg.StaticFS == nil {
		cfg.StaticFS = defaultStaticFS()
	}

	r.RemoveExtraSlash = true

	svc := &service{cfg: cfg}
	svc.mount(r)
}

func (s *service) mount(r *gin.Engine) {
	if s.cfg.StaticFS != nil {
		if webjarsFS, err := fs.Sub(s.cfg.StaticFS, "webjars"); err == nil {
			r.StaticFS(s.cfg.RoutePrefix+"/webjars", http.FS(webjarsFS))
		}
	}

	r.GET(s.cfg.RoutePrefix, s.redirectToIndex)
	r.GET(s.cfg.RoutePrefix+"/", s.serveDocHTML)
	r.GET(s.cfg.RoutePrefix+"/doc.html", s.serveDocHTML)
	r.GET(s.cfg.RoutePrefix+"/swagger-resources", s.serveSwaggerResources)
	r.GET(s.cfg.RoutePrefix+"/jf-swagger/swagger-resources", s.serveSwaggerResources)
	r.GET(s.cfg.RoutePrefix+"/swagger-resources/configuration/ui", s.serveUIConfig)
	r.GET(s.cfg.RoutePrefix+"/swagger-resources/configuration/security", s.serveSecurityConfig)
	r.GET(s.cfg.RoutePrefix+"/v3/api-docs/swagger-config", s.serveSwaggerConfig)
	r.GET(s.mainDocPath(), s.serveMainDocJSON)
	r.OPTIONS(s.cfg.RoutePrefix, s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/", s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/doc.html", s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/swagger-resources", s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/jf-swagger/swagger-resources", s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/swagger-resources/configuration/ui", s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/swagger-resources/configuration/security", s.noContentOK)
	r.OPTIONS(s.cfg.RoutePrefix+"/v3/api-docs/swagger-config", s.noContentOK)
	r.OPTIONS(s.mainDocPath(), s.noContentOK)

	for i, doc := range s.cfg.ExtraDocs {
		if strings.TrimSpace(doc.FilePath) == "" {
			continue
		}
		docPath := s.extraDocPath(i)
		filePath := doc.FilePath
		r.GET(docPath, func(c *gin.Context) { s.serveFileDocJSON(c, filePath) })
		r.OPTIONS(docPath, s.noContentOK)
	}
}

func (s *service) serveDocHTML(c *gin.Context) {
	s.setCORS(c)
	data, err := fs.ReadFile(s.cfg.StaticFS, "doc.html")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "knife4j doc page not found"})
		return
	}
	pageHTML := strings.Replace(string(data), "<title>Knife4j</title>", "<title>"+html.EscapeString(s.cfg.Title)+" · Knife4j</title>", 1)
	pageHTML = injectBaseHref(pageHTML, s.cfg.RoutePrefix)
	server := s.resolveDocServer(c)
	// 两种模式均使用 origin-only（scheme://host）作为 enableHostText。
	// OAS3 文档的 basePath 已折叠进 paths，servers[0].url 也仅含 origin，
	// 因此 enableHostText 只需提供 origin 即可。
	hostURL := buildHostOnlyURL(server)
	pageHTML = injectHostSettingsScript(pageHTML, hostURL, s.cfg.RoutePrefix)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, pageHTML)
}

func (s *service) redirectToIndex(c *gin.Context) {
	s.setCORS(c)
	c.Redirect(http.StatusTemporaryRedirect, s.cfg.RoutePrefix+"/")
}

func (s *service) serveSwaggerResources(c *gin.Context) {
	s.setCORS(c)
	c.JSON(http.StatusOK, s.swaggerResources())
}

func (s *service) serveSwaggerConfig(c *gin.Context) {
	s.setCORS(c)
	c.JSON(http.StatusOK, gin.H{
		"configUrl":         "v3/api-docs/swagger-config",
		"oauth2RedirectUrl": "",
		"urls":              s.swaggerResources(),
		"validatorUrl":      "",
	})
}

func (s *service) serveUIConfig(c *gin.Context) {
	s.setCORS(c)
	c.JSON(http.StatusOK, gin.H{
		"deepLinking":              true,
		"displayOperationId":       false,
		"defaultModelsExpandDepth": 1,
		"defaultModelExpandDepth":  1,
		"defaultModelRendering":    "example",
		"displayRequestDuration":   true,
		"docExpansion":             "none",
		"filter":                   true,
		"operationsSorter":         "alpha",
		"showExtensions":           true,
		"tagsSorter":               "alpha",
		"supportedSubmitMethods": []string{
			"get", "put", "post", "delete", "options", "head", "patch", "trace",
		},
	})
}

func (s *service) serveSecurityConfig(c *gin.Context) {
	s.setCORS(c)
	c.JSON(http.StatusOK, gin.H{})
}

func (s *service) serveMainDocJSON(c *gin.Context) {
	s.setCORS(c)
	server := s.resolveDocServer(c)
	doc := s.getPatchedDocJSON("main:"+server.cacheKey(), s.getDocJSON(), server)
	if len(doc) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read swagger doc"})
		return
	}
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, string(doc))
}

func (s *service) serveFileDocJSON(c *gin.Context, filePath string) {
	s.setCORS(c)
	server := s.resolveDocServer(c)
	data, err := os.ReadFile(filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "swagger doc not found: " + filePath})
		return
	}
	doc := s.getPatchedDocJSON("file:"+filePath+":"+server.cacheKey(), data, server)
	c.Header("Content-Type", "application/json; charset=utf-8")
	c.String(http.StatusOK, string(doc))
}

func (s *service) noContentOK(c *gin.Context) {
	s.setCORS(c)
	c.Status(http.StatusNoContent)
}

func (s *service) setCORS(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", s.cfg.AllowedOrigins)
	c.Header("Access-Control-Allow-Methods", s.cfg.AllowedMethods)
	c.Header("Access-Control-Allow-Headers", s.cfg.AllowedHeaders)
}

func (s *service) resolveBasePath(host string) string {
	for _, rule := range s.cfg.BasePathRules {
		if rule.HostContains != "" && strings.Contains(host, rule.HostContains) {
			return rule.BasePath
		}
	}
	return s.cfg.DefaultBasePath
}

func (s *service) getPatchedDocJSON(cacheKey string, raw []byte, server docServer) []byte {
	if cached, ok := s.docCache.Load(cacheKey); ok {
		return cached.([]byte)
	}
	patched := patchDocJSON(raw, s.cfg, server)
	s.docCache.Store(cacheKey, patched)
	return patched
}

func (s *service) getDocJSON() []byte {
	if len(s.cfg.DocJSON) > 0 {
		return s.cfg.DocJSON
	}
	if pathValue := strings.TrimSpace(s.cfg.DocJSONPath); pathValue != "" {
		if data, err := os.ReadFile(pathValue); err == nil {
			return data
		} else {
			log.Printf("ginknife4j: failed to read doc json from path %q: %v", pathValue, err)
		}
	}
	if envName := strings.TrimSpace(s.cfg.DocJSONEnv); envName != "" {
		if pathValue := strings.TrimSpace(os.Getenv(envName)); pathValue != "" {
			if data, err := os.ReadFile(pathValue); err == nil {
				return data
			} else {
				log.Printf("ginknife4j: failed to read doc json from env %q path %q: %v", envName, pathValue, err)
			}
		}
	}
	return []byte(`{"swagger":"2.0","info":{"title":"API Documentation","version":"v0.0.1"},"paths":{}}`)
}

func (s *service) resolveDocServer(c *gin.Context) docServer {
	host := resolveRequestHost(c.Request)
	return docServer{
		Scheme:   resolveRequestScheme(c.Request),
		Host:     host,
		BasePath: s.resolveBasePath(host),
	}
}

func (s *service) swaggerResources() []swaggerResource {
	resources := make([]swaggerResource, 0, len(s.cfg.ExtraDocs)+1)
	mainURL := s.mainDocRelativePath()
	resources = append(resources, swaggerResource{
		Name:           s.cfg.MainDocLabel,
		URL:            mainURL,
		SwaggerVersion: s.docVersion(),
		Location:       mainURL,
		ContextPath:    "",
	})
	for i, doc := range s.cfg.ExtraDocs {
		if strings.TrimSpace(doc.FilePath) == "" {
			continue
		}
		docPath := s.extraDocRelativePath(i)
		resources = append(resources, swaggerResource{
			Name:           doc.Label,
			URL:            docPath,
			SwaggerVersion: s.docVersion(),
			Location:       docPath,
			ContextPath:    "",
		})
	}
	return resources
}

func (s *service) mainDocPath() string {
	if s.cfg.OpenAPI3 {
		return s.cfg.RoutePrefix + "/v3/api-docs"
	}
	return s.cfg.RoutePrefix + "/v2/api-docs"
}

func (s *service) mainDocRelativePath() string {
	if s.cfg.OpenAPI3 {
		return "v3/api-docs"
	}
	return "v2/api-docs"
}

func (s *service) docVersion() string {
	if s.cfg.OpenAPI3 {
		return "3.0"
	}
	return "2.0"
}

func (s *service) extraDocPath(index int) string {
	if s.cfg.OpenAPI3 {
		return path.Join(s.cfg.RoutePrefix, "v3/api-docs/extra", strconv.Itoa(index+1))
	}
	return path.Join(s.cfg.RoutePrefix, "v2/api-docs/extra", strconv.Itoa(index+1))
}

func (s *service) extraDocRelativePath(index int) string {
	if s.cfg.OpenAPI3 {
		return path.Join("v3/api-docs/extra", strconv.Itoa(index+1))
	}
	return path.Join("v2/api-docs/extra", strconv.Itoa(index+1))
}

func (s docServer) cacheKey() string {
	return s.Scheme + "|" + s.Host + "|" + s.BasePath
}

func resolveRequestHost(r *http.Request) string {
	if host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); host != "" {
		parts := strings.Split(host, ",")
		return strings.TrimSpace(parts[0])
	}
	if host := strings.TrimSpace(r.Header.Get("X-Original-Host")); host != "" {
		return host
	}
	return strings.TrimSpace(r.Host)
}

func resolveRequestScheme(r *http.Request) string {
	for _, headerName := range []string{"X-Forwarded-Proto", "X-Forwarded-Scheme", "X-Scheme"} {
		if scheme := strings.TrimSpace(r.Header.Get(headerName)); scheme != "" {
			parts := strings.Split(scheme, ",")
			return strings.ToLower(strings.TrimSpace(parts[0]))
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

func injectHostSettingsScript(pageHTML, hostURL, routePrefix string) string {
	var parts []string

	// 通过 history.replaceState 将 URL 固定为 RoutePrefix/，
	// 使 Knife4j 提取页面路径前缀时得到空字符串，避免将 RoutePrefix 拼入 API 路径。
	// Knife4j 的 springdoc 模式会从 pathname 拼出 //v3/api-docs/swagger-config（双斜杠），
	// 这由 Register() 中设置的 RemoveExtraSlash 在路由层消化。
	if routePrefix != "" {
		targetJSON, _ := json.Marshal(routePrefix + "/")
		parts = append(parts,
			`try{history.replaceState(null,"",`+string(targetJSON)+`)}catch(e){}`)
	}

	// 设置 enableHost 让 knife4j 的 "调试" 请求使用正确的 origin
	if hostURL = strings.TrimSpace(hostURL); hostURL != "" {
		hostJSON, err := json.Marshal(hostURL)
		if err == nil {
			parts = append(parts,
				`var key="Knife4jGlobalSettings";var settings={};`+
					`try{var raw=window.localStorage.getItem(key);if(raw){settings=JSON.parse(raw)||{}}}catch(e){}`+
					`settings.enableHost=true;settings.enableHostText=`+string(hostJSON)+`;`+
					`window.localStorage.setItem(key,JSON.stringify(settings))`)
		}
	}

	if len(parts) == 0 {
		return pageHTML
	}
	script := "<script>(function(){" + strings.Join(parts, ";") + "})();</script>"
	if strings.Contains(pageHTML, "<script src=") {
		return strings.Replace(pageHTML, "<script src=", script+"\n  <script src=", 1)
	}
	if strings.Contains(pageHTML, "</body>") {
		return strings.Replace(pageHTML, "</body>", script+"\n</body>", 1)
	}
	return pageHTML + script
}

// injectBaseHref 在 <head> 最前面注入 <base href>，
// 确保所有相对路径的资源引用（link/script）都基于 RoutePrefix/ 解析。
func injectBaseHref(pageHTML, routePrefix string) string {
	baseHref := normalizeRoutePrefix(routePrefix) + "/"
	baseTag := `<base href="` + html.EscapeString(baseHref) + `">`
	if strings.Contains(pageHTML, "<base ") {
		return pageHTML
	}
	const headTag = "<head>"
	if idx := strings.Index(pageHTML, headTag); idx >= 0 {
		insertAt := idx + len(headTag)
		return pageHTML[:insertAt] + "\n  " + baseTag + pageHTML[insertAt:]
	}
	return pageHTML
}

func normalizeRoutePrefix(prefix string) string {
	trimmed := strings.TrimSpace(prefix)
	if trimmed == "" {
		return "/swagger/v1"
	}
	trimmed = strings.TrimRight(trimmed, "/")
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}
