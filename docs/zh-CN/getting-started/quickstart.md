# 快速开始

5 分钟内运行 powa-sentinel。

> **前置条件**：已安装并运行 PoWA。详见 [前置条件](prerequisites.md)。

## 1. 创建最小配置

创建 `config.yaml`：

```yaml
database:
  host: "127.0.0.1"
  port: 5432
  user: "powa_readonly"
  password: "YOUR_PASSWORD"
  dbname: "powa"

schedule:
  cron: "0 0 9 * * 1"

analysis:
  window_duration: "24h"
  comparison_offset: "168h"

rules:
  slow_sql:
    top_n: 10
  regression:
    threshold_percent: 50

notifier:
  type: "console"
```

将 `database` 凭证替换为你的 PoWA 仓库连接信息。本地测试使用 `console`；生产环境可改为 `wecom`（见 [配置](configuration.md)）。

## 2. 使用 Docker 运行

```bash
docker run -d \
  --name powa-sentinel \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config.yaml \
  ghcr.io/nesnilnehc/powa-sentinel:latest
```

或使用 Docker Compose：

```yaml
services:
  powa-sentinel:
    image: ghcr.io/nesnilnehc/powa-sentinel:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/config.yaml
    restart: unless-stopped
```

## 3. 验证

```bash
curl http://localhost:8080/healthz
```

预期：`200 OK`。

下一步：[配置](configuration.md) | [部署](../guides/deployment.md)
