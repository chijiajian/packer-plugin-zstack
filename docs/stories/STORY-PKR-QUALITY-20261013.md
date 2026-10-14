# Packer ZStack 质量修复与升级任务

日期：2026-10-13。审计基线：`26eb13b`。本文件记录本次任务、验收证据与剩余限制；旧的 STORY-PKR-BACKLOG.md 是历史需求，不作为本次完成状态。

## 执行原则

- 失败、取消或未生成目标镜像时不能返回 Artifact；不得删除用户提供的源镜像、快照或 VM。
- 分开记录本次创建的 VM、临时导入镜像和目标镜像。默认清理临时资源，失败时清理本次创建的目标镜像；提供显式保留选项。
- SDK 调用必须使用构建 context 和有界超时。取消后的清理使用独立、有界 context。创建前指定资源 UUID，以便请求超时、服务端仍在执行时追踪资源；无法确认的清理必须报告 UUID。
- 单网络、单备份存储的现有配置保持兼容；扩展能力应是可选项。
- 本地测试、HTTP 契约测试与真实云端验收分别记录，不以 mock 测试替代云端验收。不发布版本或修改外部工单。

## 任务清单

| ID | 优先级 | 任务 | 验收条件 | 状态 |
|---|---|---|---|---|
| Q01 | P1 | 失败状态与产物归属 | 所有校验写入 error；halt/cancel 返回 nil Artifact；Artifact 只引用本次目标镜像；回归测试证明源镜像不被删除 | 已完成（验证见下） |
| Q02 | P1 | 取消、API 超时和 VM 清理 | 请求携带 context；创建/等待/导出有期限；取消后清理；destroy/expunge 可恢复；HTTP 延迟响应测试 | 已完成（验证见下） |
| Q03 | P1 | 请求与 URL 日志脱敏 | 禁止 SDK 原始请求 debug；日志不出现 URL 凭证、会话凭证或 userdata；错误也做脱敏 | 已完成（验证见下） |
| Q04 | P2 | 镜像元数据及快照验证 | AddImage 传递 architecture；快照须 Ready/Enabled/Root；无效响应拒绝；停止 VM 后确认 Stopped | 已完成（验证见下） |
| Q05 | P2 | 导出错误与产物元数据 | 权限/服务异常不得按 API 名吞掉；仅明确不支持时降级；导出 URL 通过 State 暴露，远程产物 Files 为空 | 已完成（验证见下） |
| Q06 | P2 | 镜像清理闭环 | 临时导入镜像默认清理；失败目标镜像默认清理；保留选项有效；用户源资源不受影响 | 已完成（验证见下） |
| Q07 | P2 | 配置与资源选择校验 | 必填、来源互斥、认证互斥、端口和 CPU/内存校验；名称查询拒绝重名；环境变量作为默认值 | 已完成（验证见下） |
| Q08 | P2 | 通信与 SSH 密钥 | 生成完整 HCL schema；支持 bastion/WinRM TLS/端口；临时公钥可清理；自有私钥/agent 不被自动密钥覆盖 | 已完成（验证见下） |
| Q09 | P2 | SDK 与 Go 升级 | ZStack SDK v0.0.10；Packer SDK v0.6.12；Go 1.26.9（满足 SDK >=1.26.8）；go.mod/.go-version/CI 一致；检查 go-cty 替代兼容性 | 已完成（验证见下） |
| Q10 | P2 | 测试与 CI 门禁 | race/vet/build/HCL schema/插件检查；新增真实 Driver HTTP 契约及 Builder 流程回归；PR 和发布前执行测试 | 已完成（验证见下） |
| Q11 | P2 | 放置、网络与磁盘配置扩展 | 可选 zone/cluster/host、根盘规格、多个 L3 网络、数据盘规格；请求映射测试；明确仅产出根盘模板及数据盘清理策略 | 已完成（验证见下） |
| Q12 | P2 | 文档与验收 | 中英文说明、迁移注意事项、示例更新；记录本地验证；真实 ZStack Linux/ARM/快照/取消/导出验收单独标记 | 文档完成；剩余 Q12-B/D/E 见验收记录 |

执行顺序：先 Q09（建立统一编译环境），再 Q01–Q08、Q11，最后 Q10、Q12；各任务的验收测试随实现加入。

## 版本与官方依据

