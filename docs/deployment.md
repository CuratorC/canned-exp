# 部署指南

## 1. 克隆项目

```bash
# 克隆 canned-exp 和 gocanned（go.mod 有本地 replace）
git clone <repo-url> gocanned
git clone <repo-url> canned-exp
# 确保 gocanned 和 canned-exp 在同一父目录下：
#   parent/
#   ├── gocanned/
#   └── canned-exp/
```

## 2. 配置环境变量

```bash
cd canned-exp
cp .env .env.backup   # 如需保留本地配置
```

创建 `.env` 文件：

```ini
# 应用
APP_ENV=prod
APP_PORT=3100
TIMEZONE=Asia/Shanghai

# 数据库（默认 SQLite）
DB_DRIVER=sqlite
DB_NAME=storage/experience.db

# Redis
REDIS_HOST=127.0.0.1
REDIS_PORT=6379

# 日志
LOG_LEVEL=info
LOG_TYPE=json

# 向量嵌入（必填）
EMBEDDING_PROVIDER=zhipu
EMBEDDING_API_KEY=<your-api-key>
EMBEDDING_BASE_URL=https://open.bigmodel.cn/api/paas/v4/embeddings
EMBEDDING_MODEL=embedding-3
EMBEDDING_DIMENSIONS=2048

# 认证
AUTH_TOTP_SECRET=<base32-secret>
AUTH_SESSION_TTL=24h
```

## 3. 打包

```bash
# 本地编译（需要 Go 1.26+）
CGO_ENABLED=1 go build -o bin/canned-exp ./main.go

# 交叉编译（目标 Linux amd64，SQLite 需要 CGO）
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC=x86_64-linux-gnu-gcc \
  go build -o bin/canned-exp ./main.go
```

部署所需文件：

```
deploy/
├── canned-exp          # 二进制
├── .env                # 配置
└── storage/            # 数据库目录（自动创建）
```

## 4. 初始化数据库

```bash
./canned-exp migrate status    # 查看迁移状态
./canned-exp migrate run       # 执行所有待运行的迁移
```

迁移文件自动注册，`migrate run` 会创建 `experiences`、`vectors`、`agents`、`personality_keys`、`personalities`、`framework_mappings`、`memories` 等表。

> 注意：`serve` 命令启动时也会自动运行迁移，但推荐先手动执行 `migrate run` 确认数据库就绪。

## 5. 注册为 systemd 服务

创建 `/etc/systemd/system/canned-exp.service`：

```ini
[Unit]
Description=Canned-Exp Experience Server
After=network.target

[Service]
Type=simple
User=canned-exp
Group=canned-exp
WorkingDirectory=/opt/canned-exp
ExecStart=/opt/canned-exp/bin/canned-exp serve
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

# 环境变量文件
EnvironmentFile=/opt/canned-exp/.env

# 日志
StandardOutput=journal
StandardError=journal
SyslogIdentifier=canned-exp

[Install]
WantedBy=multi-user.target
```

```bash
# 创建用户
sudo useradd -r -s /bin/false canned-exp

# 部署文件
sudo mkdir -p /opt/canned-exp/{bin,storage}
sudo cp bin/canned-exp /opt/canned-exp/bin/
sudo cp .env /opt/canned-exp/
sudo chown -R canned-exp:canned-exp /opt/canned-exp

# 启动
sudo systemctl daemon-reload
sudo systemctl enable canned-exp
sudo systemctl start canned-exp

# 查看状态
sudo systemctl status canned-exp
journalctl -u canned-exp -f
```

## 验证

```bash
# 健康检查
curl http://localhost:3100/health-check

# MCP SSE 端点
curl -N http://localhost:3100/sse
```

## 常用运维命令

```bash
sudo systemctl start canned-exp     # 启动
sudo systemctl stop canned-exp      # 停止
sudo systemctl restart canned-exp   # 重启
journalctl -u canned-exp -f         # 查看实时日志
journalctl -u canned-exp --since today  # 今日日志
```
