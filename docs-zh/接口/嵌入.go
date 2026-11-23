//go:build ignore

// 中文文档副本：Go 工具链拒绝非 ASCII 的包目录路径，故本包不参与构建与测试。
package apidocs

import _ "embed"

// OpenAPI is the public AgentFlow OpenAPI 3.1 contract.
//
//go:embed 开放接口.yaml
var OpenAPI []byte
