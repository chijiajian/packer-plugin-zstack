# Packer Plugin ZStack（中文）

`ZStack` 是一个 HashiCorp [Packer](https://www.packer.io) 多组件插件，用于基于 ZStack 云平台的 VM 实例或卷快照构建自定义镜像。完整的功能列表请参见 [docs](docs)。

> English version: see [README.md](README.md)。

## 功能概览

- **两种构建路径**
  - VM 构建：导入/选择源镜像 → 创建 VM → SSH 连接 → 执行 provisioner → 关机 → 创建快照 → 生成镜像 → 导出
  - 快照直接构建（设置 `source_volume_snapshot_uuid`）：跳过 VM 创建、SSH 与 provisioner，直接基于已有卷快照生成镜像模板
- **两种鉴权方式**
  - 账号密码：`account_name` + `account_password`
  - AccessKey（推荐）：`access_key_id` + `access_key_secret`
- **多种镜像源**：已有镜像名 / `image_uuid` 直传 / `source_image_url` 从 HTTP(S) 远程导入
- **网络与规格**：支持名称解析为 UUID，或直接传入 `network_uuid` / `instance_offering_uuid`；可用 `cpu_num` + `memory_size` 替代实例规格
- **可配置超时**：`image_ready_timeout`、`vm_running_timeout`
- **导出兼容**：备份存储不支持镜像导出时降级为告警，不会让整个构建失败
- **敏感信息脱敏**：日志摘要只输出 `*_set=true/false` 而不会泄漏密钥/密码

## 安装

### 使用 `packer init`

从 Packer 1.7 开始可用：

```hcl
packer {
  required_plugins {
    zstack = {
      version = ">= 2.0.0"
      source  = "github.com/zstackio/zstack"
    }
  }
}
```

随后执行 `packer init`。

## 鉴权环境变量

| 变量 | 说明 |
| --- | --- |
| `ZSTACK_HOST` | 管理节点地址（必填） |
| `ZSTACK_PORT` | 管理节点端口（默认 8080） |
| `ZSTACK_ACCOUNT_NAME` / `ZSTACK_ACCOUNT_PASSWORD` | 账号密码方式 |
| `ZSTACK_ACCESS_KEY_ID` / `ZSTACK_ACCESS_KEY_SECRET` | AccessKey 方式（推荐） |

两种方式互斥，只能配置其一。

## 快速开始

### 1. 基于已有镜像构建

```hcl
source "zstack" "example" {
  zstack_host       = "zstack.example.com"
  access_key_id     = env("ZSTACK_ACCESS_KEY_ID")
  access_key_secret = env("ZSTACK_ACCESS_KEY_SECRET")

  source_image           = "ubuntu-22.04"
  network_name           = "default-network"
  instance_offering_name = "medium-vm"
  backup_storage_name    = "local-backup"

  instance_name     = "packer-example"
  image_name        = "packer-example-image"
  image_description = "Built by Packer"

  ssh_username = "root"
  ssh_password = "your-ssh-password"
}

build {
  sources = ["source.zstack.example"]

  provisioner "shell" {
    inline = ["echo hello"]
  }
}
```

### 2. 从远程 URL 导入并构建

```hcl
source "zstack" "from_url" {
  zstack_host      = "zstack.example.com"
  account_name     = env("ZSTACK_ACCOUNT_NAME")
  account_password = env("ZSTACK_ACCOUNT_PASSWORD")

  source_image     = "ubuntu-22.04-from-url"
  source_image_url = "http://mirrors.example.com/ubuntu-22.04.qcow2"
  format           = "qcow2"     # 默认 qcow2
  platform         = "Linux"     # 默认 Linux
  guest_os_type    = "Ubuntu"

  network_name        = "default-network"
  cpu_num             = 2
  memory_size         = 4096
  backup_storage_name = "local-backup"

  instance_name = "packer-from-url"
  image_name    = "packer-from-url-image"
  ssh_username  = "root"
  ssh_password  = "your-ssh-password"
}
```

### 3. UUID 直通模式（跳过名称查询）

```hcl
source "zstack" "uuid_passthrough" {
  zstack_host       = "zstack.example.com"
  access_key_id     = env("ZSTACK_ACCESS_KEY_ID")
  access_key_secret = env("ZSTACK_ACCESS_KEY_SECRET")

  image_uuid             = "xxxxxxxx-image-uuid-xxxxxxxx"
  network_uuid           = "xxxxxxxx-net-uuid-xxxxxxxx"
  instance_offering_uuid = "xxxxxxxx-offer-uuid-xxxxxxxx"
  backup_storage_uuid    = "xxxxxxxx-bs-uuid-xxxxxxxx"

  instance_name = "packer-uuid"
  image_name    = "packer-uuid-image"
  ssh_username  = "root"
  ssh_password  = "your-ssh-password"
}
```

### 4. 从已有卷快照构建（跳过 VM/SSH）

```hcl
source "zstack" "from_snapshot" {
  zstack_host      = "zstack.example.com"
  account_name     = env("ZSTACK_ACCOUNT_NAME")
  account_password = env("ZSTACK_ACCOUNT_PASSWORD")

  source_volume_snapshot_uuid = "snapshot-uuid-here"

  image_name          = "packer-from-snapshot"
  image_description   = "Built from ZStack volume snapshot"
  platform            = "Linux"
  architecture        = "x86_64"
  backup_storage_name = "local-backup"
}

build {
  sources = ["source.zstack.from_snapshot"]
}
```

> 该模式下 Packer 不会创建 VM，也不会建立 SSH 连接、不会执行 provisioner。

## 配置项参考

### 鉴权（必填一组）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `zstack_host` | string | 管理节点 URL，必填 |
| `port` | int | 管理节点端口，默认 8080 |
| `account_name` / `account_password` | string | 账号密码（仅支持平台管理员 `admin`） |
| `access_key_id` / `access_key_secret` | string | AccessKey 方式（推荐） |

### 镜像参数

| 字段 | 说明 |
| --- | --- |
| `source_image` | 源镜像名称 |
| `image_uuid` | 源镜像 UUID（提供后跳过名称查询） |
| `source_image_url` | HTTP/HTTPS 远程镜像 URL |
| `format` | 镜像格式（qcow2、raw 等，默认 qcow2） |
| `platform` | 平台类型（Linux/Windows，默认 Linux） |
| `guest_os_type` | 客户机 OS 类型（Ubuntu、CentOS …） |
| `architecture` | CPU 架构（x86_64、aarch64） |
| `image_name` | 目标镜像名称 |
| `image_description` | 目标镜像描述（默认与名称相同） |
| `source_volume_snapshot_uuid` | 已有卷快照 UUID，启用快照直建模式 |

### 网络与实例

| 字段 | 说明 |
| --- | --- |
| `network_name` / `network_uuid` | L3 网络名或 UUID |
| `instance_offering_name` / `instance_offering_uuid` | 实例规格 |
| `cpu_num` / `memory_size` | 自定义 CPU / 内存（MB），可替代实例规格 |
| `instance_name` | 临时 VM 名称 |

### 备份存储

| 字段 | 说明 |
| --- | --- |
| `backup_storage_name` / `backup_storage_uuid` | 镜像存放的备份存储；**两者至少配置一个** |

### SSH（仅常规构建路径需要）

| 字段 | 说明 |
| --- | --- |
| `ssh_username` | SSH 用户名 |
| `ssh_password` | SSH 密码 |

### 超时

| 字段 | 默认值 | 说明 |
| --- | --- | --- |
| `image_ready_timeout` | `5m` | 等待镜像变为 Ready 状态的最长时间 |
| `vm_running_timeout` | `5m` | 等待 VM 进入 Running 状态的最长时间 |

## 构建步骤说明

**常规构建路径：**

```
StepPreValidate → StepAddImage（仅 source_image_url）→ StepWaitForImageReady
                → StepSourceImageValidate（仅命名/UUID 模式）
                → StepInstanceOfferingValidate（可选）
                → StepCreateSSHKey → StepCreateVMInstance → StepWaitForRunning
                → StepAttachGuestTools → StepConnect → StepProvision
                → StepStopVmInstance → StepCreateImage → StepExpungeVmInstance
                → StepExportImage
```

**快照直建路径：**

```
StepPreValidate → StepCreateImageFromSnapshot → StepWaitForImageReady → StepExportImage
```

## E2E 测试

参考 [`example/local-dev.pkr.hcl`](example/local-dev.pkr.hcl) 与 [`example/load_images.sh`](example/load_images.sh)。

```bash
PACKER_PLUGIN_PATH=$(pwd) packer validate example/local-dev.pkr.hcl
PACKER_PLUGIN_PATH=$(pwd) packer build    example/local-dev.pkr.hcl
```

> **建议**：`source_image_url` 模式下，备份存储优先用 `backup_storage_name`，插件会自动解析为 UUID，免去记忆 UUID。

## 行为约定

- 备份存储参数必填（无论是常规构建还是快照构建），因为镜像通过 “快照 → 模板” 流程产生。
- 备份存储不支持导出时，构建会跳过导出步骤并打印警告，整个构建仍判定为成功。
- 创建临时 SSH 密钥仅在 `ssh_password` 与 `ssh_private_key_file` 都未提供时发生，密钥仅保留在内存中。

## 贡献

- 发现 Bug 或对使用有疑问，请在 GitHub Issue 中反馈。
- 欢迎 PR；功能性改动请先开 Issue 讨论。
