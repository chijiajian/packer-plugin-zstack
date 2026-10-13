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
| Q12 | P2 | 文档与验收 | 中英文说明、迁移注意事项、示例更新；记录本地验证；真实 ZStack Linux/ARM/快照/取消/导出验收单独标记 | 文档完成；部分云端验收待环境确认 |

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
| SSH root 用户 | 未通过 | 网络在 VM 启动后可达，但 root 公钥认证失败；3 分钟超时后 VM 清理完成 |
| SSH ubuntu 用户 | 未通过 | 与 root 一样公钥认证失败；GetVmSshKey 已确认服务端收到公钥，不能仅据此断定客户机已应用；超时后 VM 清理完成 |
| ARM | 未执行 | 环境只有 x86_64 主机；architecture 传参与 HCL 有自动回归测试 |
| WinRM、bastion、ImageStore、多网络/放置组合 | 未进行云端验证 | HCL/参数契约测试覆盖；需要对应镜像和基础设施 |

验收可复用 `builder/zstack/acceptance_test.go`：`ZSTACK_ACC_SCENARIO=snapshot` 在本次 VM 上创建根盘快照并验证快照路径；`cancel` 在 provision hook 取消；`ZSTACK_ACC_DATA_DISK_OFFERING_UUID` 可增加临时数据盘。`ZSTACK_ACC_COMMUNICATOR=ssh` 与 `ZSTACK_ACC_SSH_USERNAME` 控制 SSH。每次执行只操作本次创建的资源。

### 与官方能力的剩余差距

本插件定位为根盘镜像 builder，未实现官方所有 API：多备份存储目标、镜像组/多盘模板、共享/标签/权限管理、复杂调度与网络服务配置仍不在此次范围。Windows 和 ARM 的字段可配置，但不能据此宣称云端兼容性已验证。SDK context 缺陷由插件传输层绕开，尚未向上游提交修复。清理请求无法取消已被服务端接受的异步任务，超时 UUID 必须再次核查。

导出删除依据：[官方 API DeleteExportedImageFromBackupStorage](https://docs.zstack.io/en/zcf/zstack-cloud/5.5.30/dev-guide?section=ZStack_2.1_Dev_Guide_0011&version=5.5.30)。

### 收尾与待执行验收任务

- 最终库存核查：本次 `packer-quality-*` VM、镜像、快照及对应根盘查询均为 0；取消测试时间之后的新建数据盘查询无残留；原源镜像仍为 Ready。此结论不涵盖上文首轮导出副本的存储文件。
- 安全核查：`.env` 权限 `0600`，`git check-ignore` 命中，`git ls-files .env` 为空；逐文件比对本次 AK/SK，Git 可见文件匹配数为 0。凭证和验收日志不进入提交；代码通过 PR 提交审核，本次不发布版本。
- **Q12-A（待环境输入）**：提供已确认可用的 SSH 用户/认证方式或支持 cloud-init 的源镜像，复验 SSH provisioner 及临时密钥清理。当前 API 已保存公钥，root/ubuntu 均认证失败，原因不能仅靠管理 API 确定。
- **Q12-B（待环境）**：ARM、Windows/WinRM、bastion、ImageStore、多网卡和放置组合的真实验收。当前覆盖为本地 HCL/参数/生命周期回归，不代表全部部署组合通过。
- **Q12-C（待存储核查）**：确认首轮已 expunge 镜像 `5389ceda4d92410983de461dbe883ba5` 的导出副本是否残留；管理 API 因 ImageVO 不存在无法删除。后续轮次已验证先删除副本的修复流程。

SSH systemTag 请求格式已与[官方 CreateVmInstance 文档](https://www.zstack.io/docs/4.3.6/html5-en/ZStack_Dev_Guide/topic/ZStack_2.1_Dev_Guide_0039.html)核对：传递公钥明文；userdata 另外进行 base64 编码。
