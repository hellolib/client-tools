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

## wecom

`wecom` 包提供了基于 `wecom-cli` 的初始化和配置读取能力，典型使用顺序如下：

1. 先调用 `PrepareWecomCLI`，确保本机已安装并完成 `wecom-cli` 初始化。
2. 再调用 `ParseCliConfig`，读取并解析本地 CLI 配置。

### PrepareWecomCLI

签名：

```go
func PrepareWecomCLI(ctx context.Context) error
```

用途：

- 检查本机是否存在 `wecom-cli`
- 如果未安装，会尝试通过 `npm install -g @wecom/cli` 自动安装
- 安装完成后执行 `wecom-cli init`

说明：

- 这是一个交互式初始化过程，会阻塞当前进程
- `wecom-cli init` 会在终端输出验证信息，需要人工完成登录/授权
- 如果本机没有 `npm`，方法会直接返回错误

示例：

```go
package main

import (
	"context"
	"log"

	"github.com/hellolib/client-tools/app/wecom"
)

func main() {
	if err := wecom.PrepareWecomCLI(context.Background()); err != nil {
		log.Fatal(err)
	}
}
```

适用场景：

- 首次在一台机器上初始化企业微信 CLI 环境
- 需要在程序启动前保证 `wecom-cli` 已安装且配置可用

### ParseCliConfig

签名：

```go
func ParseCliConfig() (*AppConfig, error)
```

用途：

- 读取 `wecom-cli` 已生成的加密配置
- 使用 AES-256-GCM 解密并解析机器人凭证
- 返回 `*AppConfig`

当前返回结构包含：

```go
type AppConfig struct {
	Bot Bot
}

type Bot struct {
	ID         string `json:"id"`
	Secret     string `json:"secret"`
	CreateTime int64  `json:"create_time"`
}
```

配置来源：

- 配置目录根据操作系统不同：
  - macOS: `~/.config/wecom`
  - Linux: `$XDG_CONFIG_HOME/wecom` 或 `~/.config/wecom`
  - Windows: `%APPDATA%\wecom`
- 加密凭证文件：`bot.enc`
- 加密密钥：优先从 `.encryption_key` 文件读取，其次从系统 keyring 读取

示例：

```go
package main

import (
	"context"
	"log"

	"github.com/hellolib/client-tools/app/wecom"
)

func main() {
	if err := wecom.PrepareWecomCLI(context.Background()); err != nil {
		log.Fatal(err)
	}

	cfg, err := wecom.ParseCliConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("bot_id=%s", cfg.Bot.ID)
}
```

注意：

- `ParseCliConfig` 依赖本地已经存在有效的 `wecom-cli` 配置，因此通常应在 `PrepareWecomCLI` 成功后调用
- 如果配置不存在或未完成初始化，会返回配置相关错误
- 密钥支持从文件或系统 keyring（macOS Keychain / Linux Keyutils / Windows Credential Manager）读取
