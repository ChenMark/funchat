# FunChat — 条件解锁式趣味社交 APP

> 你的每一次聊天都有惊喜 🔥

---

## 🚀 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0
- (可选) Docker + Docker Compose

### 一键启动

```bash
# 本地开发
./start.sh

# 或 Docker 部署
docker-compose up -d
```

### 验证

```bash
curl http://localhost:8080/health
# {"code":0,"message":"success","data":{"status":"ok"}}
```

### 停止

```bash
./stop.sh
# 或
docker-compose down
```

---

## 📁 项目结构

```
fun-chat/
├── backend/                  # Go + Gin 后端
│   ├── cmd/server/main.go    # 入口
│   ├── internal/
│   │   ├── handler/          # API 处理器 (auth/friend/message/condition/burn)
│   │   ├── middleware/       # 中间件 (JWT/CORS/日志)
│   │   ├── model/            # 数据模型 (9张表)
│   │   └── repository/       # 数据库层 (GORM)
│   ├── pkg/                  # 工具包 (JWT/SMS/响应/校验)
│   ├── migrations/           # SQL 迁移脚本
│   └── .env.example          # 环境变量模板
├── ios/                      # iOS SwiftUI 客户端
│   └── FunChat/
│       ├── Services/         # 网络层
│       └── Views/            # 页面 (Auth/Chat/Friends)
├── android/                  # Android Compose 客户端
│   └── app/src/main/java/com/funchat/
│       ├── data/api/         # 网络层
│       └── ui/               # 页面 (auth/chat/friends)
├── docker-compose.yml        # Docker 编排
├── Dockerfile                # 后端容器化
├── start.sh / stop.sh        # 启动/停止脚本
└── docs/                     # 设计/测试/报告文档
```

---

## 🔌 API 总览

### 认证 (Phase 1)
| 接口 | 说明 |
|------|------|
| POST /api/v1/auth/send-code | 发送验证码 |
| POST /api/v1/auth/verify-code | 校验验证码 |
| POST /api/v1/auth/register | 注册 |
| POST /api/v1/auth/login | 登录 |
| POST /api/v1/auth/wechat/login | 微信登录 |
| POST /api/v1/auth/apple/login | Apple 登录 |
| POST /api/v1/auth/refresh | Token 刷新 |

### 好友 (Phase 1)
| 接口 | 说明 |
|------|------|
| GET /api/v1/friends/search | 搜索用户 |
| POST /api/v1/friends/request | 发送申请 |
| GET /api/v1/friends/requests | 申请列表 |
| POST /api/v1/friends/accept | 同意 |
| POST /api/v1/friends/reject | 拒绝 |
| GET /api/v1/friends | 好友列表 |
| DELETE /api/v1/friends/:id | 删除好友 |

### 消息 (Phase 2)
| 接口 | 说明 |
|------|------|
| GET /ws | WebSocket 连接 |
| POST /api/v1/messages/send | 发送消息 |
| GET /api/v1/messages/history | 消息历史 |
| GET /api/v1/conversations | 聊天列表 |

### 条件解锁 (Phase 3)
| 接口 | 说明 |
|------|------|
| POST /api/v1/conditions/set | 设置条件 |
| POST /api/v1/conditions/verify-location | 定位校验 |
| POST /api/v1/conditions/verify-steps | 步数校验 |
| POST /api/v1/conditions/set-quiz | 设置题目 |
| POST /api/v1/conditions/verify-quiz | 答题校验 |

### 阅后即焚 (Phase 4)
| 接口 | 说明 |
|------|------|
| POST /api/v1/messages/send-burn | 发送焚毁消息 |
| POST /api/v1/messages/:id/burn-read | 读取(开始倒计时) |
| GET /api/v1/messages/:id/burn-status | 焚毁状态 |

---

## ⚙️ 配置

复制 `.env.example` 为 `.env` 并填写：

```bash
# 数据库
DB_HOST=127.0.0.1
DB_USER=funchat
DB_PASSWORD=funchat123
DB_NAME=funchat

# JWT
JWT_SECRET=your-secret-here

# 腾讯云 SMS (可选，未配置时开发模式)
SMS_SECRET_ID=
SMS_SECRET_KEY=
SMS_APP_ID=
SMS_SIGN_NAME=
SMS_TEMPLATE_ID=
```

---

## 🧪 测试

```bash
cd backend
go test ./internal/handler/   # 17 个单元测试
```

---

## 📦 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.21 + Gin + GORM + gorilla/websocket |
| 数据库 | MySQL 8.0 |
| 实时通信 | WebSocket (Hub 模式) |
| 短信 | 腾讯云 SMS |
| 存储 | 腾讯云 COS |
| iOS | SwiftUI + URLSessionWebSocket |
| Android | Jetpack Compose + OkHttp WebSocket |
| 部署 | Docker + Docker Compose |

---

## 📄 文档

- [完整交付报告](FunChat_V1.0_完整交付报告.md)
- [全量测试用例](FunChat_V1.0_全量测试用例.md) (120+)
- [项目规划](FunChat_项目状态盘点与下一步规划.md)
- [设计系统](DES-001_设计系统规格文档.md)
- [登录注册设计稿](DES-002_登录注册页设计稿规格文档.md)
- [聊天页面设计稿](DES-003_聊天列表_聊天详情页设计稿规格文档.md)

---

*FunChat V1.0 — 条件解锁式趣味社交 APP*
