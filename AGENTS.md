# Agent 协作约定

## 与上游 nacos-group/nacos-cli 的关系

本仓库独立开发。与 `nacos-group/nacos-cli` 之间只做被动同步：可以从该仓库拉取更新到本仓库，不要主动改动上游，以免影响其他人的正常开发。

必须遵守：

- 不要向 `nacos-group/nacos-cli` 推送代码、分支、标签或任何其他 ref。
- 不要在 `nacos-group/nacos-cli` 上创建或更新合并请求（Pull Request），也不要在该仓库发表评论、评审、Issue 或其他会打扰上游维护者的写操作。
- 需要对齐上游时，只从 `nacos-group/nacos-cli` fetch / pull 到本仓库，随后只在本仓库内提交、推送和开合并请求。
- 本仓库自己的分支、提交和合并请求只针对当前仓库，不指向 `nacos-group/nacos-cli`。