- [ZStack SDK v0.0.10](https://github.com/zstackio/zstack-sdk-go-v2/releases/tag/v0.0.10)：修复空 responseKey 时 inventory 解包；StopVmInstance 在 v0.0.4 返回空 inventory 的问题已由本地 HTTP 响应复现。
- [Packer SDK v0.6.12](https://github.com/hashicorp/packer-plugin-sdk/releases/tag/v0.6.12)：go.mod 最低 Go 1.26.8。
- [Packer Builder 失败与取消约定](https://developer.hashicorp.com/packer/docs/plugins/creation/custom-builders)。
- [ZStack Cloud 5.5.30 镜像 API](https://docs.zstack.io/en/zcf/zstack-cloud/5.5.30/dev-guide?section=ZStack_2.1_Dev_Guide_0003&version=5.5.30)。已在 ZStack 5.5.38 / Ceph / x86_64 环境验收；其他组合见下表。

## 审计基线验证

- `go test -race ./...`、`go vet ./...` 通过；builder 覆盖率 67.6%，Builder.Run、collectArtifact、Driver 原为 0%。
- 临时回归测试复现：halt 后返回源镜像 Artifact、取消后 VM 不清理、导出权限错误被吞、导入架构遗漏、导出元数据未进入 State。
- 临时升级 ZStack SDK v0.0.10 后原有测试通过；StopVmInstance 返回 UUID/Stopped 正常。

## 执行与验收记录

### 新发现：SDK HTTP context 未生效

v0.0.10 的 `pkg/util/httputils/httputils.go:requestInternal` 使用 `http.NewRequest`，没有绑定传入的 context。HTTP 延迟响应测试证实请求未及时取消，空响应还可触发 `httpGet` panic。Q02 扩展为：保留 SDK param/view/jsonutils，插件自行管理一个最小 HTTP 层，使用 NewRequestWithContext、每次请求重新签名、可取消异步轮询、禁止跨站重定向；契约测试覆盖这些边界。升级 SDK 本身不能解决此问题。

真实环境只读鉴权已通过；服务端对 Date 时区格式有兼容性差异，CST 格式有效，GMT/UTC 格式报 requestTimeTooSkewed。验收使用 Asia/Shanghai，与本机 SDK 原始行为一致。

### Q06 补充：导出副本的删除顺序

官方 API 单独提供 `DeleteExportedImageFromBackupStorage`，必须在镜像 inventory 仍存在时调用。Artifact.Destroy 和失败产物清理现先删除已成功导出的副本，再 delete/expunge 镜像；重试跳过已经完成的阶段。首轮验收此前已删除镜像 inventory，补调导出删除被服务端拒绝为资源不存在；该轮导出副本是否随镜像 expunge 删除尚无法通过此 API 证明，保留待核查记录（测试镜像 UUID `5389ceda4d92410983de461dbe883ba5`）。

### 本地验证

- Go 1.26.9；ZStack SDK v0.0.10；Packer SDK v0.6.12；Go tool 固定 packer-sdc 到同一 SDK 版本。
- `go test -race -coverprofile=/tmp/packer-quality-cover.out ./...` 通过，builder 覆盖率 **72.7%**（基线 67.6%）；新增 HTTP 截止时间、异步轮询、跨站重定向拒绝、错误脱敏、产物归属、快照校验、清理重试、完整 Builder 流程等测试。可选云端测试默认跳过。
- `make test plugin-check` 通过：race、vet、编译、插件协议检查。实际 Packer 1.9.4 CLI 的 `packer validate` 通过，覆盖多网络、WinRM TLS/端口和新超时字段。
- `go generate ./...` 前后 HCL schema 哈希一致；不再手工维护字段子集。`go mod tidy` 已执行。
- Q08 另外补齐 `communicator_config`、`instance_id` 和实际连接 Host 的 provision hook 元数据，供依赖连接信息的 provisioner 使用。
- PR CI 新增测试、vet、插件检查、格式/module/schema 漂移检查；发布流程在加载签名密钥前执行测试。

### 云端验收（2026-10-13）

使用本机 `.env`，权限 0600；不在此文档记录地址或凭证。日志保存于 Git 忽略的 `.acceptance/`。API 返回版本 **5.5.38**；Ceph、ImageStore 均存在，但本次执行使用 Ceph。全部构建均为新建 `packer-quality-*` 临时资源。

| 场景 | 结果 | 证据或限制 |
|---|---|---|
| x86_64 Ubuntu 源镜像，communicator none | 通过 | 119 秒；VM 创建/运行、Guest Tools、关机、根盘模板、Ready、导出、destroy/expunge；原始镜像仍 Ready |
| 本次临时根盘快照 → 模板 | 通过 | 在本次 VM 上创建临时快照，独立快照构建输出 Ready；删除输出及测试快照 |
| 导出副本 → 镜像删除顺序 | 通过 | 第二轮 147 秒；先 DeleteExportedImageFromBackupStorage，再 delete/expunge 镜像 |
| 取消构建 + 临时数据盘 | 通过 | 在 provision hook 取消；返回 context.Canceled、无产物、VM 已 expunge；数据盘清理无告警，见最终库存核查 |
| 旧源镜像 SSH root 用户 | 未通过 | 网络在 VM 启动后可达，但 root 公钥认证失败；3 分钟超时后 VM 清理完成 |
| 旧源镜像 SSH ubuntu 用户 | 未通过 | 与 root 一样公钥认证失败；GetVmSshKey 已确认服务端收到公钥，不能仅据此断定客户机已应用；超时后 VM 清理完成 |
| ARM | 未执行 | 环境只有 x86_64 主机；architecture 传参与 HCL 有自动回归测试 |
| WinRM、bastion、ImageStore、多网络/放置组合 | 未进行云端验证 | HCL/参数契约测试覆盖；需要对应镜像和基础设施 |

验收可复用 `builder/zstack/acceptance_test.go`：`ZSTACK_ACC_SCENARIO=snapshot` 在本次 VM 上创建根盘快照并验证快照路径；`cancel` 在 provision hook 取消；`ZSTACK_ACC_DATA_DISK_OFFERING_UUID` 可增加临时数据盘。`ZSTACK_ACC_COMMUNICATOR=ssh` 与 `ZSTACK_ACC_SSH_USERNAME` 控制 SSH。每次执行只操作本次创建的资源。

### 与官方能力的剩余差距

本插件定位为根盘镜像 builder，未实现官方所有 API：多备份存储目标、镜像组/多盘模板、共享/标签/权限管理、复杂调度与网络服务配置仍不在此次范围。Windows 和 ARM 的字段可配置，但不能据此宣称云端兼容性已验证。SDK context 缺陷由插件传输层绕开，尚未向上游提交修复。清理请求无法取消已被服务端接受的异步任务，超时 UUID 必须再次核查。

导出删除依据：[官方 API DeleteExportedImageFromBackupStorage](https://docs.zstack.io/en/zcf/zstack-cloud/5.5.30/dev-guide?section=ZStack_2.1_Dev_Guide_0011&version=5.5.30)。

### 收尾与待执行验收任务

- 最终库存核查：本次 `packer-quality-*` VM、镜像、快照及对应根盘查询均为 0；取消测试时间之后的新建数据盘查询无残留；原源镜像仍为 Ready。此结论不涵盖上文首轮导出副本的存储文件。
- 安全核查：`.env` 权限 `0600`，`git check-ignore` 命中，`git ls-files .env` 为空；逐文件比对本次 AK/SK，Git 可见文件匹配数为 0。凭证和验收日志不进入提交；代码通过 PR 提交审核，本次不发布版本。
- **Q12-A（密码与私网自动密钥/清理通过；公共网问题见 Q12-D）**：用户提供的两套镜像已完成非 root/root 密码认证及 SSH provision hook 全流程验收。用户提供的 Ubuntu 镜像最初禁用 cloud-init；按追加指示，通过 SSH 安装 Guest Tools 并修正 cloud-init，已制成临时模板并重验；客户机访问元数据端点得到空响应，密钥认证仍未通过，后续见 Q12-D。旧源镜像的公钥失败记录作为历史证据保留。
- **Q12-B（部分完成，2026-10-14）**：已实测 ImageStore、双/三网卡、默认网卡顺序及 zone/cluster/host 放置；ImageStore 暴露导出残留缺陷 Q12-E。ARM 基础流程已通过；Linux 私网自动密钥及 VM bastion 组合已通过；Windows 结果见文末，不代表全部部署组合通过。
- **Q12-C（已完成，2026-10-14）**：首轮镜像 `5389ceda4d92410983de461dbe883ba5` 的存储侧核查通过：RBD 数据、导出授权对象和 RBD trash 均无残留，证据见下节。后续轮次已验证先删除副本的修复流程。

SSH systemTag 请求格式已与[官方 CreateVmInstance 文档](https://www.zstack.io/docs/4.3.6/html5-en/ZStack_Dev_Guide/topic/ZStack_2.1_Dev_Guide_0039.html)核对：传递公钥明文；userdata 另外进行 base64 编码。

### SSH 补充验收（用户提供基础镜像后）

验收脚本现在支持 `ZSTACK_ACC_IMAGE_URL` / `ZSTACK_ACC_IMAGE_UUID` 二选一、可选 `ZSTACK_ACC_SSH_PASSWORD`，并断言实际 SSH 用户与配置一致。镜像地址、用户名和密码保存在本机 `.env`，不写入仓库。

- **Ubuntu（文件名 Ubuntu20.04）非 root 密码认证：通过**。212 秒，覆盖约 4 GB qcow2 的 URL 导入、SSH 登录、`id -un` / `uname -m` / 系统文件检查、停止 VM、制镜像、导出、导出副本删除、VM/源镜像/产物清理。
- **同一 Ubuntu 镜像的临时密钥认证：镜像前置条件不满足**。通过密码登录本次测试 VM，只读确认 `/etc/cloud/cloud-init.disabled` 存在，`cloud-init status` 为 `disabled`，普通用户与 root 的 authorized_keys 中均无 Packer 临时公钥，实例数据源为 `iid-datasource-none`。因此密码成功不能视为自动密钥注入/清理通过；该轮未修改基础镜像的 cloud-init 设置；后续准备临时副本的验收见下节。
- **root 密码认证镜像：通过**。503 秒，源文件约 9.5 GB，qcow2 虚拟磁盘 60 GiB；URL 导入、实际 root 身份及远程命令、制镜像、导出、导出副本/VM/源镜像/产物清理全部完成。

自动密钥认证要求客户机能够处理云平台的公钥注入，并配置正确的 SSH 用户；禁用 cloud-init 的镜像可使用已有密码或私钥认证。

补充验收收尾：两轮密码构建通过；密钥场景因已确认的 cloud-init 禁用而超时失败，失败后的 VM 与导入镜像已清理。最终 API 查询本次测试 VM、镜像和根盘均为 0。`go test -race ./...`、`go vet ./...`、`git diff --check` 通过；新提供的具体密码与私有镜像 URL 未出现在 Git 可见文件中，`.env` 保持 0600 且不被跟踪。


### Q12-A：SSH 安装 Guest Tools 并准备 cloud-init 镜像（2026-10-13 至 14）

按用户追加要求，参考本机 `qaenv/redis/prepare_vm.sh` 与 `qaenv/logserver/prepare_vm.sh` 中的 cloud-init 和 Guest Tools 安装部分，在本次新建临时 VM 上执行，不运行应用安装、SSH 全局放宽、磁盘填零等无关步骤。

- 实测源镜像文件名虽带 Ubuntu20.04，`/etc/os-release` 为 **Ubuntu 24.04.2 LTS**；已有 cloud-init **24.4**。
- 阻碍首次启动初始化的配置包括 `/etc/cloud/cloud-init.disabled`，以及 `99-installer.cfg` 中的 `datasource_list: [None]` 和再次生成禁用标记的安装器 userdata。
- 挂载 Guest Tools ISO 后执行 `bash ./zs-tools-install.sh --all-accept`；Guest Tools **5.5.38** 已安装，`qemu-guest-agent` 与 `zwatch-vm-agent` 均为 active。该安装脚本本身不安装或启用 cloud-init。
- 临时副本移走 `99-installer.cfg`、移除禁用标记；cloud-init 当前版本由 `DataSourceEc2` 识别 ZStack（实际 platform 为 `zstack`），故使用 `datasource_list: [ Ec2, None ]`。默认用户与测试 SSH 用户一致，并具备测试所需免密 sudo。
- 启用 cloud-init 四个 systemd 服务，执行 `cloud-init clean --logs --machine-id`，卸载光盘、关机，再制成临时 Ready 模板。原始镜像未修改。
- 新增 `ZSTACK_ACC_SCENARIO=key-cleanup`：无 SSH 密码，真实生成/注入密钥，等待 cloud-init 完成，先验证用户 authorized_keys 有本次密钥，再于清理后、制镜像前检查用户/root 两处均无该密钥。该用例本轮未到达公钥清理断言：SSH 密钥登录超时，自动清理测试 VM，未生成产物。

依据：[cloud-init EC2/ZStack 实现](https://github.com/canonical/cloud-init/blob/24.4/cloudinit/sources/DataSourceEc2.py)、[cloud-init clean](https://docs.cloud-init.io/en/latest/reference/cli.html#clean)。


#### 本轮结果与新增 Q12-D

- **安装/启用已验证，自动密钥验收未通过**：Prepared 模板为 Ready；从它启动的 Packer VM 在 323 秒后因 SSH 公钥认证超时失败，无产物，测试 VM 已自动清理。不能将 Guest Tools 安装成功或 cloud-init 启用视为密钥注入成功。
- 启用后的 cloud-init 状态为 `enabled-by-generator`、`degraded done`，元数据重试失败后回退至 `DataSourceNone`。客户机使用 `curl --noproxy '*'` 直连 `http://169.254.169.254/2009-04-04/meta-data/instance-id`，TCP 连接成功但返回 `Empty reply from server`；因此不是仅由客户端代理配置造成。
- 在准备 VM 上独立调用官方 `SetVmSshKey`，清理 cloud-init 缓存并停止/启动后复现相同空响应；本次复核不依赖 Packer 的创建流程。诊断时将该临时 VM 的 Ec2 元数据等待缩短至 30 秒、单次请求 5 秒，只访问 IPv4 端点；这不构成对服务端问题的修复。
- **Q12-D（待云端排查，P2）**：排查当前测试 L3 网络的 Userdata 服务及对应元数据代理/后端，确认为何元数据 HTTP 请求被直接关闭。网络 inventory 已声明启用 Userdata；服务故障的具体根因尚未确认。验收条件：新 VM 首次启动能读取其 instance-id，cloud-init 正常完成，然后 `key-cleanup` 场景通过。需进一步查看承载该服务的主机日志，不能通过扩大 Packer SSH 超时或手工写入公钥来替代验收。
- 本地 `go test -race ./...` 与 `go vet ./...` 通过；云端密钥用例仍为未通过。凭证、远程脚本输出与临时私钥继续只留在 Git 忽略的本机文件中。

本轮收尾（2026-10-14）：准备 VM、导入源镜像、Prepared 模板均已删除并 expunge；`packer-quality-*` VM/镜像库存为 0，关联准备根盘及测试命名卷库存为 0，原有源镜像仍 Ready。`.env` 保持 0600，AK/SK、独有 SSH 密码和私有镜像 URL 在 Git 可见文件中的匹配数为 0。当时 Q12-D 与 Q12-B/C 尚未完成；10 月 14 日的后续核查及更新状态见下节。


### 2026-10-14：补充验收执行任务

- ImageStore 当前为 Enabled/Connected，不属于缺少环境。本轮执行 URL 导入、SSH 密码 provision、根盘制镜像、导出及产物清理。
- 在当前网络和主机中选择兼容组合，验证多网卡、显式默认网卡及放置约束，核对实际 VM inventory，分别记录云端配置与客户机网络验证范围。
- 使用备份存储节点的受控 SSH 访问核查 Q12-C 指定 UUID 的导出文件；只处理已确认归属本次验收的残留。


#### Q12-C 存储侧核查结果

通过受控 SSH 对 Ceph 备份存储做只读核查，未执行额外删除：

- 从该 UUID 的历史 agent 日志确认实际 installPath，避免以猜测的 pool/路径检查。
- 当前环境的 `cephbackupstorage/cephagent.py` 实现直接只读打开 RBD 并流式提供下载；授权保存在同 pool 的 `<imageUUID>-export` RADOS 对象，不创建独立 qcow2 导出文件。该环境的镜像删除实现也会删除授权对象。
- 对历史路径执行 `rbd info` 返回 ENOENT；`rados stat <imageUUID>-export` 返回 ENOENT；`rbd trash ls --format json` 按该 UUID 精确过滤为空。
- 私有证据：`.acceptance/ceph-target-confirm.log`、`original-export-check.log`、`ceph-export-impl.log`。节点凭证仅保留在忽略的 `.env`。

该结论仅针对本环境的指定历史镜像与 agent 实现；插件仍显式调用删除导出 API，以覆盖不同备份存储的语义。


#### 多网卡与放置矩阵

| 组合 | 结果 | 核查范围 |
|---|---|---|
| 两张网卡，管理网为第一张及默认网，指定 zone/cluster/host，Ceph 输出 | 通过，204 秒 | 实际双 NIC/IP、默认网、communicator IP、放置 inventory、密码 SSH 身份及命令；导出 URL 可读，Destroy 后 HTTP 404/410 |
| 两张网卡，私网为第二张及默认网，指定 zone/cluster/host，ImageStore 输出 | 网络/放置通过；导出清理失败，896 秒 | 实际双 NIC/IP、默认网及 communicator IP 正确；制镜像/导出成功，Destroy 后下载仍 HTTP 206，见 Q12-E |
| 三张网卡，管理网位于第三张并设为默认网，另一宿主机，Ceph 输出 | 通过，131 秒 | 三个 NIC/IP、默认网、communicator IP、zone/cluster/host；导出 URL 可读，Destroy 后 HTTP 404/410 |

双网卡 SSH 场景验证管理网的真实连通性；communicator=none 场景验证云端分配与插件 IP 选择。尚未把每张客户机网卡的 DHCP、路由或跨网互通都列为通过。选用已接入同一集群的既有网络，未修改共享网络和路由器。


#### Q12-E：ImageStore 删除导出 API 成功但物理文件残留（P2，已复现）

- ImageStore 全流程 API 验收第一轮返回成功（920 秒），但进一步 SSH 核查发现输出对应的 `export/image-<hash>.raw` 仍存在，不能把 API success 等同于物理删除。
- 当前 5.5.38 的存储 agent 日志显示：export 创建 `image-<hash>.raw`，DeleteExportedImage 及后续镜像删除却检查 `<hash>.qcow2`；原始镜像 manifest 与 FUSE 导出入口已移除，raw 副本保留。
- 插件已按要求在 expunge 前调用删除导出 API。本缺陷定位于当前 ImageStore 后端删除行为；不在插件内引入节点 SSH 或文件系统删除。增加真实下载探针以防验收误判。
- 执行任务：复核第二轮下载 URL，记录删除后的状态；核对本轮 UUID → installPath hash → 实际导出路径后仅清理本轮残留，复查 API 库存和物理文件。后续云端修复需兼容实际导出格式及文件名，并回归链接失效和物理删除。


ImageStore 性能记录：两组 Ceph 根盘到 ImageStore 的制镜像耗时分别约 9 分 06 秒、9 分 50 秒，均超过 `image_ready_timeout` 的默认 5 分钟。本轮显式设为 10 分钟；实际使用应按镜像大小和带宽增加该值及 `api_timeout`，不能将大镜像传输误判为 SDK 不兼容。


Q12-E 复核与收尾：

- 第二轮（输出 `785347c515124338897f2d1b5d5ac25c`）896 秒，新增 HTTP 探针在 Destroy 前读取成功，Destroy 后仍返回 **206**，测试如期判定失败。第一轮输出为 `af224de525704def8ff6e10bafd731e4`。两轮 API inventory 均已删除，但对应 raw 文件仍可下载。
- 根据此前保存的两个 installPath hash、agent export 日志及文件类型/大小/修改时间核对归属，仅删除这两份 `image-<hash>.raw`（各 40 GiB，合计实际占用 80 GiB）。没有修改共享 agent 或其它文件。
- 定点清理后两个下载地址均返回 **404**；export 目录按两个 hash 复查无文件。私有证据：`store-export-detail.private.log`、`imagestore-pre-cleanup-probe.log`、`imagestore-owned-cleanup.log`、`imagestore-post-cleanup-probe.log`、`imagestore-final-files.log`。
- **Q12-E 状态：本轮残留清理完成，云端删除缺陷尚待修复。ImageStore 整体清理验收不能标为通过。** 新探针保留对 HTTP 404/410 的要求，不将 206 或 API success 放宽成成功。

本轮最终验证：`go test -race ./...`、`go vet ./...`、`git diff --check` 通过。最终 `packer-quality` VM、镜像、卷、快照查询均为空，原始源镜像仍 Ready；匹配两次 ImageStore 输出的导出文件为 0。凭证与私有 URL 的 Git 可见文件扫描匹配数为 0；`.env` 为 0600 且受 Git 忽略。Q12-C 已关闭；Q12-B 仅剩未测平台/组合，Q12-D 元数据服务和 Q12-E ImageStore 后端缺陷仍待处理。


### ARM 环境补充验收准备（2026-10-14）

用户指出 qaenv/redis-aarch64/build.pkr.hcl 中的独立 ARM 环境。已将目标连接及认证信息复制到本机忽略的 `.env`（0600），不提交原 HCL 或凭证。管理节点可达，读取版本接口后，认证资源查询明确返回 Access Key 不存在或禁用；qaenv 中未找到同一节点的其它配置。当前尚未创建 ARM 测试资源，需更新该环境可用的认证信息。

验收脚本新增 `ZSTACK_ACC_ARCHITECTURE=aarch64`：显式传递架构，SSH provision 断言客户机 `uname -m` 为 aarch64，并断言输出模板 Architecture。已完成本地 race/vet 检查；尚不能标记 ARM 云端验收通过。此前用户要求保留的 ImageStore 导出复现现场保持不动。


### Q12-E 后续复现与保留现场（2026-10-14）

前述两份文件清理属于历史操作。用户随后明确要求不再手动清理，已重新使用直接 REST API 复现并保留现场：raw 输出镜像、停止的测试 VM、导入的源镜像及导出文件均保留，不再执行手动删除或 GC。删除导出 API 返回成功后，原下载地址仍返回 HTTP 206，inventory 的 exportUrl/MD5 已清空，UI 按 exportUrl 非空筛选时因此看不到它。qcow2 对照组删除后返回 404。

该问题已按用户确认的 P2、Bug 类型、后端 Component 提交至 [ZSTAC-89526](http://jira.zstack.io/browse/ZSTAC-89526)。当前证据将问题定位到 Cloud ImageStore 删除行为；具体 agent 修复仍需后端确认。凭证、完整 URL 和原始日志保留在本机忽略目录，不进入 Git。上述“测试资源查询为空”的结论仅适用于历史轮次，不适用于此次主动保留的复现现场。


### ARM 新环境验收结果（2026-10-14）

用户提供替代 ARM 环境后，使用本机 `.env` 中的专用 ARM 凭证执行验收；目标 ZStack 版本为 5.5.38。选择已有公共网络和 ARM 集群，使用其 LocalStorage 与 ImageStore，未修改共享资源配置。输入为用户提供的 ZCF ARM64 v1.1.2 压缩 qcow2 URL，SSH 用户为 root。

`TestAcceptanceCloud` 实测 **PASS，150.74 秒**，核查：

- URL 导入独立临时源镜像，显式指定 aarch64；ARM VM 正常启动，实际放置符合指定集群。
- 密码 SSH 登录成功，`id -un` 为 root，`uname -m` 为 aarch64，操作系统标识文件可读。
- 关机、根盘制镜像、Ready 等待、ImageStore 导出成功；输出模板 Architecture 为 aarch64。
- 本轮临时 VM 和导入源镜像清理完成，无 cleanup warnings；输出镜像 Destroy 成功，导出 URL 删除前可读、删除后返回 HTTP 404/410。

这关闭了 ARM 基础流程的未验收项。ARM 多网卡、其它主存储组合和生成密钥/cloud-init 场景尚未单独认证。本轮 LocalStorage 路径通过不代表 Q12-E 的 Ceph 根盘 raw 导出问题已修复；ZSTAC-89526 及用户要求保留的旧环境现场保持原状态。私有原始证据为 `.acceptance/cloud-arm.log`，认证信息和日志不提交 Git。

### cloud-init 文档复核与 Windows 验收任务（2026-10-14）

- Q12-D 重新核对用法：ZStack 官方创建说明要求安装 cloud-init 的镜像同时导入 User Data；SSH Key 管理说明也明确平台删除配置不等于客户机 authorized_keys 清空。此前只传 sshkey、未传 User Data 的结果不足以直接定性元数据服务缺陷。准备独立临时模板，保持默认用户与 datasource 配置一致，补充最小 `#cloud-config` User Data 后重验自动密钥及制镜像前清理，不手工注入测试公钥来替代验证。
- Windows/WinRM：使用用户指定的 Windows 10 qcow2，在 x86 环境创建独立测试 VM，核对 Guest Tools、WinRM 监听与认证；只在该临时 VM 内按需准备 WinRM。验证 Administrator 身份、文件传输、远程 PowerShell 命令、制镜像、导出和资源清理。候选密码只保存于本机忽略的 `.env`。
- 既有 Q12-E raw 导出复现现场继续保留。

文档依据：[ZStack 标准创建云主机](https://www.zstack.io/docs/4.5.0/html5/a_print_ZStack_User_Guide/topic/ZStack_User_Guide_0033_2.html)、[SSH Key Management](https://www.zstack.io/docs/4.2.0/html5-en/a_print_ZStack_User_Guide/topic/ZStack_User_Guide_0033_8.html)、[cloud-init ZStack datasource](https://docs.cloud-init.io/en/23.3.3/reference/datasources/zstack.html)、[Packer WinRM communicator](https://developer.hashicorp.com/packer/docs/communicators/winrm)。

#### cloud-init 复核结果

- ZStack 的 `UserdataBuilder` 会将 `sshkey` 转成额外的 `#cloud-config`，包含 `ssh_authorized_keys` 与 `disable_root: false`。因此没有额外传 User Data 不能单独解释先前失败；仍按官方说明补充最小 User Data 做了对照。
- 公共网络对照仍失败（326.24 秒）：相同 cloud-init 24.4 模板、默认用户 zstack、显式 User Data，元数据接口 TCP 可连但无 HTTP 响应，最终 DataSourceNone，公钥认证失败。客户机真实地址/路由与 inventory 一致。宿主机 lighttpd 1.4.67 日志同步记录 `mod_rewrite invalid result (not beginning with '/')`；配置的兜底重写含 `../zstack-default/...`。这将问题进一步缩小到该网络元数据请求的匹配/重写链路；具体规则为何命中兜底仍需 Cloud 侧确认。未修改共享 lighttpd、iptables 或 agent。
- 使用同一 prepared 镜像，在已有私网中重验；将本轮自有准备 VM 增加一张私网 NIC 作为跳板，避免宿主机缺少到私网的路由。该用例 **PASS，75.23 秒**：平台注入生成公钥，真实免密码 SSH、cloud-init 完成、用户 authorized_keys 存在该公钥，执行真实清理后用户/root 均无该公钥，再关机、制镜像、清理产物。未手工写入测试公钥，未使用密码作为目标 VM 的认证回退。
- 自动密钥注入、制镜像前公钥清理以及 SSH bastion 的指定 Linux 组合已验收通过。Q12-D 收窄为公共网络元数据服务异常，不能再将自动密钥能力整体标记为未通过；其它网络、平台和 cloud-init 版本仍需按部署验收。
- 私有证据：`cloud-key-userdata.log`、`cloudinit-guest-address.log`、`cloudinit-host-http.log`、`cloud-key-private-jumpvm.log`。首轮使用宿主机作 bastion 的失败属于宿主机无私网路由，不作为插件缺陷。

#### Windows / HTTPS WinRM 验收结果

- 使用用户提供的 Windows 10 镜像，在 x86 / ZStack 5.5.38 环境导入独立源镜像（约 37.3 GiB），创建 2 CPU / 4 GiB 准备 VM。第二个候选密码通过 Administrator 认证，密码值不写入文档。
- 原镜像没有可用的 WinRM 监听；通过 Guest Tools 只在该临时 VM 中准备 HTTPS WinRM、自签名证书及限定到测试客户端 IP 的防火墙规则，关闭默认 HTTP WinRM 入站规则。首次启动和服务初始化需数分钟，验收连接超时显式使用 10 分钟。
- 首轮 Packer 通过 HTTPS/NTLM 登录、上传文件、PowerShell 核查上传内容并删除、验证 Administrator/64 位操作系统、优雅关机和制镜像成功；最终 Ceph 导出报 `ORG_ZSTACK_STORAGE_CEPH_BACKUP_10001` / `export/addtoken`，整轮 **FAIL，328.02 秒**。不将该失败写成全流程通过。
- 第二轮显式 `skip_export=true`，Packer 构建 **PASS，321.32 秒**：相同 WinRM/传输/远程命令断言、输出 WindowsVirtio/x86_64 元数据、VM 清理及产物 Destroy 均通过，源模板保持 Ready。
- 对本轮自有 Windows prepared 镜像单独执行直接 REST 导出复测成功：下载四字节返回 HTTP 206，删除导出副本后返回 HTTP 404。首次失败的具体原因尚未确认；这记录为导出失败后的独立复测通过，不是同一整轮 Packer 导出重跑通过。部署验收仍应运行带导出探针的完整场景。
- 未执行 Sysprep/generalize；不声明该镜像克隆后的机器 SID、计算机名和应用封装已认证，也不扩展为所有 Windows/WinRM TLS/认证方式都通过。
- 验收脚本增加 WinRM、平台/资源规格、User Data、SSH bastion 参数，以及文件传输和输出平台断言；插件的 API 错误脱敏列表补入 WinRM 密码。本地 race/vet 与 diff 检查通过。
- 私有证据：`windows-prepare.log`、`cloud-windows-winrm.log`、`cloud-windows-winrm-noexport.log`、`windows-export-recheck.log`、`windows-export-probe.log`。

本轮收尾：准备 VM（含其新增私网 NIC）、本轮导入源镜像、prepared 模板及关联根盘均已删除；按本轮 VM/输出 UUID 复查库存为 0。既有原始源镜像保持 Ready，用户要求保留的 Q12-E raw 输出镜像保持 Ready、复现 VM 保持 Stopped，没有执行该现场的删除或 GC。凭证扫描无匹配，`.env` 维持 0600 且受 Git 忽略。

### Q13：CI 等待步骤超时竞态（2026-10-14）

- 现象：提交 `10eb99c` 的 GitHub Actions 在 `TestStepWaitForImageReady_Run/TimeoutHalts` 失败；预期包含 `timeout`，实际为 `context deadline exceeded`。
- 原因：等待镜像/VM 的步骤同时创建 `context.WithTimeout` 和 `time.After`，两个超时信号竞争，返回错误不一致。此前单次本地检查未触发，不能证明该路径稳定。
- 修复：两处均使用 context 的单一截止时间，超时错误附带资源上下文并通过 `%w` 保留 `context.DeadlineExceeded`；主动取消保留 `context.Canceled`。补充错误类型断言。
- 本地验证：相关等待测试开启 race 重复 100 次通过；与 CI 相同的 `make test plugin-check`、格式、`go mod tidy` 和生成 schema 差异检查均通过。云端 CI 结果以修复提交的 Actions 为准。

### Q14：ISO 安装与应答文件支持（2026-10-14，已实现并通过 x86 Linux 验收）

目标：参考 vSphere ISO 与 Proxmox ISO，将安装介质启动、应答文件传递、真实操作系统安装与根盘模板制作串成完整流程。安装阶段禁止依赖 ZStack userdata/sshkey 元数据；安装账户、SSH/WinRM 和临时公钥由用户提供的安装应答文件配置。

实施任务：
- 显式 ISO 构建模式，导入时使用 ISO mediaType，校验已有 ISO、根盘规格、互斥源和必要连接参数；默认镜像构建行为保持兼容。
- 接入 Packer SDK HTTP 服务与辅助 CD 生成（cd_files/cd_content/cd_label）；生成的辅助 ISO 导入自有云资源后挂载，应答文件由用户通过 file/templatefile 编写，插件不替代操作系统安装器解析语义。
- 支持 boot_command/boot_wait，通过 ZStack 返回的 VNC 控制台发送安装参数；支持显式可达 HTTP 地址。安装介质、应答文件介质和硬盘启动顺序分开管理。
- 等待 SSH/WinRM 后验证已安装系统，卸载安装介质并从根盘重新启动、重新连接后才允许配置和制镜像，不能以 VM Running 或 communicator=none 作为安装成功。
- 复用现有临时密钥清理、关机、根盘制镜像、导出和所有权清理；失败/取消也清理本轮 VM、辅助 ISO、自动导入源等资源，不删除用户已有 ISO，不触碰保留的 Q12-E 现场。
- 配置、HTTP/CD 文件传递、API 参数、取消和清理测试；示例与使用限制文档；执行 race/vet/plugin-check/schema 检查。
- 在 x86 真实环境使用标准安装 ISO 和无人值守应答文件完成安装、硬盘启动、SSH 命令/文件传输、根盘模板及产物重启验证，记录资源 UUID、耗时及独立导出探针；Windows/ARM 等未执行组合明确列出，不扩大认证范围。

参考：[vSphere ISO](https://github.com/vmware/packer-plugin-vsphere/blob/main/builder/vsphere/iso/builder.go)、[Proxmox 共享构建步骤](https://github.com/hashicorp/packer-plugin-proxmox/blob/main/builder/proxmox/common/builder.go)。凭证和应答文件中的秘密只保存本机忽略目录，不提交 Git。

Q14 实施进展：新增 `zstack-iso` builder（默认 builder 也可显式 `iso_install=true`），HTTP/CD 应答文件传递、VNC 启动命令、两次安装检查及卸载介质后的硬盘启动已接入。ISO 模式校验并禁止 ZStack userdata/sshkey，且 VM 参数生成层也不会发送这些标签。新增配置、导入/启动 API、已有 ISO 校验、CD 发布/所有权清理及取消测试，提供 Debian 示例。

真实环境准备使用 ZStack 5.5.38 / x86 / Ceph，Debian 13.7.0 amd64 netinst 标准 ISO，SHA256 `a7ef94ac2fb9a7fec454552abd629b7cc9d5155c886165a45649f5ce6167e355` 已与 Debian 官方校验文件比对。首轮发现 SDK 生成的约 900 KiB 辅助盘触发 ZStack 1 MiB 最小镜像限制，已添加 ISO 尾部零填充和回归断言。下一轮读取 Preseed 并完成基础系统安装，但 Debian APT 自动扫描辅助 CD，因其不含包索引而进入交互提示；中止该轮并按精确 UUID 删除本轮 VM、两张 ISO 和根盘。修正 Preseed：由安装器 early hook 暂时限制 APT 光驱自动扫描，late script 用完后移除该配置。没有手工完成安装来替代自动验收。修正后重新执行完整安装与产物启动检查，结果见下文。

#### Q14 最终验收结果

完整自动验收 **PASS，731.30 秒**，私有日志 `.acceptance/iso-cloud-final.log`：

- 标准 Debian ISO URL 导入、独立 20 GiB 根盘、HTTP Preseed 与第二张 CD 的安装收尾脚本均被真实安装器使用；无 ZStack userdata/sshkey 注入，也未手工完成安装步骤。
- 安装完成后 root 密码 SSH，检查安装/CD 使用标记和 ext4 根文件系统；插件关机、卸载安装盘/应答盘、设置 HardDisk 启动，重新连接并再次检查通过。
- 真实 SSH 文件上传、内容读取/删除、root 身份和 x86_64 架构检查通过；关机、根盘模板创建、Ready 等待、Ceph 导出通过，无 cleanup warnings。
- 输出模板另建新 VM，不加载安装介质、不配置 userdata，硬盘启动、安装标记、SSH 身份/架构和文件传输复验通过。导出 URL 删除前可读，Artifact Destroy 后 HTTP 404/410。
- 按精确 UUID 复查本轮构建 VM、产物验证 VM、源 ISO、辅助 ISO、输出模板和安装根盘均不存在，匹配本轮名称的根盘库存为空。
- 构建 VM `8367215d11484df0b0c2afa8d286b507`，根盘 `861ec70391af467f8296b05d23480e90`，输出模板 `620a53f8307b4d3bb6642b1255589266`，产物验证 VM `618108e0cf694f1bb4fdfc5dd138ceb4`，辅助盘 `4d65d0275f8542f997b23b0b1b027a07`。这些均为本轮临时资源，已清理。
- 原始已有源镜像仍 Ready；Q12-E 保留镜像仍 Ready、复现 VM 仍 Stopped，未执行该现场的删除或 GC。

本地 `make test plugin-check`、race/vet、生成 schema、格式和模块图检查通过；插件 describe 包含默认 builder 和 iso。配置/已有 ISO/架构校验、CD 最小尺寸与仅暴露单文件、硬盘启动前卸载顺序、错误短路与取消测试通过。凭证与安装密码 hash 扫描无匹配；私有配置/日志继续忽略。

认证范围为 x86_64、BIOS、Debian 13.7、SSH、Ceph 这一组合。Windows ISO/Autounattend/WinRM、ARM ISO、UEFI、其他安装器及存储组合未做本轮真实认证；已有 Windows qcow2/ARM 基础验收不替代 ISO 安装验收。应答文件由用户维护，插件不解析操作系统专用语义。

示例补充验证：本机 Packer 1.9.4 使用隔离的开发插件目录执行 `packer validate -var-file=<本机私有变量文件> examples/iso` 通过。示例固定 `1.0.1-dev`，避免已有较高版本旧插件遮蔽 ISO builder；`templatefile` 使用相对模板目录的文件名，修复旧版 Packer 重复拼接 `path.root` 的问题。
