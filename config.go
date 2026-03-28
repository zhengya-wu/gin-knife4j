package ginknife4j

import "io/fs"

type Option func(*Config)

type Config struct {
	Title           string
	MainDocLabel    string
	RoutePrefix     string
	OpenAPI3        bool
	DocJSON         []byte
	DocJSONPath     string
	DocJSONEnv      string
	BasePathRules   []BasePathRule
	DefaultBasePath string
	TagNames        map[string]string
	ExtraDocs       []ExtraDoc
	SecuritySchemes []string
	StaticFS        fs.FS
	AllowedOrigins  string
	AllowedMethods  string
	AllowedHeaders  string
}

// BasePathRule 允许根据请求 Host 动态匹配 basePath。
// 当请求 Host 包含 HostContains 时，使用对应的 BasePath。
type BasePathRule struct {
	HostContains string `yaml:"hostContains" json:"hostContains"`
	BasePath     string `yaml:"basePath" json:"basePath"`
}

// ExtraDoc 定义额外的文档源，会在 knife4j 左上角文档切换下拉中显示。
type ExtraDoc struct {
	Label    string `yaml:"label" json:"label"`
	FilePath string `yaml:"filePath" json:"filePath"`
}

func defaultConfig() Config {
	return Config{
		Title:           "API Documentation",
		MainDocLabel:    "主接口文档",
		RoutePrefix:     "/swagger/v1",
		DocJSONEnv:      "SWAGGER_DOC_JSON_PATH",
		DefaultBasePath: "/v1",
		SecuritySchemes: []string{"JWT"},
		AllowedOrigins:  "*",
		AllowedMethods:  "GET, OPTIONS",
		AllowedHeaders:  "Content-Type",
	}
}

// WithTitle 设置文档页面标题。
func WithTitle(title string) Option {
	return func(cfg *Config) {
		cfg.Title = title
	}
}

// WithMainDocLabel 设置主文档在下拉列表中的显示名称。
func WithMainDocLabel(label string) Option {
	return func(cfg *Config) {
		cfg.MainDocLabel = label
	}
}

// WithRoutePrefix 设置 knife4j UI 的路由前缀，例如 "/swagger/v1"。
func WithRoutePrefix(prefix string) Option {
	return func(cfg *Config) {
		cfg.RoutePrefix = prefix
	}
}

// WithOpenAPI3 开启 OpenAPI 3 模式。
// 启用后，源文档必须是原生 OpenAPI 3 格式（含 "openapi" 字段）。
// 不支持传入 Swagger 2 文档自动转换；如需使用 Swagger 2 文档，请使用默认模式。
func WithOpenAPI3(enabled bool) Option {
	return func(cfg *Config) {
		cfg.OpenAPI3 = enabled
	}
}

// WithDocJSON 直接传入 Swagger/OpenAPI 文档的 JSON 字节。
func WithDocJSON(doc []byte) Option {
	return func(cfg *Config) {
		cfg.DocJSON = doc
	}
}

// WithDocJSONPath 指定 Swagger/OpenAPI JSON 文档的文件路径。
func WithDocJSONPath(path string) Option {
	return func(cfg *Config) {
		cfg.DocJSONPath = path
	}
}

// WithDocJSONEnv 指定包含文档文件路径的环境变量名。
func WithDocJSONEnv(env string) Option {
	return func(cfg *Config) {
		cfg.DocJSONEnv = env
	}
}

// WithBasePathRules 设置基于 Host 的 basePath 匹配规则。
// 可用于多环境（开发 / 测试 / 生产）使用不同网关前缀的场景。
func WithBasePathRules(rules ...BasePathRule) Option {
	return func(cfg *Config) {
		cfg.BasePathRules = append([]BasePathRule(nil), rules...)
	}
}

// WithDefaultBasePath 设置默认 basePath（当无 BasePathRule 匹配时使用）。
func WithDefaultBasePath(basePath string) Option {
	return func(cfg *Config) {
		cfg.DefaultBasePath = basePath
	}
}

// WithTagNames 设置 tag 名称的映射，可将英文 tag 显示为中文。
func WithTagNames(names map[string]string) Option {
	return func(cfg *Config) {
		cfg.TagNames = cloneStringMap(names)
	}
}

// WithExtraDocs 添加额外的文档源，在 knife4j UI 的下拉列表中切换。
func WithExtraDocs(docs ...ExtraDoc) Option {
	return func(cfg *Config) {
		cfg.ExtraDocs = append([]ExtraDoc(nil), docs...)
	}
}

// WithSecuritySchemes 设置保留的安全方案名称（过滤掉未列出的）。
func WithSecuritySchemes(schemes ...string) Option {
	return func(cfg *Config) {
		cfg.SecuritySchemes = append([]string(nil), schemes...)
	}
}

// WithStaticFS 自定义 knife4j 前端静态资源。
// 传入的 fs.FS 根目录应包含 doc.html 和 webjars/ 子目录。
func WithStaticFS(fsys fs.FS) Option {
	return func(cfg *Config) {
		cfg.StaticFS = fsys
	}
}

// WithCORS 设置跨域响应头。
func WithCORS(origins, methods, headers string) Option {
	return func(cfg *Config) {
		if origins != "" {
			cfg.AllowedOrigins = origins
		}
		if methods != "" {
			cfg.AllowedMethods = methods
		}
		if headers != "" {
			cfg.AllowedHeaders = headers
		}
	}
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
