# Contributing

感谢你关注 `gin-knife4j`。

## 开发前准备

- 使用 Go 1.21 或更高版本
- 克隆仓库后执行 `go test ./...`
- 如修改了内嵌前端资源，请同时检查 `static/`、`THIRD_PARTY_NOTICES` 和 `README.md`

## 提交建议

- 尽量保持 PR 聚焦单一主题
- 对外行为有变化时，请同步更新 `README.md` 或示例
- 涉及静态资源更新时，请保留上游许可证与 notices

## 提交前检查

```bash
go test ./...
go vet ./...
```
