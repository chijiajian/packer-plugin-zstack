# Packer Plugin ZStack（中文）

`ZStack` 是一个 HashiCorp [Packer](https://www.packer.io) 多组件插件，用于基于 ZStack 云平台的 VM 实例或卷快照构建自定义镜像。完整的功能列表请参见 [docs](docs)。

> English version: see [README.md](README.md)。

## 功能概览

- **两种构建路径**
  - VM 构建：导入/选择源镜像 → 创建 VM → SSH 连接 → 执行 provisioner → 关机 → 从 root volume 生成镜像 → 导出
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
      version = ">= 1.1.0"
      source  = "github.com/chijiajian/zstack"
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

参考 [`examples/basic.pkr.hcl`](examples/basic.pkr.hcl) 与 [`examples/load_images.sh`](examples/load_images.sh)。

```bash
PACKER_PLUGIN_PATH=$(pwd) packer validate examples/basic.pkr.hcl
PACKER_PLUGIN_PATH=$(pwd) packer build    examples/basic.pkr.hcl
```

> **建议**：`source_image_url` 模式下，备份存储优先用 `backup_storage_name`，插件会自动解析为 UUID，免去记忆 UUID。

## 行为约定

- 备份存储参数必填（无论是常规构建还是快照构建），因为生成的镜像会存放在备份存储中，并可能从该备份存储导出。
- 备份存储不支持导出时，构建会跳过导出步骤并打印警告，整个构建仍判定为成功。
- 创建临时 SSH 密钥仅在 `ssh_password` 与 `ssh_private_key_file` 都未提供时发生，密钥仅保留在内存中。

## 贡献

- 发现 Bug 或对使用有疑问，请在 GitHub Issue 中反馈。
- 欢迎 PR；功能性改动请先开 Issue 讨论。

## 质量修复与升级（2026-10）

本次开发版本使用 Go **1.26.9**、ZStack SDK **v0.0.10**、Packer Plugin SDK **v0.6.12**。`go-cty` 替换保留，以兼容 Packer SDK 的 HCL 生成工具。

- 环境变量仅作为默认值，显式 HCL 配置优先；两种认证方式不可混用。将凭证存入本机 `.env`（权限 `0600`），再执行 `set -a; source .env; set +a`。`.env` 与 `.acceptance/` 均被 Git 忽略；请勿使用 `git add -f` 添加凭证。
- 必须指定目标镜像名、备份存储，以及有效的镜像或根盘快照来源。VM 路径还需要实例名、网络和实例规格（或正数 CPU/内存）。名称查询匹配多个资源时必须改用 UUID。
- 失败或取消不再返回源镜像作为产物。只有本次创建且 Ready 的目标镜像可作为 Artifact。远程镜像的 `Files()` 为空；使用 `Id()` 或 `State("image_uuid")`，导出链接读取 `State("image_url")`。
- 临时 VM 默认 destroy + expunge；新建数据盘也会删除。临时 URL 导入镜像默认清理，设 `keep_source_image = true` 可保留；失败产物设 `keep_image_on_error = true` 可保留。已有源镜像与快照始终不删除。清理异常在日志和 `State("cleanup_warnings")` 中报告 UUID；服务端异步创建可能在本地取消后完成，应按 UUID 核查，不能仅按名称删除。
- `api_timeout` 默认 `30m`，`cleanup_timeout` 默认 `5m`。`image_ready_timeout`、`vm_running_timeout` 默认 `5m`，现在也约束对应创建调用；大镜像应显式增加。SDK 底层未传递 context，插件使用可取消 HTTP 请求和异步轮询补齐。
- `skip_export = true` 跳过导出。仅明确不支持导出的存储会自动降级；权限、容量等错误会使构建失败。
- `network_uuids` 可指定多个网络（与 `network_name`/`network_uuid` 互斥），`default_network_uuid` 选择默认网卡。可选 `zone_uuid`、`cluster_uuid`、`host_uuid`、`root_disk_offering_uuid`、`data_disk_offering_uuids`。产物只包含根盘；数据盘内容不会进入模板。
- 完整 HCL communicator schema 支持 SSH bastion、WinRM host/port/TLS。`ssh_clear_authorized_keys = true` 可在制镜像前清理临时公钥；自有私钥和 SSH agent 不会被自动生成密钥覆盖。快照路径只允许 `communicator = "none"`，源快照必须为 Ready、Enabled、Root。

本地验证：`make test plugin-check`；重建 HCL schema：`go generate ./...`。真实云端测试需显式设置 `ZSTACK_ACC=1`、`ZSTACK_ACC_IMAGE_UUID`、`ZSTACK_ACC_NETWORK_UUID`、`ZSTACK_ACC_BACKUP_STORAGE_UUID`，然后执行 `go test ./builder/zstack -run '^TestAcceptanceCloud$' -v -count=1 -timeout=35m`。此测试会创建临时资源并清理镜像产物；默认不连接 SSH，设置 `ZSTACK_ACC_COMMUNICATOR=ssh` 与 `ZSTACK_ACC_SSH_USERNAME` 可验证 SSH。环境须具备对应资源权限及可用容量。

URL 导入验收可设置 `ZSTACK_ACC_IMAGE_URL` 替代 `ZSTACK_ACC_IMAGE_UUID`，两者必须且只能选一个。SSH 验收设置 `ZSTACK_ACC_SSH_PASSWORD` 使用密码认证，留空则测试自动生成密钥。镜像地址与凭证保留在被忽略的本机 `.env`；测试会核对实际 SSH 用户、通过 provision hook 执行系统命令，并确认临时导入镜像被清理。

自动生成密钥认证要求客户机能处理云平台的公钥注入（例如启用且正确配置的 cloud-init），并使用对应 SSH 用户。密码登录成功不代表密钥注入可用；cloud-init 禁用的镜像应使用已有密码或私钥认证。

可先用密码登录临时 VM，挂载 Guest Tools ISO，并通过 SSH 执行其中的 `zs-tools-install.sh` 来准备基础镜像。Guest Tools 安装和 cloud-init 配置是两个步骤：还需检查 `cloud-init.disabled` 及将数据源固定为 `None` 的安装器配置。本次 Ubuntu 24.04.2 / cloud-init 24.4 使用 `DataSourceEc2` 识别 ZStack；准备镜像时配置 `datasource_list: [ Ec2, None ]`，默认用户与 `ssh_username` 一致。移走安装器覆盖配置、启用 cloud-init、执行 `cloud-init clean --logs --machine-id` 后关机，制作新模板，再用新 VM 验证。只修改临时副本，保留原始源镜像。参考 [cloud-init 数据源实现](https://github.com/canonical/cloud-init/blob/24.4/cloudinit/sources/DataSourceEc2.py) 和[镜像缓存清理命令](https://docs.cloud-init.io/en/latest/reference/cli.html#clean)。

客户机还必须能够访问云端元数据服务；Guest Tools 安装或 cloud-init 启用本身不能证明密钥注入成功，应在新 VM 上验证首次启动和 cloud-init 正常完成。

设置 `ZSTACK_ACC_SCENARIO=key-cleanup`，启用 SSH 并留空密码，可验证临时密钥注入和删除。测试镜像须具备 cloud-init、Python 3 和免密 sudo。测试等待 cloud-init 完成，先断言 SSH 用户的公钥文件包含临时密钥，再于清理步骤之后、关机和制镜像之前核对该用户与 root 的 `authorized_keys` 均已移除该密钥。
