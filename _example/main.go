package main

import (
	"log"

	"github.com/gin-gonic/gin"
	ginknife4j "github.com/zhengya-wu/gin-knife4j"
)

func main() {
	r := gin.Default()

	// Swagger 2 模式：使用 swagger.json（Swagger 2 格式）
	ginknife4j.Register(r,
		ginknife4j.WithTitle("示例 API 文档 (Swagger 2)"),
		ginknife4j.WithDocJSONPath("./swagger.json"),
		ginknife4j.WithDefaultBasePath("/api/v1"),
		ginknife4j.WithRoutePrefix("/swagger/v1"),
		ginknife4j.WithTagNames(map[string]string{
			"user":  "用户管理",
			"order": "订单服务",
		}),
		ginknife4j.WithSecuritySchemes("JWT"),
	)

	// OpenAPI 3 模式：使用 openapi3.json（原生 OpenAPI 3 格式）
	ginknife4j.Register(r,
		ginknife4j.WithTitle("示例 API 文档 (OpenAPI 3)"),
		ginknife4j.WithDocJSONPath("./openapi3.json"),
		ginknife4j.WithDefaultBasePath("/api/v1"),
		ginknife4j.WithOpenAPI3(true),
		ginknife4j.WithRoutePrefix("/swagger/v1/openapi3"),
		ginknife4j.WithTagNames(map[string]string{
			"user":  "用户管理",
			"order": "订单服务",
		}),
		ginknife4j.WithSecuritySchemes("JWT"),
	)

	log.Println("Swagger 2:  http://localhost:8080/swagger/v1/doc.html")
	log.Println("OpenAPI 3:  http://localhost:8080/swagger/v1/openapi3/doc.html")

	r.Run(":8080")
}
