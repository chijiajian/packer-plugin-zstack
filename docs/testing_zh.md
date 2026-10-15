# 开发与真实环境验收

命令从仓库根目录执行；凭证和私有资源配置保存在本机忽略文件中。

本地验证：`make test plugin-check`；重建 HCL schema：`go generate ./...`。真实云端测试需显式设置 `ZSTACK_ACC=1`、`ZSTACK_ACC_IMAGE_UUID`、`ZSTACK_ACC_NETWORK_UUID`、`ZSTACK_ACC_BACKUP_STORAGE_UUID`，然后执行 `go test ./builder/zstack -run '^TestAcceptanceCloud$' -v -count=1 -timeout=35m`。此测试会创建临时资源并清理镜像产物；默认不连接 SSH，设置 `ZSTACK_ACC_COMMUNICATOR=ssh` 与 `ZSTACK_ACC_SSH_USERNAME` 可验证 SSH。环境须具备对应资源权限及可用容量。

URL 导入验收可设置 `ZSTACK_ACC_IMAGE_URL` 替代 `ZSTACK_ACC_IMAGE_UUID`，两者必须且只能选一个。SSH 验收设置 `ZSTACK_ACC_SSH_PASSWORD` 使用密码认证，留空则测试自动生成密钥。镜像地址与凭证保留在被忽略的本机 `.env`；测试会核对实际 SSH 用户、通过 provision hook 执行系统命令，并确认临时导入镜像被清理。

自动生成密钥认证要求客户机能处理云平台的公钥注入（例如启用且正确配置的 cloud-init），并使用对应 SSH 用户。密码登录成功不代表密钥注入可用；cloud-init 禁用的镜像应使用已有密码或私钥认证。

可先用密码登录临时 VM，挂载 Guest Tools ISO，并通过 SSH 执行其中的 `zs-tools-install.sh` 来准备基础镜像。Guest Tools 安装和 cloud-init 配置是两个步骤：还需检查 `cloud-init.disabled` 及将数据源固定为 `None` 的安装器配置。例如 Ubuntu 24.04.2 / cloud-init 24.4 使用 `DataSourceEc2` 识别 ZStack；准备镜像时配置 `datasource_list: [ Ec2, None ]`，默认用户与 `ssh_username` 一致。移走安装器覆盖配置、启用 cloud-init、执行 `cloud-init clean --logs --machine-id` 后关机，制作新模板，再用新 VM 验证。只修改临时副本，保留原始源镜像。参考 [cloud-init 数据源实现](https://github.com/canonical/cloud-init/blob/24.4/cloudinit/sources/DataSourceEc2.py) 和[镜像缓存清理命令](https://docs.cloud-init.io/en/latest/reference/cli.html#clean)。

客户机还必须能够访问云端元数据服务；Guest Tools 安装或 cloud-init 启用本身不能证明密钥注入成功，应在新 VM 上验证首次启动和 cloud-init 正常完成。

设置 `ZSTACK_ACC_SCENARIO=key-cleanup`，启用 SSH 并留空密码，可验证临时密钥注入和删除。测试镜像须具备 cloud-init、Python 3 和免密 sudo。测试等待 cloud-init 完成，先断言 SSH 用户的公钥文件包含临时密钥，再于清理步骤之后、关机和制镜像之前核对该用户与 root 的 `authorized_keys` 均已移除该密钥。

## ISO 安装验收

使用 [ISO 示例](https://github.com/chijiajian/packer-plugin-zstack/tree/main/examples/iso)准备对应系统的私有配置。必须包含安装检查命令及应答文件设置的 SSH/WinRM 凭证。

```sh
ZSTACK_ACC_ISO=1 \
ZSTACK_ACC_ISO_CONFIG_FILE=/path/to/private-config.json \
go test ./builder/zstack -run '^TestAcceptanceISO$' -v -count=1 -timeout=55m
```

测试覆盖系统安装、卸载介质后的硬盘重启、文件传输、制镜像与导出、产物新 VM 启动及资源清理。应使用独立且容量充足的测试环境。已有源镜像保留，本轮导入介质及输出镜像会清理。
