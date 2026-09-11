# mysql-api — MySQL Test Database for AIC Integration Testing

> ⭐ Like this repo? Give a star to the flagship one:
> [![GitHub stars](https://img.shields.io/github/stars/varwof/core?style=social&label=varwof/core)](https://github.com/varwof/core)

> **WARNING: This is a test utility for varwof AIC (Agent Identity Certificate) integration testing. DO NOT use in production. It has no authentication, no TLS, and exposes full database CRUD operations.**

A lightweight HTTP service that wraps a MySQL database with REST API for table/row CRUD operations. Used by `core/scripts/aic-db-mysql-v2.sh` and gateway E2E tests to verify AIC capability-based access control against a real MySQL backend.

## Prerequisites

- Go 1.26+
- MySQL/MariaDB running and accessible
- A database user with full privileges (CREATE/INSERT/UPDATE/DELETE)

## Configuration

### 1. Create MySQL database and user

```sql
CREATE DATABASE IF NOT EXISTS pkitest;
CREATE USER IF NOT EXISTS 'varwof'@'127.0.0.1' IDENTIFIED BY 'varwof-test';
GRANT ALL PRIVILEGES ON pkitest.* TO 'varwof'@'127.0.0.1';
FLUSH PRIVILEGES;
```

### 2. Create config file

Default path: `/etc/varwof/test/mysql/config.json`

```json
{
  "listen": ":9393",
  "dsn": "varwof:varwof-test@tcp(127.0.0.1:3306)/pkitest"
}
```

DSN format: `user:password@tcp(host:port)/database`

### 3. Or use CLI flags (no config file needed)

```bash
go run . --dsn "varwof:varwof-test@tcp(127.0.0.1:3306)/pkitest" --listen ":9393"
```

| Flag | Default | Description |
|------|---------|-------------|
| `--config` | `/etc/varwof/test/mysql/config.json` | Config file path |
| `--listen` | `:9393` | HTTP listen address (overrides config) |
| `--dsn` | *(required)* | MySQL DSN (overrides config) |
| `--reset` | `false` | Drop and recreate all tables on startup |

## Run

```bash
go build -o mysql-api .
./mysql-api --dsn "varwof:varwof-test@tcp(127.0.0.1:3306)/pkitest"
```

On first run, it automatically creates 3 seed tables: `employees`, `products`, `orders` with sample data.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/tables` | List all tables |
| POST | `/api/tables` | Create table |
| DELETE | `/api/tables/{name}` | Drop table |
| POST | `/api/tables/{name}/alter` | Alter table |
| POST | `/api/reset` | Reset all tables |
| GET | `/api/tables/{name}/rows` | List rows (`?where=&order=&limit=&offset=`) |
| POST | `/api/tables/{name}/rows` | Insert row |
| PUT | `/api/tables/{name}/rows/{pk}` | Update row by id |
| DELETE | `/api/tables/{name}/rows/{pk}` | Delete row by id |

## AIC Integration

The `core/scripts/` test suite uses this service as the backend for AIC capability testing:

```bash
# 1. Start mysql-api
MYSQL_DSN="varwof:varwof-test@tcp(127.0.0.1:3306)/pkitest" go run .

# 2. Setup e2e PKI (People CA + AIC certs)
bash core/scripts/setup-e2e-pki.sh

# 3. Run AIC database tests
MYSQL_DSN="varwof:varwof-test@tcp(127.0.0.1:3306)/pkitest" \
  bash core/scripts/aic-db-mysql-v2.sh
```

Environment variables used by `core/scripts/`:

| Variable | Description |
|----------|-------------|
| `MYSQL_DSN` | MySQL DSN for connectivity and gateway E2E tests |
| `MYSQL_CLI_ARGS` | Optional: mysql CLI args (e.g. `--socket=/tmp/mysql.sock -u root`) |
| `E2E_AIC_CERT` | AIC client certificate path (from `setup-e2e-pki.sh`) |
| `E2E_AIC_KEY` | AIC client key path |

## Project Structure

```
demo/
├── main.go              # HTTP server + routing
├── config.go            # Config loader (JSON)
├── db.go                # MySQL connection + schema init + seed data
├── models.go            # Request/response models
├── handler_tables.go    # Table CRUD handlers
├── handler_rows.go      # Row CRUD handlers
├── go.mod               # Module: github.com/varwof/demo/mysql-api
└── README.md
```
