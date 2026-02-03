# Quick Start

Get powa-sentinel running in under 5 minutes.

> **Prerequisite:** A running PoWA installation. See [Prerequisites](prerequisites.md) for details.

## 1. Create minimal config

Create `config.yaml`:

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

Replace `database` credentials with your PoWA repository database connection. Use `console` for local testing; switch to `wecom` for production (see [Configuration](configuration.md)).

## 2. Run with Docker

```bash
docker run -d \
  --name powa-sentinel \
  --restart unless-stopped \
  -p 8080:8080 \
  -v $(pwd)/config.yaml:/config.yaml \
  ghcr.io/nesnilnehc/powa-sentinel:latest
```

Or with Docker Compose:

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

## 3. Verify

```bash
curl http://localhost:8080/healthz
```

Expected: `200 OK`.

Next: [Configuration](configuration.md) | [Deployment](../guides/deployment.md)
