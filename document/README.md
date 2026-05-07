# Sun CLI 使用文档

## 简介

Sun CLI 是一个通用的命令行工具集，提供了多种实用功能，包括时间日期查询、哈希计算、字符串处理、UUID 生成等。

## 安装

### 编译安装

```bash
go build -o sun main.go
```

### 直接使用

将编译后的 `sun` 可执行文件添加到系统 PATH 中，即可在任何位置使用。

## 基本用法

```bash
sun [选项]
```

不带任何参数时，显示帮助信息。

## 命令列表

### 版本信息

| 命令 | 说明 |
|------|------|
| `sun -v, --version` | 显示版本信息 |

**示例：**
```bash
$ sun -v
1.0.0

$ sun --version
1.0.0
```

### 常用工具集

| 命令 | 说明 |
|------|------|
| `sun --verbose` | 显示工具集介绍 |

**示例：**
```bash
$ sun --verbose
我是常用工具集
```

### 时间日期

| 命令 | 说明 |
|------|------|
| `sun --time` | 显示当前时间（格式：HH:MM:SS） |
| `sun --date` | 显示当前日期（格式：YYYY-MM-DD） |
| `sun --datetime` | 显示当前日期和时间（格式：YYYY-MM-DD HH:MM:SS） |
| `sun --timestamp` | 显示当前时间戳（Unix 时间戳） |

**示例：**
```bash
$ sun --time
09:23:17

$ sun --date
2026-04-27

$ sun --datetime
2026-04-27 09:23:17

$ sun --timestamp
1714189397
```

### 随机数与 UUID

| 命令 | 说明 |
|------|------|
| `sun --random <arg>` | 生成指定长度的随机字符串 |
| `sun --uuid` | 生成 UUID |

**示例：**
```bash
$ sun --random 16
W4DzxqiZ3kLm9PqR

$ sun --uuid
550e8400-e29b-41d4-a716-446655440000
```

### 哈希计算

| 命令 | 说明 |
|------|------|
| `sun --md5 <arg>` | 计算字符串的 MD5 值 |
| `sun --sha1 <arg>` | 计算字符串的 SHA1 值 |
| `sun --sha256 <arg>` | 计算字符串的 SHA256 值 |
| `sun --sha512 <arg>` | 计算字符串的 SHA512 值 |

**示例：**
```bash
$ sun --md5 "hello"
5d41402abc4b2a76b9719d911017c592

$ sun --sha256 "hello"
2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
```

### 字符串转换

| 命令 | 说明 |
|------|------|
| `sun --uppercase <arg>` | 将字符串转换为大写 |
| `sun --lowercase <arg>` | 将字符串转换为小写 |

**示例：**
```bash
$ sun --uppercase "hello world"
HELLO WORLD

$ sun --lowercase "HELLO WORLD"
hello world
```

### JSON 格式化

| 命令 | 说明 |
|------|------|
| `sun --jsonformat <arg>` | 格式化 JSON 字符串 |

**示例：**
```bash
$ sun --jsonformat '{"name":"test","value":123}'
{
  "name": "test",
  "value": 123
}
```

### 字符串分割与合并

| 命令 | 说明 |
|------|------|
| `sun --split <arg> --delimiter <arg>` | 使用分隔符分割字符串 |
| `sun --join <arg> --delimiter <arg>` | 使用连接符合并多行字符串 |

**示例：**
```bash
# 分割字符串
$ sun --split "apple,banana;orange" --delimiter ",;"
apple
banana
orange

# 合并字符串
$ sun --join "apple
banana
orange" --delimiter ","
apple,banana,orange
```

### Cron 表达式解析

| 命令 | 说明 |
|------|------|
| `sun --cron <arg>` | 显示 cron 表达式最近 10 次执行时间 |

**示例：**
```bash
$ sun --cron "0 */2 * * *"
2026-04-27 10:00:00
2026-04-27 12:00:00
2026-04-27 14:00:00
...
```

## 完整命令列表

```
Usage of sun:
  --cron <arg>                   显示cron表达式最近10次执行时间
  --date                         显示当前日期
  --datetime                     显示当前日期和时间
  --delimiter <arg>              分隔符/连接符（与--split或--join一起使用）
  --join <arg>                   待合并的多行字符串（需要配合--delimiter使用）
  --jsonformat <arg>             格式化JSON字符串
  --lowercase <arg>              将字符串转换为小写
  --md5 <arg>                    计算字符串的MD5值
  --random <arg>                 生成指定长度的随机字符串
  --sha1 <arg>                   计算字符串的SHA1值
  --sha256 <arg>                 计算字符串的SHA256值
  --sha512 <arg>                 计算字符串的SHA512值
  --split <arg>                  待分割的字符串（需要配合--delimiter使用）
  --time                         显示当前时间
  --timestamp                    显示当前时间戳
  --uppercase <arg>              将字符串转换为大写
  --uuid                         生成UUID
  -v, --version                  显示版本信息
  --verbose                      显示详细信息
```

## 配置文件

Sun CLI 使用 `config.yaml` 配置文件来管理应用信息。

**配置文件示例：**
```yaml
app:
  name: sun-cli
  description: 我的通用 CLI 工具
  version: 1.0.0
```

## 技术栈

- **语言**: Go 1.16+
- **依赖库**:
  - `github.com/robfig/cron/v3` - Cron 表达式解析
  - `github.com/google/uuid` - UUID 生成
  - `gopkg.in/yaml.v3` - YAML 配置文件解析

## 许可证

本项目采用 MIT 许可证。详见 [LICENSE](../LICENSE) 文件。

## 贡献

欢迎提交 Issue 和 Pull Request！

## 更新日志

### v1.0.0
- 初始版本发布
- 支持时间日期查询
- 支持哈希计算（MD5, SHA1, SHA256, SHA512）
- 支持字符串处理（大小写转换、分割、合并）
- 支持 UUID 生成
- 支持随机字符串生成
- 支持 JSON 格式化
- 支持 Cron 表达式解析
