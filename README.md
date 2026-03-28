# gin-knife4j

[![Go Reference](https://pkg.go.dev/badge/github.com/zhengya-wu/gin-knife4j.svg)](https://pkg.go.dev/github.com/zhengya-wu/gin-knife4j)
[![Go Report Card](https://goreportcard.com/badge/github.com/zhengya-wu/gin-knife4j)](https://goreportcard.com/report/github.com/zhengya-wu/gin-knife4j)

[Knife4j](https://doc.xiaominfo.com/) 文档 UI 的 Gin 集成方案。一行代码即可为 Gin 应用添加美观的 Swagger / OpenAPI 3 在线文档，支持调试、多文档切换、Tag 重命名等功能。

## 特性

- **Swagger 2 & OpenAPI 3 双模式**：通过 `WithOpenAPI3(true/false)` 切换，互不干扰
- **动态 BasePath**：支持根据请求 Host 匹配不同的 basePath（多环境网关适配）
- **安全方案过滤**：只保留指定的 SecurityScheme
- **Tag 中文映射**：将英文 Tag 名映射为中文显示
- **多文档支持**：一个实例可挂载多份文档，在 UI 下拉菜单切换
- **零依赖前端**：Knife4j 前端资源通过 `//go:embed` 内嵌，无需额外部署
- **自定义静态资源**：可替换内置前端，使用自己的 Knife4j 版本

## 安装

```bash
go get github.com/zhengya-wu/gin-knife4j
```

## 快速开始

默认以 **Swagger 2 模式** 运行；只有显式传入 `WithOpenAPI3(true)` 时，才会输出 OpenAPI 3 路由和文档结构。

常规场景建议只启用一种模式。只有在迁移、兼容性验证或对照调试时，才建议同时注册 Swagger 2 和 OpenAPI 3 两套入口。

### Swagger 2 模式

```go
package main

import (
	"github.com/gin-gonic/gin"
	ginknife4j "github.com/zhengya-wu/gin-knife4j"
)

func main() {
	r := gin.Default()

	ginknife4j.Register(r,
		ginknife4j.WithTitle("我的 API 文档"),
		ginknife4j.WithDocJSONPath("./swagger.json"),
		ginknife4j.WithDefaultBasePath("/api/v1"),
	)

	r.Run(":8080")
}
```

启动后访问 http://localhost:8080/swagger/v1/doc.html

### OpenAPI 3 模式

当你希望 UI 以 OpenAPI 3 方式工作时，需要**显式**开启 `WithOpenAPI3(true)`，并传入**原生 OpenAPI 3 格式**的文档：

> **注意**：OpenAPI 3 模式不支持传入 Swagger 2 文档自动转换。如果你只有 Swagger 2 文档，请使用默认模式，或先通过 [swagger-converter](https://converter.swagger.io/) 等工具手动转换。

```go
ginknife4j.Register(r,
	ginknife4j.WithTitle("我的 API 文档"),
	ginknife4j.WithDocJSONPath("./openapi3.json"), // 必须是原生 OpenAPI 3 文档
	ginknife4j.WithDefaultBasePath("/api/v1"),
	ginknife4j.WithOpenAPI3(true),
	ginknife4j.WithRoutePrefix("/swagger/v1/openapi3"),
)
```

启动后访问 http://localhost:8080/swagger/v1/openapi3/doc.html

### 迁移场景：同时启用两种模式

```go
// Swagger 2
ginknife4j.Register(r,
	ginknife4j.WithTitle("API 文档 (Swagger 2)"),
	ginknife4j.WithDocJSONPath("./swagger.json"),
	ginknife4j.WithDefaultBasePath("/api/v1"),
	ginknife4j.WithRoutePrefix("/swagger/v1"),
)

// OpenAPI 3（使用原生 OAS3 文档）
ginknife4j.Register(r,
	ginknife4j.WithTitle("API 文档 (OpenAPI 3)"),
	ginknife4j.WithDocJSONPath("./openapi3.json"),
	ginknife4j.WithDefaultBasePath("/api/v1"),
	ginknife4j.WithOpenAPI3(true),
	ginknife4j.WithRoutePrefix("/swagger/v1/openapi3"),
)
```

> 仅在迁移、兼容性验证或对照调试时推荐这样做。生产环境通常只保留一种模式即可。
>
> 两个实例必须使用不同的 `RoutePrefix`，否则路由会冲突。

## 配置项

| Option | 说明 | 默认值 |
|--------|------|--------|
| `WithTitle(title)` | 文档页面标题 | `"API Documentation"` |
| `WithMainDocLabel(label)` | 主文档在下拉菜单中的显示名 | `"主接口文档"` |
| `WithRoutePrefix(prefix)` | UI 路由前缀 | `"/swagger/v1"` |
| `WithOpenAPI3(enabled)` | 是否启用 OpenAPI 3 模式 | `false` |
| `WithDocJSON(bytes)` | 直接传入文档 JSON 字节 | - |
| `WithDocJSONPath(path)` | 文档 JSON 文件路径 | - |
| `WithDocJSONEnv(env)` | 包含文档路径的环境变量名 | `"SWAGGER_DOC_JSON_PATH"` |
| `WithDefaultBasePath(path)` | 默认 basePath | `"/v1"` |
| `WithBasePathRules(rules...)` | Host 匹配 basePath 规则 | - |
| `WithTagNames(map)` | Tag 名称映射（英文→中文） | - |
| `WithExtraDocs(docs...)` | 额外文档源 | - |
| `WithSecuritySchemes(names...)` | 保留的安全方案 | `["JWT"]` |
| `WithStaticFS(fs)` | 自定义前端静态资源 | 内置 Knife4j |
| `WithCORS(origins, methods, headers)` | CORS 响应头 | `"*", "GET, OPTIONS", "Content-Type"` |

## 文档来源优先级

1. `WithDocJSON(bytes)` — 直接传入字节
2. `WithDocJSONPath(path)` — 从文件读取
3. `WithDocJSONEnv(env)` — 从环境变量指向的文件读取
4. 内置空文档（fallback）

> 建议在示例或生产配置中始终显式提供真实文档源，避免因文件路径错误回退到空文档。

## 动态 BasePath（多环境适配）

当应用部署在不同环境（开发 / 测试 / 生产），网关前缀可能不同。通过 `BasePathRules` 可根据请求 Host 动态匹配：

```go
ginknife4j.Register(r,
	ginknife4j.WithDocJSONPath("./swagger.json"),
	ginknife4j.WithDefaultBasePath("/api/v1"),
	ginknife4j.WithBasePathRules(
		ginknife4j.BasePathRule{HostContains: "dev.", BasePath: "/dev/api/v1"},
		ginknife4j.BasePathRule{HostContains: "staging.", BasePath: "/staging/api/v1"},
	),
)
```

组合使用 `WithDefaultBasePath`、`WithBasePathRules` 和 `WithExtraDocs` 的示例：

```go
ginknife4j.Register(r,
	ginknife4j.WithTitle("聚合接口文档"),
	ginknife4j.WithDocJSONPath("./swagger.json"),
	ginknife4j.WithMainDocLabel("主站接口"),
	ginknife4j.WithDefaultBasePath("/api/v1"),
	ginknife4j.WithBasePathRules(
		ginknife4j.BasePathRule{HostContains: "dev.", BasePath: "/dev/api/v1"},
		ginknife4j.BasePathRule{HostContains: "staging.", BasePath: "/staging/api/v1"},
		ginknife4j.BasePathRule{HostContains: "api.example.com", BasePath: "/prod/api/v1"},
	),
	ginknife4j.WithExtraDocs(
		ginknife4j.ExtraDoc{Label: "后台接口", FilePath: "./swagger-admin.json"},
		ginknife4j.ExtraDoc{Label: "开放平台", FilePath: "./swagger-open.json"},
	),
)
```

效果如下：

- 访问 `dev.example.com` 时，文档调试请求会使用 `/dev/api/v1`
- 访问 `staging.example.com` 时，文档调试请求会使用 `/staging/api/v1`
- 访问 `api.example.com` 时，文档调试请求会使用 `/prod/api/v1`
- 其他 Host 未命中规则时，回退到默认的 `/api/v1`
- Knife4j 左上角文档切换下拉中会额外出现“后台接口”和“开放平台”

## Tag 中文映射

```go
ginknife4j.Register(r,
	ginknife4j.WithDocJSONPath("./swagger.json"),
	ginknife4j.WithTagNames(map[string]string{
		"user":  "用户管理",
		"order": "订单服务",
		"auth":  "认证授权",
	}),
)
```

## 工作原理

- **Swagger 2 模式**：自动设置文档的 `host`、`basePath`、`schemes` 字段
- **OpenAPI 3 模式**：要求源文档为原生 OAS3 格式，自动设置 `servers[0].url`（含完整 basePath）
- **路径修正**：通过 `history.replaceState` 防止 Knife4j 前端将 UI 路由前缀误拼入 API 路径
- **调试支持**：自动注入 `enableHost` 配置，确保 "调试" 按钮发送请求到正确的地址

## 示例

仓库提供 `_example` 可运行示例，内置了 `_example/swagger.json`（Swagger 2）和 `_example/openapi3.json`（原生 OpenAPI 3），分别演示两种模式。

```bash
cd _example
go run .
```

默认访问地址：

- Swagger 2: `http://localhost:8080/swagger/v1/doc.html`
- OpenAPI 3: `http://localhost:8080/swagger/v1/openapi3/doc.html`

## 第三方资源说明

仓库内嵌了 Knife4j 前端静态资源以便开箱即用。相关第三方资源来源与许可证说明见 [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES)。

## Contributing

欢迎提交 Issue 或 Pull Request。贡献说明见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## License

[MIT](LICENSE)
