# Client Tools

## lark suite

`larksuite` 包提供了基于 `lark-cli` 的初始化和配置读取能力，典型使用顺序如下：

1. 先调用 `PrepareLarkCLI`，确保本机已安装并完成 `lark-cli` 初始化。
2. 再调用 `ParseCliConfig`，读取并解析本地 CLI 配置。

### PrepareLarkCLI

签名：

```go
func PrepareLarkCLI(ctx context.Context) error
```

用途：

- 检查本机是否存在 `lark-cli`
- 如果未安装，会尝试通过 `npm install -g @larksuite/cli` 自动安装
- 安装完成后执行 `lark-cli config init --new`

说明：

- 这是一个交互式初始化过程，会阻塞当前进程
- `lark-cli config init --new` 会在终端输出二维码或验证信息，需要人工完成登录/授权
- 如果本机没有 `npm`，方法会直接返回错误

示例：

```go
package main

import (
	"context"
	"log"

	"github.com/hellolib/client-tools.git/larksuite"
)

func main() {
	if err := larksuite.PrepareLarkCLI(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

适用场景：

- 首次在一台机器上初始化飞书 CLI 环境
- 需要在程序启动前保证 `lark-cli` 已安装且配置可用

### ParseCliConfig

签名：

```go
func ParseCliConfig() (*core.CliConfig, error)
```

用途：

- 读取 `lark-cli` 已生成的配置
- 解析当前应用的 `AppID`、`AppSecret`、品牌信息和默认用户信息
- 返回 `*core.CliConfig`

当前返回结构包含：

```go
type CliConfig struct {
	AppID      string
	AppSecret  string
	Brand      LarkBrand
	DefaultAs  string
	UserOpenId string
	UserName   string
}
```

配置来源：

- 默认读取 `~/.lark-cli/config.json`
- 如果设置了环境变量 `LARKSUITE_CLI_CONFIG_DIR`，则会从该目录下的 `config.json` 读取
- `AppSecret` 支持直接写入配置，也支持通过文件或系统 keychain 引用

示例：

```go
package main

import (
	"context"
	"log"

	"github.com/hellolib/client-tools.git/larksuite"
)

func main() {
	if err := larksuite.PrepareLarkCLI(context.Background()); err != nil {
		log.Fatal(err)
	}

	cfg, err := larksuite.ParseCliConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("app_id=%s user=%s", cfg.AppID, cfg.UserName)
}
```

注意：

- `ParseCliConfig` 依赖本地已经存在有效的 `lark-cli` 配置，因此通常应在 `PrepareLarkCLI` 成功后调用
- 如果配置不存在或未完成初始化，会返回配置相关错误
- 当前实现默认读取第一个 app 和第一个 user
