package wecom

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hellolib/client-tools/utils"
)

const (
	nonceSize      = 12 // 96 bits
	tagSize        = 16 // 128 bits
	keyringService = "wecom-cli"
	keyringUser    = "encryption-key"
)

// Bot 企业微信机器人凭证
type Bot struct {
	ID         string `json:"id"`
	Secret     string `json:"secret"`
	CreateTime int64  `json:"create_time"`
}

// AppConfig 解析后的配置
type AppConfig struct {
	Bot Bot
}

// ParseCliConfig 从加密文件解析企业微信 CLI 配置
func ParseCliConfig() (*AppConfig, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("获取配置目录失败: %w", err)
	}

	botEncPath := filepath.Join(configDir, "bot.enc")
	encryptedData, err := os.ReadFile(botEncPath)
	if err != nil {
		return nil, fmt.Errorf("读取 bot.enc 失败: %w", err)
	}

	key, err := loadEncryptionKey(configDir)
	if err != nil {
		return nil, fmt.Errorf("加载加密密钥失败: %w", err)
	}

	plaintext, err := decrypt(key, encryptedData)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	var bot Bot
	if err := json.Unmarshal(plaintext, &bot); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %w", err)
	}

	return &AppConfig{Bot: bot}, nil
}

// getConfigDir 获取配置目录
func getConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = os.Getenv("LOCALAPPDATA")
		}
		if appData == "" {
			return "", fmt.Errorf("无法获取 Windows APPDATA 目录")
		}
		return filepath.Join(appData, "wecom"), nil

	case "darwin":
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("无法获取用户主目录: %w", err)
		}
		return filepath.Join(homeDir, ".config", "wecom"), nil

	default: // linux and others
		configHome := os.Getenv("XDG_CONFIG_HOME")
		if configHome == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("无法获取用户主目录: %w", err)
			}
			configHome = filepath.Join(homeDir, ".config")
		}
		return filepath.Join(configHome, "wecom"), nil
	}
}

// loadEncryptionKey 加载加密密钥（优先文件，其次 keyring）
func loadEncryptionKey(configDir string) ([]byte, error) {
	keyPath := filepath.Join(configDir, ".encryption_key")
	if key, err := loadKeyFromFile(keyPath); err == nil {
		return key, nil
	}
	return loadKeyFromKeyring()
}

// loadKeyFromFile 从文件加载密钥
func loadKeyFromFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("Base64 解码失败: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("密钥长度错误: 期望 32 字节, 实际 %d 字节", len(key))
	}

	return key, nil
}

// loadKeyFromKeyring 从系统 keyring 加载密钥
func loadKeyFromKeyring() ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return loadKeyFromMacOSKeychain()
	case "linux":
		return loadKeyFromLinuxKeyutils()
	case "windows":
		return loadKeyFromWindowsCredentialManager()
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

func loadKeyFromMacOSKeychain() ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password", "-s", keyringService, "-w")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("从 macOS Keychain 读取失败: %w", err)
	}
	return decodeKey(string(output))
}

func loadKeyFromLinuxKeyutils() ([]byte, error) {
	cmd := exec.Command("keyctl", "request", "user", keyringService+":"+keyringUser)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("从 Linux Keyutils 请求密钥失败: %w", err)
	}

	keyID := strings.TrimSpace(string(output))
	cmd = exec.Command("keyctl", "print", keyID)
	output, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("读取密钥内容失败: %w", err)
	}

	return decodeKey(string(output))
}

func loadKeyFromWindowsCredentialManager() ([]byte, error) {
	psScript := fmt.Sprintf(`
		$cred = Get-StoredCredential -Target '%s:%s'
		if ($cred) { $cred.GetNetworkCredential().Password }
	`, keyringService, keyringUser)

	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("从 Windows Credential Manager 读取失败: %w", err)
	}

	return decodeKey(string(output))
}

// decodeKey 解码 Base64 密钥
func decodeKey(b64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("Base64 解码失败: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("密钥长度错误: 期望 32 字节, 实际 %d 字节", len(key))
	}
	return key, nil
}

// decrypt AES-256-GCM 解密
func decrypt(key, data []byte) ([]byte, error) {
	if len(data) < nonceSize+tagSize {
		return nil, fmt.Errorf("数据太短: 长度 %d, 最小需要 %d", len(data), nonceSize+tagSize)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建 GCM 模式失败: %w", err)
	}

	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}

	return plaintext, nil
}

// PrepareWecomCLI 确保安装 wecom-cli 并运行初始化命令
func PrepareWecomCLI(ctx context.Context) error {
	return prepareWecomCLIWithRunner(ctx, utils.ExecRunner{})
}

func prepareWecomCLIWithRunner(ctx context.Context, runner utils.CommandRunner) error {
	// 检查 wecom-cli 是否已安装
	if _, err := runner.LookPath("wecom-cli"); err != nil {
		// 检查 npm 是否可用
		if _, npmErr := runner.LookPath("npm"); npmErr != nil {
			return fmt.Errorf("wecom-cli 未找到且 npm 不可用，请先安装 Node.js/npm")
		}
		// 安装 wecom-cli
		if err := runner.Run(ctx, "npm", "install", "-g", "@wecom/cli"); err != nil {
			return fmt.Errorf("安装 @wecom/cli 失败: %w", err)
		}
		// 再次检查
		if _, err := runner.LookPath("wecom-cli"); err != nil {
			return fmt.Errorf("npm 安装后仍未找到 wecom-cli，请检查 PATH")
		}
	}

	// 运行初始化
	if err := runner.Run(ctx, "wecom-cli", "init"); err != nil {
		return fmt.Errorf("wecom-cli init 失败: %w", err)
	}
	return nil
}
