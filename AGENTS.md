# 分支规则

本仓库有两条长期主线，地位相同。仓库里没有名为 `master` 的分支，`main` 就是 master。

- `main`：nacos-cli 主线。nacos-cli 的版本号跟随这条主线，当前二进制版本是 `1.0.4`（见 `package.json` 与 `bin/cli.js` 的 `VERSION`）。
- `kp-nacos`：Kailin Project（kp）适配主线，视为第二条 master。kp 版本号独立于 nacos-cli 版本号，当前是 `1.0.0`，定义在 `cmd/kp_version.go` 的 `KPVersion`。

## 合并

从现在起，每一项功能改动都要同时进入 `main` 和 `kp-nacos`。

- 只合并到其中一条主线不算完成。
- 功能分支完成后，分别合并到 `main` 和 `kp-nacos`。两条主线已经分叉时，各自解决冲突，不用其中一条覆盖另一条。
- `kp-nacos` 上已经存在、而 `main` 上还没有的差异留在 `kp-nacos`。这些差异包括 Nacos v1 的 `config-get` / `config-set`，以及 kp 版本号展示。没有明确要求时，不要把这些既有差异回写到 `main`，也不要在同步时从 `kp-nacos` 丢掉。

## 版本与打包

- 打包 kp 适配线时，从 `kp-nacos` 构建。
- 构建写入的 nacos-cli 版本跟随 `main` 的当前版本。
- 详细版本号在 nacos-cli 版本信息之后附上 kp 版本：`<nacos-cli 版本> (commit: ..., built: ...) kp <KPVersion>`。
- `nacos-cli help` 的首行同时显示 nacos-cli 版本号和 kp 版本号。
- kp 线的发布标签使用 `kp-*`，不要使用会触发上游 Release 工作流的 `v*` 标签。现有 Release 工作流会向 `nacos-group/nacos-cli` 和 npm 发布。
