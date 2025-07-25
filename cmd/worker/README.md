# Aresdata Worker - 高可用采集平台

这是一个基于 Go (Kratos 框架) 编写的高可用、多源、多账户数据采集平台。其核心任务是模拟真人的深度浏览行为，从多个具有复杂反爬虫机制的网站（如飞瓜数据）稳定、高效地采集所需数据。

本项目的设计彻底解决了单点故障（单一IP、单一账号）的瓶颈，并构建了一套灵活、可扩展的架构，能够轻松应对未来新增数据源和采集链路的需求。

## 核心架构思想

平台严格遵循“关注点分离”原则，构建了一个职责清晰的指挥链体系。

| 角色       | 代码层                             | 职责                                                       | 比喻        |
  |:---------|:--------------------------------|:---------------------------------------------------------|:----------|
| **总调度员** | `internal/task/`                | 负责最高级别的宏观调度：**“什么时候”**、**“轮到哪个数据源了”**、**“这一批做多少个”**。     | **工厂总调度** |
| **指挥官**  | `internal/fetcher/*_uc.go`      | 负责一个完整业务的编排：接到命令后，知道**“第一步干什么（查库）”**、**“第二步干什么（调用士兵）”**。 | **车间主任**  |
| **士兵**   | `internal/fetcher/*_fetcher.go` | 负责最具体的技术执行：接到命令后，知道**如何操作具体的工具**（启动浏览器、发送HTTP请求）。        | **流水线工人** |
| **兵工厂**  | `internal/fetcher/manager.go`   | 负责根据蓝图（配置）生产和管理所有“士兵”，供“指挥官”调遣。                          | **设备科**   |

## 核心反反爬虫策略

- **配置驱动**：所有的采集行为（数据源、账号、代理、频率）都由 `configs/config.yaml` 文件驱动，无需修改任何代码即可调整采集策略。
- **多源轮换**：`Task` 层会自动在配置文件中定义的所有数据源之间进行轮换，避免在单一数据源上请求过于集中。
- **多账号轮换**：`Fetcher` 层在执行具体任务时，会自动从账号池中获取不同账号的Cookie/Token，模拟多用户操作。
- **拟人化调度**：引入了随机化的批次大小、视频间的短暂休眠、批次间的长时间休眠，以及乱序处理机制，最大程度规避行为检测。
- **深度指纹伪装**：`HeadlessFetcher` 在启动浏览器时，会自动应用随机化的窗口大小、真实的User-Agent，并注入JS脚本隐藏
  `navigator.webdriver` 等自动化特征。

## 如何使用 (Getting Started)

#### 1. 配置 `configs/config.yaml`

这是整个采集平台的“大脑”和“控制面板”。一个典型的配置如下：

  ```yaml
  data:
    # ... 其他数据库等配置 ...
    datasources:
      - name: "feigua_headless_primary"  # 主力无头浏览器数据源
        type: "headless"
        timeout: 60
        # 该数据源的账号池，可以是多个 cookie 文件
        account_pool:
          - "configs/assets/cookies_01.json"
          - "configs/assets/cookies_02.json"
        # 无头浏览器专属配置
        headless:
          user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

      - name: "feigua_http_backup"       # 备用HTTP数据源
        type: "http"
        base_url: "[https://api.feigua.cn](https://api.feigua.cn)"
        timeout: 30
        throttle_min_wait_ms: 1000
        throttle_max_wait_ms: 3000
        # 该数据源的账号池
        account_pool:
          - "configs/assets/cookies_03.json"
  ````

#### 2\. 安装依赖

  ```bash
  go mod tidy
  ```

#### 3\. 生成 `wire` 依赖注入文件

我们在项目中大量使用 `google/wire` 来管理复杂的依赖关系。修改 `provider`后，需要重新生成。

  ```bash
  # 在项目根目录执行
  go generate ./...
  # 或者进入 cmd/worker 目录执行
  # cd cmd/worker && go generate
  ```

#### 4\. 启动 Worker 进程

Worker 进程是所有后台采集任务的执行者。

  ```bash
  go run ./cmd/worker/main.go --conf ./configs
  ```

启动后，Worker 将根据 `config.yaml` 的配置，自动加载并启动所有已注册的后台任务。

## 已实现的后台任务 (Tasks)

你可以通过 `go run ./cmd/worker/main.go --conf ./configs --task.run [task_name]` 来立即执行一个特定任务。

    - **`fetch:video_rank`**

        - **类型**: HTTP
        - **描述**: 定期采集视频带货榜单数据。这是所有下游数据（视频、商品、达人）的入口。

    - **`fetch:video_details_headless`**

        - **类型**: Headless
        - **描述**: 核心的详情采集任务。自动查找需要首次采集的视频，并使用高度拟人化的策略（多源、多账号、随机休眠）进行采集。

    - **`remedy:video_details_headless`**

        - **类型**: Headless
        - **描述**: 数据修复任务。定期巡检，找出因网络波动等原因部分采集失败的视频，并对其进行重新采集，确保数据完整性。

    - **`process:*`**

        - **类型**: ETL
        - **描述**: 数据处理任务。负责将 `source_data` 表中的原始JSON数据进行解析、清洗、转换，并存入最终的业务表（`videos`, `products`, `bloggers` 等）。

## 如何扩展

得益于当前的模块化架构，扩展新的采集能力变得非常简单。

#### 场景：新增一个“商品详情”的采集任务

    1.  **定义接口**：在 `internal/fetcher/fetcher.go` 中，为 `Fetcher` 接口添加新方法，例如 `CaptureProductDetails(...)`。
    2.  **实现士兵**：在 `http_fetcher.go` 或 `headless_fetcher.go` 中，实现 `CaptureProductDetails` 的具体逻辑。
    3.  **创建指挥官**：在 `internal/fetcher/` 下创建 `product_uc.go`，编写 `ProductUsecase`，负责编排获取商品详情和存储的业务流程。
    4.  **注册总调度员**：在 `internal/task/` 下创建 `fetch_product_details.go`，定义新的 `Task`，并将其注册到 `cmd/worker/main.go` 的任务调度器中。
    5.  **更新配置**：如果需要，在 `config.yaml` 中为新任务配置特定的数据源。
