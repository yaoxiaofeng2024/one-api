# One-API 从零到精通实战教程

## 教程概述

本教程将带你从零开始，系统性地学习和掌握 One-API 项目。通过 10 个章节的循序渐进学习，你将深入理解这个 LLM API 管理分发系统的架构设计、核心实现、扩展开发和生产部署。

**学习前提：**
- 具备基础的 Go 语言知识
- 了解 HTTP/API 基本概念
- 熟悉 Git 基本操作
- 有 Docker 使用经验更佳

**学习成果：**
- 能够独立理解和修改 One-API 核心代码
- 能够添加新的 AI 模型渠道支持
- 能够进行二次开发和功能扩展
- 能够独立完成生产环境部署和运维

---

## 第一章：项目初识与环境准备

### 1.0 完整依赖清单

在开始之前，请确保你的系统已安装以下所有依赖：

#### 必需依赖

| 依赖 | 版本要求 | 用途 | 下载地址 |
|------|---------|------|----------|
| **Go** | >= 1.20 | 后端开发语言 | https://go.dev/dl/ |
| **TDM-GCC** | 10.3.0 | SQLite CGO 支持 | https://github.com/jmeubank/tdm-gcc/releases |
| **Git** | 任意版本 | 代码克隆 | https://git-scm.com/download/win |
| **Node.js** | >= 16 (LTS) | 前端构建 | https://nodejs.org/ |
| **PowerShell** | 5.0+ | Windows 终端 | 系统自带 |

#### 可选依赖（生产环境推荐）

| 依赖 | 版本 | 用途 | 安装方式 |
|------|------|------|----------|
| **Docker Desktop** | 最新 | 容器化部署 | https://www.docker.com/products/docker-desktop |
| **MySQL** | 8.0 | 生产数据库 | Docker 镜像 |
| **Redis** | 7.x | 缓存系统 | Docker 镜像 |
| **VS Code** | 最新 | 代码编辑器 | https://code.visualstudio.com/ |

#### 快速检查脚本

复制以下脚本到 PowerShell 中运行，自动检查所有依赖：

```powershell
Write-Host "=== One-API 环境检查 ===" -ForegroundColor Cyan
Write-Host ""

# 检查 Go
try {
    $goVersion = go version 2>&1
    Write-Host "✓ Go: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Go: 未安装" -ForegroundColor Red
}

# 检查 GCC
try {
    $gccVersion = gcc --version 2>&1 | Select-Object -First 1
    Write-Host "✓ GCC: $gccVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ GCC: 未安装（SQLite 必需）" -ForegroundColor Red
}

# 检查 Git
try {
    $gitVersion = git --version 2>&1
    Write-Host "✓ Git: $gitVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Git: 未安装" -ForegroundColor Red
}

# 检查 Node.js
try {
    $nodeVersion = node --version 2>&1
    Write-Host "✓ Node.js: $nodeVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Node.js: 未安装" -ForegroundColor Red
}

# 检查 npm
try {
    $npmVersion = npm --version 2>&1
    Write-Host "✓ npm: $npmVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ npm: 未安装" -ForegroundColor Red
}

# 检查 Docker
try {
    $dockerVersion = docker --version 2>&1
    Write-Host "✓ Docker: $dockerVersion" -ForegroundColor Green
} catch {
    Write-Host "✗ Docker: 未安装（可选）" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== 检查完成 ===" -ForegroundColor Cyan
```

**预期输出：**

```
=== One-API 环境检查 ===

✓ Go: go version go1.25.0 windows/amd64
✓ GCC: gcc (tdm64-1) 10.3.0
✓ Git: git version 2.42.0.windows.2
✓ Node.js: v25.2.1
✓ npm: 11.6.2
✓ Docker: Docker version 24.0.7, build afdd53b

=== 检查完成 ===
```

如果看到红色的 ✗，说明该依赖未安装，请按照下面的步骤安装。

---

### 1.1 项目介绍

#### One-API 是什么？

One-API 是一个开源的 LLM（大语言模型）API 管理和分发系统。它的核心价值在于：

1. **统一接口**：将不同厂商的 AI API（OpenAI、Claude、Gemini、文心一言等）统一为标准 OpenAI 格式
2. **密钥管理**：集中管理多个渠道的 API Key，避免分散配置
3. **负载均衡**：智能分配请求到多个渠道，提高可用性
4. **用量统计**：详细记录每个用户、每个模型的调用情况
5. **额度控制**：支持配额管理、令牌过期、IP 限制等功能

#### 核心功能概览

✅ 支持 30+ AI 模型提供商：
- OpenAI ChatGPT 系列（含 Azure）
- Anthropic Claude 系列
- Google Gemini/PaLM 系列
- 百度文心一言
- 阿里通义千问
- 讯飞星火
- 智谱 ChatGLM
- DeepSeek
- Moonshot
- 字节豆包
- ...以及更多

✅ 企业级特性：
- 多用户管理系统
- API Token 管理
- 渠道负载均衡
- 失败自动重试
- 流式响应支持
- 详细的日志和统计

#### 技术栈分析

**后端技术：**
- **Go 1.20+**：高性能后端语言
- **Gin Framework**：轻量级 Web 框架
- **GORM**：ORM 数据库操作库
- **SQLite/MySQL/PostgreSQL**：数据库支持
- **Redis**：缓存系统（可选）

**前端技术：**
- **React 18**：现代前端框架
- **Ant Design**：UI 组件库
- **Axios**：HTTP 客户端

**部署方式：**
- Docker 容器化部署
- 单二进制文件运行
- 支持 Linux/Windows/macOS

#### 应用场景

1. **企业内部 AI 平台**：统一管理多个 AI 供应商，降低采购成本
2. **SaaS 服务集成**：为多个客户提供 AI 能力，按用量计费
3. **开发测试环境**：快速切换不同模型进行对比测试
4. **学术研究**：批量调用 API 进行数据采集和分析
5. **个人开发者**：整合多个免费/低价渠道，降低成本

---

### 1.2 环境搭建

#### 实践任务清单

- [ ] 安装 Go 1.20+ 环境
- [ ] 安装 Node.js 和 npm
- [ ] 安装 Docker（可选）
- [ ] 克隆项目代码
- [ ] 验证环境配置

#### 详细步骤

##### Step 1: 安装 Go 语言环境（Windows）

**前置依赖：安装 TDM-GCC（SQLite 必需）**

⚠️ **重要：** One-API 使用 SQLite 数据库时需要 CGO 支持，必须先安装 GCC 编译器。

**安装 TDM-GCC：**

1. **下载 TDM-GCC**
   - 访问：https://github.com/jmeubank/tdm-gcc/releases/download/v10.3.0-tdm64-1/tdm64-gcc-10.3.0.exe
   - 文件大小：约 50MB
   - 下载时间：1-3 分钟（取决于网络速度）

2. **安装步骤**
   - 双击运行 `tdm64-gcc-10.3.0.exe`
   - 点击 **"Create"** 按钮
   - 保持默认路径：`C:\TDM-GCC-64`
   - 点击 **"Next"**
   - 所有组件保持默认勾选
   - 点击 **"Install"**
   - 等待安装完成（约 2 分钟）
   - 点击 **"Finish"**

3. **验证安装**
   
   **重要：关闭所有 PowerShell 窗口，重新打开一个新的！**
   
   ```powershell
   gcc --version
   # 应该输出：gcc (tdm64-1) 10.3.0
   ```

4. **如果 gcc 命令找不到**
   ```powershell
   # 检查环境变量
   $env:Path -split ';' | Select-String "TDM-GCC"
   
   # 如果没有输出，手动添加：
   [Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\TDM-GCC-64\bin", [EnvironmentVariableTarget]::User)
   
   # 重新打开 PowerShell 后再次验证
   gcc --version
   ```

---

**Go 安装步骤：**

**Windows 用户：**
1. 访问 https://go.dev/dl/
2. 下载 `go1.21.x.windows-amd64.msi` 安装包
3. 双击运行安装程序，使用默认设置即可
   - 默认安装路径：`C:\Go`
   - 会自动配置环境变量
4. 安装完成后，打开 PowerShell 验证：
```powershell
go version
# 输出示例：go version go1.21.6 windows/amd64
```

**如果 `go version` 命令找不到：**
1. 右键"此电脑" → "属性" → "高级系统设置"
2. 点击"环境变量"
3. 在"系统变量"中找到 `Path`
4. 确保包含 `C:\Go\bin`
5. 重新打开 PowerShell 再次尝试

##### Step 2: 安装 Node.js（Windows）

1. 访问 https://nodejs.org/ 
2. 下载 **LTS 版本**（推荐 18.x 或 20.x）的 Windows 安装包（.msi）
3. 双击运行，全部使用默认选项即可
4. 安装完成后，打开 PowerShell 验证：
```powershell
node --version
# 输出示例：v20.10.0

npm --version
# 输出示例：10.2.3
```

**如果命令找不到：**
- 重新打开 PowerShell（确保关闭所有之前的窗口）
- 或者重启电脑

##### Step 3: 安装 Docker Desktop（Windows）

**方式一：使用 WSL2（推荐）**

1. **启用 WSL2**（以管理员身份运行 PowerShell）：
```powershell
wsl --install
```
重启电脑后继续。

2. **下载 Docker Desktop**：
   - 访问 https://www.docker.com/products/docker-desktop
   - 下载 Docker Desktop for Windows
   - 双击安装，保持默认选项

3. **启动 Docker Desktop**：
   - 从开始菜单找到 Docker Desktop
   - 首次启动会提示启用 WSL2，点击确认
   - 等待右下角图标变为绿色（Running 状态）

4. **验证安装**：
```powershell
docker --version
# 输出示例：Docker version 24.0.7, build afdd53b

docker compose version
# 输出示例：Docker Compose version v2.21.0
```

**方式二：使用 Hyper-V（旧版 Windows）**

如果你的 Windows 不支持 WSL2，可以使用 Hyper-V：
1. 启用 Hyper-V（控制面板 → 程序和功能 → 启用或关闭 Windows 功能）
2. 安装 Docker Desktop 时选择 Hyper-V 后端

**常见问题：**
- 如果启动失败，检查 BIOS 中是否启用了虚拟化
- 确保 Windows 版本为 Windows 10 专业版/企业版/教育版 或 Windows 11

##### Step 4: 克隆项目代码（Windows）

**前提：** 确保已安装 Git
- 下载地址：https://git-scm.com/download/win
- 安装时全部使用默认选项

**克隆步骤：**

1. 打开 PowerShell
2. 进入你的工作目录（例如 D 盘）：
```powershell
cd D:\aicodes\oneapi
```

3. 克隆项目：
```powershell
git clone https://github.com/songquanpeng/one-api.git
```

4. 进入项目目录：
```powershell
cd one-api
```

5. 查看项目结构：
```powershell
ls
# 或使用 Get-ChildItem
```

**如果 Git 克隆速度慢：**
可以使用国内镜像加速：
```powershell
git clone https://ghproxy.com/https://github.com/songquanpeng/one-api.git
```

##### Step 5: 下载 Go 依赖（Windows）

在项目根目录打开 PowerShell，执行：

```powershell
# 下载所有依赖
go mod download

# 验证依赖完整性
go mod verify
```

**如果下载速度慢，配置国内代理：**

```powershell
# 临时设置（仅当前会话有效）
$env:GOPROXY="https://goproxy.cn,direct"

# 永久设置（推荐）
go env -w GOPROXY=https://goproxy.cn,direct

# 验证设置
go env GOPROXY
# 应该输出：https://goproxy.cn,direct
```

然后重新执行：
```powershell
go mod download
```

这次应该会快很多！

#### 环境检查清单（Windows）

完成以上步骤后，在 PowerShell 中运行以下命令检查环境：

```powershell
# 检查 Go 版本（应 >= 1.20）
go version

# 检查 Node 版本（应 >= 16）
node --version

# 检查 npm 版本
npm --version

# 检查 Docker（如果安装了）
docker --version

# 进入项目目录并检查依赖
cd D:\aicodes\oneapi\one-api
go mod verify
```

**预期输出示例：**
```
go version go1.21.6 windows/amd64
v20.10.0
10.2.3
Docker version 24.0.7, build afdd53b
```

全部通过后，恭喜你！Windows 环境搭建完成 🎉

**常见问题排查：**

❌ **问题 1：`go` 命令找不到**
```powershell
# 解决方案：检查环境变量
$env:Path -split ';' | Select-String "Go"

# 如果没有输出，手动添加：
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Go\bin", [EnvironmentVariableTarget]::Machine)
# 然后重新打开 PowerShell
```

❌ **问题 2：权限错误**
```powershell
# 以管理员身份运行 PowerShell
# 右键 PowerShell → "以管理员身份运行"
```

❌ **问题 3：防火墙阻止 Docker**
```
# 允许 Docker 通过防火墙
# 控制面板 → Windows Defender 防火墙 → 允许应用通过防火墙
# 找到 Docker Desktop，勾选专用和公用
```

---

### 1.3 Docker 环境配置（MySQL + Redis）

如果你不想使用 SQLite，可以使用 Docker 部署 MySQL 和 Redis，这是**生产环境推荐方案**。

#### 前置条件

确保 Docker Desktop 已安装并正在运行：
```powershell
docker --version
docker ps
```

#### Step 1: 使用 Docker 部署 MySQL

**创建并启动 MySQL 容器：**

```powershell
docker run -d `
  --name mysql-oneapi `
  -e MYSQL_ROOT_PASSWORD=OneAPI@2024 `
  -e MYSQL_DATABASE=oneapi `
  -e MYSQL_USER=oneapi `
  -e MYSQL_PASSWORD=OneAPI@2024 `
  -p 3306:3306 `
  -v mysql-data:/var/lib/mysql `
  --restart always `
  mysql:8.0
```

**参数说明：**
- `--name mysql-oneapi`: 容器名称
- `MYSQL_ROOT_PASSWORD`: root 用户密码（请修改为强密码）
- `MYSQL_DATABASE`: 自动创建的数据库名
- `MYSQL_USER`: 创建普通用户
- `MYSQL_PASSWORD`: 普通用户密码
- `-p 3306:3306`: 映射端口
- `-v mysql-data:/var/lib/mysql`: 数据持久化
- `--restart always`: 开机自启

**验证 MySQL 是否启动成功：**

```powershell
# 查看容器状态
docker ps | Select-String mysql

# 查看日志
docker logs mysql-oneapi

# 应该看到类似输出：
# [Server] A temporary password is generated for root@localhost
# ...ready for connections.
```

**测试连接：**

```powershell
# 进入容器内测试
docker exec -it mysql-oneapi mysql -u oneapi -pOneAPI@2024 -e "SELECT VERSION();"

# 应该输出 MySQL 版本号
```

**常用管理命令：**

```powershell
# 停止 MySQL
docker stop mysql-oneapi

# 启动 MySQL
docker start mysql-oneapi

# 重启 MySQL
docker restart mysql-oneapi

# 查看日志
docker logs -f mysql-oneapi

# 删除容器（数据会保留在卷中）
docker rm -f mysql-oneapi

# 删除数据卷（谨慎操作！）
docker volume rm mysql-data
```

---

#### Step 2: 使用 Docker 部署 Redis

**创建并启动 Redis 容器：**

```powershell
docker run -d `
  --name redis-oneapi `
  -p 6379:6379 `
  -v redis-data:/data `
  --restart always `
  redis:7-alpine `
  redis-server --appendonly yes
```

**参数说明：**
- `--name redis-oneapi`: 容器名称
- `-p 6379:6379`: 映射端口
- `-v redis-data:/data`: 数据持久化
- `--appendonly yes`: 启用 AOF 持久化
- `redis:7-alpine`: 轻量级镜像

**验证 Redis 是否启动成功：**

```powershell
# 查看容器状态
docker ps | Select-String redis

# 测试连接
docker exec -it redis-oneapi redis-cli ping
# 应该输出：PONG
```

**常用管理命令：**

```powershell
# 停止 Redis
docker stop redis-oneapi

# 启动 Redis
docker start redis-oneapi

# 查看日志
docker logs -f redis-oneapi

# 进入 Redis CLI
docker exec -it redis-oneapi redis-cli

# 在 Redis CLI 中测试
127.0.0.1:6379> SET test hello
OK
127.0.0.1:6379> GET test
"hello"
127.0.0.1:6379> EXIT
```

---

#### Step 3: 配置 One-API 使用 MySQL + Redis

**创建 `.env` 配置文件：**

```powershell
cd D:\aicodes\oneapi\one-api

@"
# 数据库配置（MySQL）
SQL_DSN=oneapi:OneAPI@2024@tcp(localhost:3306)/oneapi

# Redis 配置
REDIS_CONN_STRING=redis://localhost:6379

# Session 密钥（必须修改！生成方法见下方）
SESSION_SECRET=CHANGE_THIS_TO_RANDOM_STRING

# 其他配置
PORT=3000
TZ=Asia/Shanghai
DEBUG=false
THEME=default
"@ | Out-File -FilePath .env -Encoding utf8
```

**生成随机 SESSION_SECRET：**

```powershell
# 生成 32 位随机字符串
$secret = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))
Write-Host "生成的 SESSION_SECRET: $secret"

# 直接更新到 .env 文件
(Get-Content .env) -replace 'SESSION_SECRET=.*', "SESSION_SECRET=$secret" | Set-Content .env
```

**验证配置：**

```powershell
# 查看 .env 文件内容
Get-Content .env
```

应该看到类似内容：
```
# 数据库配置（MySQL）
SQL_DSN=oneapi:OneAPI@2024@tcp(localhost:3306)/oneapi

# Redis 配置
REDIS_CONN_STRING=redis://localhost:6379

# Session 密钥（必须修改！生成方法见下方）
SESSION_SECRET=k8Jx2mP9qR4tY7wZ3nB6vC1xF5hL0jA8

# 其他配置
PORT=3000
TZ=Asia/Shanghai
DEBUG=false
THEME=default
```

---

#### Step 4: 启动 One-API

```powershell
cd D:\aicodes\oneapi\one-api
go run main.go
```

**预期输出：**

```
[INFO] One API v0.6.0 started
[INFO] using MySQL as database
[INFO] database initialized
[INFO] redis client initialized
[INFO] root account created
[INFO] server started on port 3000
```

**关键日志说明：**
- ✅ `using MySQL as database` - 使用 MySQL 而非 SQLite
- ✅ `redis client initialized` - Redis 连接成功
- ✅ `database initialized` - 数据库表创建成功
- ✅ `root account created` - 默认管理员账号创建成功

---

#### Step 5: 验证系统运行

1. **浏览器访问**：http://localhost:3000
2. **登录系统**：
   - 用户名：`root`
   - 密码：`123456`
3. **⚠️ 立即修改密码！**

4. **检查数据库连接**：
   ```powershell
   # 查看数据库中的表
   docker exec -it mysql-oneapi mysql -u oneapi -pOneAPI@2024 oneapi -e "SHOW TABLES;"
   
   # 应该看到：users, tokens, channels, logs 等表
   ```

5. **检查 Redis 缓存**：
   ```powershell
   # 查看 Redis 中的键
   docker exec -it redis-oneapi redis-cli KEYS '*'
   
   # 首次启动可能为空，使用系统后会逐渐有缓存
   ```

---

#### 完整 Docker Compose 方案（推荐）

如果你想一键启动所有服务，可以使用 Docker Compose。

**创建 `docker-compose.yml` 文件：**

```powershell
cd D:\aicodes\oneapi

@'
version: '3.8'

services:
  # MySQL 数据库
  mysql:
    image: mysql:8.0
    container_name: mysql-oneapi
    environment:
      MYSQL_ROOT_PASSWORD: OneAPI@2024
      MYSQL_DATABASE: oneapi
      MYSQL_USER: oneapi
      MYSQL_PASSWORD: OneAPI@2024
    ports:
      - "3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql
    restart: always
    command: --default-authentication-plugin=mysql_native_password

  # Redis 缓存
  redis:
    image: redis:7-alpine
    container_name: redis-oneapi
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data
    restart: always
    command: redis-server --appendonly yes

  # One-API 应用
  one-api:
    image: justsong/one-api
    container_name: one-api-app
    ports:
      - "3000:3000"
    environment:
      - SQL_DSN=oneapi:OneAPI@2024@tcp(mysql:3306)/oneapi
      - REDIS_CONN_STRING=redis://redis:6379
      - SESSION_SECRET=CHANGE_THIS_TO_RANDOM_STRING
      - TZ=Asia/Shanghai
    volumes:
      - ./one-api-data:/data
    depends_on:
      - mysql
      - redis
    restart: always

volumes:
  mysql-data:
  redis-data:
'@ | Out-File -FilePath docker-compose.yml -Encoding utf8
```

**启动所有服务：**

```powershell
cd D:\aicodes\oneapi
docker-compose up -d
```

**查看服务状态：**

```powershell
docker-compose ps
```

应该看到三个容器都在运行：
```
NAME                STATUS         PORTS
mysql-oneapi        Up             0.0.0.0:3306->3306/tcp
redis-oneapi        Up             0.0.0.0:6379->6379/tcp
one-api-app         Up             0.0.0.0:3000->3000/tcp
```

**查看日志：**

```powershell
# 查看所有服务日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f one-api
```

**停止所有服务：**

```powershell
docker-compose down
```

**停止并删除数据（谨慎操作！）：**

```powershell
docker-compose down -v
```

**访问系统：**

浏览器打开：http://localhost:3000
- 用户名：`root`
- 密码：`123456`

---

### 1.4 项目结构总览

#### 完整目录树

```
one-api/
│
├── main.go                    # 🚀 程序入口文件
├── go.mod                     # Go 模块依赖声明
├── go.sum                     # 依赖版本锁定
├── Dockerfile                 # Docker 构建文件
├── docker-compose.yml         # Docker Compose 配置
├── .env.example               # 环境变量示例
│
├── common/                    # 🔧 通用工具包
│   ├── config/               # 配置管理（环境变量读取）
│   ├── logger/               # 日志系统
│   ├── helper/               # 辅助函数集合
│   ├── i18n/                 # 国际化支持
│   ├── image/                # 图片处理
│   ├── message/              # 消息推送
│   ├── network/              # 网络工具
│   ├── blacklist/            # IP 黑名单
│   └── ...
│
├── controller/                # 🎮 HTTP 控制器层（业务逻辑）
│   ├── user.go               # 用户管理接口
│   ├── token.go              # Token 管理接口
│   ├── channel.go            # 渠道管理接口
│   ├── relay.go              # 请求转发接口
│   ├── redemption.go         # 兑换码管理
│   ├── log.go                # 日志查询
│   ├── billing.go            # 账单查询
│   ├── option.go             # 系统选项配置
│   ├── model.go              # 模型列表
│   ├── group.go              # 用户分组
│   ├── misc.go               # 杂项接口
│   └── auth/                 # 认证相关
│       ├── github.go         # GitHub OAuth
│       ├── lark.go           # 飞书 OAuth
│       └── wechat.go         # 微信 OAuth
│
├── model/                     # 💾 数据模型层（数据库操作）
│   ├── main.go               # 数据库初始化
│   ├── user.go               # 用户模型
│   ├── token.go              # Token 模型
│   ├── channel.go            # 渠道模型
│   ├── ability.go            # 能力模型（用户-渠道-模型关系）
│   ├── redemption.go         # 兑换码模型
│   ├── log.go                # 日志模型
│   ├── option.go             # 系统选项模型
│   ├── cache.go              # 缓存操作
│   └── utils.go              # 工具函数
│
├── middleware/                # 🛡️ 中间件层
│   ├── auth.go               # 认证中间件
│   ├── rate-limit.go         # 限流中间件
│   ├── cors.go               # CORS 跨域
│   ├── cache.go              # 缓存中间件
│   ├── logger.go             # 请求日志
│   ├── recover.go            # 异常恢复
│   ├── request-id.go         # 请求 ID 生成
│   ├── turnstile-check.go    # Cloudflare 验证
│   ├── distributor.go        # 请求分发
│   └── gzip.go               # Gzip 压缩
│
├── router/                    # 🗺️ 路由配置
│   ├── api.go                # API 路由（/api/*）
│   ├── web.go                # Web 路由（前端页面）
│   ├── relay.go              # Relay 路由（/v1/*）
│   └── main.go               # 路由注册入口
│
├── relay/                     # ⚡ 核心转发层（最重要！）
│   ├── adaptor.go            # 适配器工厂
│   ├── adaptor/              # 各渠道适配器（40+）
│   │   ├── openai/           # OpenAI 适配器（基础）
│   │   ├── anthropic/        # Claude 适配器
│   │   ├── gemini/           # Gemini 适配器
│   │   ├── baidu/            # 文心一言适配器
│   │   ├── ali/              # 通义千问适配器
│   │   ├── zhipu/            # ChatGLM 适配器
│   │   ├── xunfei/           # 讯飞星火适配器
│   │   ├── deepseek/         # DeepSeek 适配器
│   │   ├── moonshot/         # Moonshot 适配器
│   │   └── ... (30+ 更多)
│   ├── controller/           # 转发控制器
│   │   ├── chat.go           # 聊天接口转发
│   │   ├── image.go          # 绘图接口转发
│   │   ├── audio.go          # 语音接口转发
│   │   ├── error.go          # 错误处理
│   │   └── distribute.go     # 负载均衡
│   ├── model/                # 请求/响应模型
│   ├── meta/                 # 元数据管理
│   ├── channeltype/          # 渠道类型定义
│   ├── relaymode/            # 转发模式定义
│   ├── apitype/              # API 类型定义
│   ├── billing/              # 计费逻辑
│   └── constant/             # 常量定义
│
├── monitor/                   # 📊 监控模块
│   ├── metric.go             # 指标收集
│   └── manage.go             # 监控管理
│
└── web/                       # 🎨 前端代码
    ├── default/              # 默认主题
    │   ├── src/
    │   │   ├── components/   # React 组件
    │   │   ├── pages/        # 页面
    │   │   ├── helpers/      # 工具函数
    │   │   ├── context/      # React Context
    │   │   └── theme/        # 主题配置
    │   ├── public/           # 静态资源
    │   └── package.json
    ├── berry/                # Berry 主题
    └── dark/                 # Dark 主题
```

#### 架构流程图

```
┌─────────────────────────────────────────────────────────────┐
│                      用户请求                                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Router 路由层                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                  │
│  │ /api/*   │  │   /*     │  │ /v1/*    │                  │
│  │ 管理接口  │  │ 前端页面  │  │ 转发接口  │                  │
│  └──────────┘  └──────────┘  └──────────┘                  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Middleware 中间件链                          │
│  Recovery → Logger → CORS → RateLimit → Auth → Handler     │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│                  Controller 控制器层                          │
│  • 参数验证                                                   │
│  • 业务逻辑                                                   │
│  • 调用 Model                                                 │
│  • 返回响应                                                   │
└──────────────────────┬──────────────────────────────────────┘
                       │
            ┌──────────┴──────────┐
            │                     │
            ▼                     ▼
┌──────────────────┐   ┌──────────────────────┐
│   Model 数据层    │   │  Relay 转发层（核心）  │
│                  │   │                      │
│ • 数据库操作      │   │ • 选择渠道            │
│ • 缓存管理        │   │ • 获取 Adaptor       │
│ • 配额计算        │   │ • 转换请求格式        │
│                  │   │ • 调用第三方 API       │
│                  │   │ • 转换响应格式        │
│                  │   │ • 记录日志            │
└────────┬─────────┘   └──────────┬───────────┘
         │                        │
         ▼                        ▼
┌──────────────────┐   ┌──────────────────────┐
│  Database        │   │  第三方 AI API         │
│  • SQLite        │   │  • OpenAI             │
│  • MySQL         │   │  • Claude             │
│  • PostgreSQL    │   │  • Gemini             │
│                  │   │  • ... (30+ 厂商)      │
└──────────────────┘   └──────────────────────┘
```

#### 核心模块职责说明

| 模块 | 职责 | 关键文件 |
|------|------|----------|
| **Router** | 路由分发，将请求导向正确的处理器 | `router/*.go` |
| **Middleware** | 横切关注点（认证、限流、日志等） | `middleware/*.go` |
| **Controller** | 业务逻辑处理，协调 Model 和 Relay | `controller/*.go` |
| **Model** | 数据持久化，数据库 CRUD 操作 | `model/*.go` |
| **Relay** | 核心转发，适配不同厂商 API | `relay/**/*.go` |
| **Common** | 通用工具，配置、日志、辅助函数 | `common/**/*.go` |

#### 数据流向示例

**场景：用户调用聊天接口**

```
1. 用户发送 POST /v1/chat/completions
   ↓
2. Router 匹配到 relay.go 中的路由
   ↓
3. 经过中间件链：
   - TokenAuth() 验证 API Key
   - RateLimit() 检查限流
   ↓
4. Controller.relayChatCompletions() 接收请求
   ↓
5. Relay 层处理：
   a. 解析 Token，获取用户信息
   b. 根据模型名称选择合适的 Channel
   c. 获取对应的 Adaptor（如 OpenAI Adaptor）
   d. 转换请求格式为 OpenAI 标准格式
   e. 发起 HTTP 请求到第三方 API
   f. 接收响应并转换格式
   g. 计算 Token 消耗，扣除配额
   h. 记录调用日志
   ↓
6. 返回响应给用户
```

---

## 第二章：启动第一个 One-API 服务

### 2.1 快速启动（三种方式）

#### 方式选择指南

| 方式 | 适用场景 | 优点 | 缺点 |
|------|---------|------|------|
| **SQLite** | 学习、测试、小规模使用 | 无需配置，开箱即用 | 性能较低，不支持高并发 |
| **MySQL + Redis** | 生产环境、大规模使用 | 高性能，可扩展 | 需要配置数据库 |
| **Docker Compose** | 快速部署、团队协作 | 一键启动所有服务 | 需要 Docker 环境 |

---

#### 方式一：SQLite 模式（最简单，适合学习）

#### 实践任务清单（所有方式通用）

- [ ] 编译并运行项目
- [ ] 访问 Web 界面
- [ ] 使用默认账号登录
- [ ] 探索管理后台功能
- [ ] 修改默认密码（重要！）

#### 启动步骤

##### 方式一：直接运行（Windows 推荐初学者）

**步骤：**

1. 打开 PowerShell
2. 进入项目目录：
```powershell
cd D:\aicodes\oneapi\one-api
```

3. 直接运行（会自动下载依赖并编译）：
```powershell
go run main.go
```

**首次运行可能需要几分钟下载依赖，请耐心等待。**

你会看到类似输出：
```
2024/01/01 12:00:00 One API v0.6.0 started
2024/01/01 12:00:00 running in release mode
2024/01/01 12:00:00 database initialized
2024/01/01 12:00:00 root account created
2024/01/01 12:00:00 server started on port 3000
```

**如果看到错误：**
- 检查 Go 版本：`go version`（需要 >= 1.20）
- 检查依赖：`go mod download`
- 检查端口占用：`netstat -ano | findstr :3000`

##### 方式二：编译后运行（Windows）

**编译为 exe 文件：**

```powershell
# 进入项目目录
cd D:\aicodes\oneapi\one-api

# 编译（生成 one-api.exe）
go build -ldflags "-s -w" -o one-api.exe

# 运行
.\one-api.exe --port 3000
```

**参数说明：**
- `-ldflags "-s -w"`：减小文件大小（去除调试信息）
- `--port`: 指定端口号（默认 3000）
- `--log-dir`: 指定日志目录（例如 `--log-dir .\logs`）

**编译后的优势：**
- ✅ 无需 Go 环境即可运行
- ✅ 启动速度更快
- ✅ 可以分发给其他人使用

**创建快捷方式：**
1. 右键 `one-api.exe` → "创建快捷方式"
2. 右键快捷方式 → "属性"
3. 在"目标"后添加参数：`--port 3000`
4. 双击快捷方式即可启动

##### 方式三：Docker 运行（Windows 最简单）

**前提：** 已完成 Step 3 安装 Docker Desktop

**步骤：**

1. **确保 Docker Desktop 正在运行**
   - 查看系统托盘，Docker 图标应为绿色
   - 如果不是，从开始菜单启动 Docker Desktop

2. **在 PowerShell 中运行：**
```powershell
docker run -d `
  --name one-api `
  -p 3000:3000 `
  -e TZ=Asia/Shanghai `
  -v ${PWD}\data:C:\data `
  justsong/one-api
```

**注意：** PowerShell 中使用反引号 `` ` `` 作为换行符

**如果镜像拉取慢，使用国内镜像：**
```powershell
docker run -d `
  --name one-api `
  -p 3000:3000 `
  -e TZ=Asia/Shanghai `
  -v ${PWD}\data:C:\data `
  registry.cn-hangzhou.aliyuncs.com/justsong/one-api
```

3. **验证容器运行：**
```powershell
docker ps
# 应该能看到 one-api 容器

docker logs one-api
# 查看启动日志
```

4. **停止和删除容器（如需重新部署）：**
```powershell
docker stop one-api
docker rm one-api
```

#### 验证启动（Windows）

1. **浏览器访问**：http://localhost:3000
   
   如果无法访问，检查：
   ```powershell
   # 检查端口是否被占用
   netstat -ano | findstr :3000
   
   # 检查防火墙
   # 控制面板 → Windows Defender 防火墙 → 允许应用通过防火墙
   ```

2. **登录系统**：
   - 用户名：`root`
   - 密码：`123456`

3. **⚠️ 重要：立即修改密码！**
   - 点击右上角头像 → "个人信息"
   - 修改密码为强密码（建议包含大小写字母、数字、特殊字符）

#### 探索管理后台

登录后，你将看到：

**左侧菜单：**
- 📊 仪表盘 - 查看系统统计
- 👤 用户管理 - 管理注册用户
- 🔑 令牌管理 - 生成 API Token
- 📡 渠道管理 - 配置 AI 供应商
- 💳 兑换码 - 批量生成充值码
- 📝 日志查询 - 查看调用记录
- ⚙️ 系统设置 - 配置系统参数

**实践任务：**
1. 查看仪表盘统计数据
2. 尝试创建一个新用户
3. 为该用户生成一个 Token
4. 浏览渠道管理页面（暂不添加）
5. 查看系统设置中的可配置项

---

### 2.2 理解启动流程

#### main.go 完整解析

让我们逐行分析 `main.go` 的启动过程：

```go
package main

import (
    // ... 导入依赖
)

//go:embed web/build/*
var buildFS embed.FS  // 嵌入前端静态文件

func main() {
    // Step 1: 初始化通用配置
    common.Init()
    
    // Step 2: 设置日志系统
    logger.SetupLogger()
    logger.SysLogf("One API %s started", common.Version)
    
    // Step 3: 设置 Gin 模式
    if os.Getenv("GIN_MODE") != gin.DebugMode {
        gin.SetMode(gin.ReleaseMode)
    }
    
    // Step 4: 初始化数据库
    model.InitDB()        // 主数据库
    model.InitLogDB()     // 日志数据库
    
    // Step 5: 创建默认管理员账号
    err = model.CreateRootAccountIfNeed()
    if err != nil {
        logger.FatalLog("database init error: " + err.Error())
    }
    
    // Step 6: 延迟关闭数据库连接
    defer func() {
        err := model.CloseDB()
        if err != nil {
            logger.FatalLog("failed to close database: " + err.Error())
        }
    }()
    
    // Step 7: 初始化 Redis（如果配置了）
    err = common.InitRedisClient()
    if err != nil {
        logger.SysError("init redis client failed: " + err.Error())
    }
    
    // Step 8: 初始化 HTTP 客户端
    client.InitHTTPClient()
    
    // Step 9: 注册路由
    router.SetRouter(buildFS)
    
    // Step 10: 启动 HTTP 服务器
    err = server.Run()
    if err != nil {
        logger.FatalLog(err.Error())
    }
}
```

#### 关键初始化步骤详解

**Step 1: common.Init()**
- 加载 `.env` 文件（如果存在）
- 读取环境变量
- 初始化配置常量
- 设置时区

**Step 2: logger.SetupLogger()**
- 创建日志目录
- 配置日志级别
- 初始化日志文件轮转

**Step 3-4: 数据库初始化**
```go
// model/main.go 中的 InitDB()
func InitDB() (err error) {
    // 根据 SQL_DSN 判断数据库类型
    if os.Getenv("SQL_DSN") == "" {
        // 使用 SQLite
        db, err = gorm.Open(sqlite.Open("one-api.db"), &gorm.Config{})
    } else {
        // 使用 MySQL 或 PostgreSQL
        db, err = gorm.Open(mysql.Open(os.Getenv("SQL_DSN")), &gorm.Config{})
    }
    
    // 自动迁移表结构
    db.AutoMigrate(&User{}, &Token{}, &Channel{}, ...)
    
    // 创建索引
    db.Exec("CREATE INDEX IF NOT EXISTS idx_user_id ON logs(user_id)")
    // ...
}
```

**Step 5: 创建 Root 账号**
```go
func CreateRootAccountIfNeed() error {
    var user User
    // 查询是否已有用户
    DB.First(&user)
    if user.Id > 0 {
        return nil  // 已存在，无需创建
    }
    
    // 创建默认 root 账号
    rootUser := User{
        Username: "root",
        Password: hashPassword("123456"),
        Role:     RoleRootUser,
        Quota:    1000000,
    }
    DB.Create(&rootUser)
}
```

#### 实践练习：添加自定义日志

在 `main.go` 中添加日志输出，观察启动顺序：

```go
func main() {
    logger.SysLog("=== One-API 启动开始 ===")
    
    common.Init()
    logger.SysLog("✓ 配置初始化完成")
    
    logger.SetupLogger()
    logger.SysLog("✓ 日志系统初始化完成")
    
    model.InitDB()
    logger.SysLog("✓ 数据库初始化完成")
    
    // ... 其他步骤
    
    logger.SysLog("=== One-API 启动完成 ===")
}
```

重新运行，观察日志输出顺序。

---

### 2.3 配置文件详解

#### 环境变量说明

One-API 通过环境变量进行配置，支持两种方式：
1. 直接设置系统环境变量
2. 创建 `.env` 文件（推荐开发环境）

##### 核心配置项

**数据库配置：**
```bash
# SQLite（默认，无需配置）
# 不使用 SQL_DSN 即使用 SQLite

# MySQL
SQL_DSN="root:password@tcp(localhost:3306)/oneapi"

# PostgreSQL
SQL_DSN="host=localhost user=postgres password=pass dbname=oneapi port=5432 sslmode=disable"
```

**Session 配置：**
```bash
# Session 加密密钥（必须设置，否则每次重启都会失效）
SESSION_SECRET="your-random-secret-key-here"

**生成随机 SESSION_SECRET（Windows PowerShell）：**

```powershell
# 生成 32 位随机字符串
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))

# 输出示例：k8Jx2mP9qR4tY7wZ3nB6vC1xF5hL0jA8

# 直接写入 .env 文件
$secret = [Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Minimum 0 -Maximum 256 }))
"SESSION_SECRET=$secret" | Out-File -FilePath .env -Encoding utf8 -Append
```
```

**Redis 配置（Windows 可选，提升性能）：**

**方式一：使用 Windows 版 Redis**

1. 下载 Redis for Windows：
   - GitHub: https://github.com/microsoftarchive/redis/releases
   - 下载 `Redis-x64-3.2.100.zip`

2. 解压到 `C:\Redis`

3. 启动 Redis：
```powershell
cd C:\Redis
.\redis-server.exe
```

4. 配置 One-API：
```powershell
# .env 文件
REDIS_CONN_STRING="redis://localhost:6379"
```

**方式二：使用 Docker 运行 Redis（推荐）**

```powershell
# 启动 Redis 容器
docker run -d `
  --name redis `
  -p 6379:6379 `
  redis:7-alpine

# 验证
redis-cli ping
# 应该输出：PONG
```

然后配置：
```powershell
# .env 文件
REDIS_CONN_STRING="redis://localhost:6379"
```

**节点配置（多机部署）：**
```bash
# 节点类型：master（主节点）或 slave（从节点）
NODE_TYPE="master"

# 配置同步频率（秒），0 表示不同步
SYNC_FREQUENCY=60

# 前端重定向地址（从节点使用）
FRONTEND_BASE_URL="https://api.yourdomain.com"
```

**其他配置：**
```bash
# 端口号
PORT=3000

# 时区
TZ=Asia/Shanghai

# 是否启用调试模式
DEBUG=true

# 日志目录
LOG_DIR=./logs

# 主题名称
default|berry|dark
THEME=default

# Cloudflare Turnstile 密钥（人机验证）
TURNSTILE_CHECK_ENABLED=true
TURNSTILE_SITE_KEY="your-site-key"
TURNSTILE_SECRET_KEY="your-secret-key"
```

#### 实践任务：创建 .env 文件

在项目根目录创建 `.env` 文件：

```powershell
# d:\aicodes\oneapi\one-api\.env

# 数据库配置（使用 SQLite，暂不配置）
# SQL_DSN="root:123456@tcp(localhost:3306)/oneapi"

# Session 密钥（必须修改！）
SESSION_SECRET="my-super-secret-key-change-this"

# Redis（可选）
# REDIS_CONN_STRING="redis://localhost:6379"

# 其他配置
PORT=3000
TZ=Asia/Shanghai
DEBUG=false
THEME=default
```

**在 Windows 中创建 .env 文件的方法：**

**方法一：使用记事本**
1. 打开记事本
2. 粘贴上面的内容
3. 文件 → 另存为
4. 文件名：`.env`（注意前面有点）
5. 保存类型：所有文件（*.*）
6. 保存到：`D:\aicodes\oneapi\one-api\`

**方法二：使用 PowerShell**
```powershell
cd D:\aicodes\oneapi\one-api

@"
SESSION_SECRET=my-super-secret-key-change-this
PORT=3000
TZ=Asia/Shanghai
DEBUG=false
THEME=default
"@ | Out-File -FilePath .env -Encoding utf8
```

**方法三：使用 VS Code**
1. 用 VS Code 打开项目文件夹
2. 新建文件 `.env`
3. 粘贴内容并保存

重新启动服务，配置会自动加载：

```powershell
go run main.go
```

**验证配置是否生效：**

查看启动日志，应该能看到：
```
running in release mode  # DEBUG=false 的效果
server started on port 3000  # PORT=3000 的效果
```

#### 配置优先级（Windows）

```powershell
.env 文件 < 系统环境变量 < 命令行参数
```

**示例：**

```powershell
# .env 中 PORT=3000
# 但命令行指定 --port 8080
.\one-api.exe --port 8080  # 最终使用 8080
```

**在 Windows 中设置系统环境变量：**

**方法一：通过图形界面**
1. 右键"此电脑" → "属性" → "高级系统设置"
2. 点击"环境变量"
3. 在"系统变量"区域点击"新建"
4. 变量名：`SESSION_SECRET`
5. 变量值：`your-secret-key`
6. 确定保存

**方法二：通过 PowerShell（管理员）**
```powershell
# 设置永久环境变量
[Environment]::SetEnvironmentVariable("SESSION_SECRET", "your-secret-key", [EnvironmentVariableTarget]::Machine)

# 验证
$env:SESSION_SECRET

# 注意：需要重新打开 PowerShell 才能生效
```

**方法三：临时设置（仅当前会话）**
```powershell
$env:SESSION_SECRET="your-secret-key"
$env:PORT="8080"

go run main.go
```

---

## 第三章：深入理解路由与中间件

### 3.1 路由系统设计

#### 三类路由详解

One-API 的路由分为三大类，分别在三个文件中定义：

**1. API 路由** (`router/api.go`) - `/api/*`

用于管理后台的前后端交互接口。

```go
func SetApiRouter(router *gin.Engine) {
    apiRouter := router.Group("/api")
    apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
    apiRouter.Use(middleware.GlobalAPIRateLimit())
    {
        // 公开接口（无需认证）
        apiRouter.GET("/status", controller.GetStatus)
        apiRouter.GET("/notice", controller.GetNotice)
        apiRouter.GET("/about", controller.GetAbout)
        
        // 用户认证接口
        userRoute := apiRouter.Group("/user")
        {
            userRoute.POST("/register", ..., controller.Register)
            userRoute.POST("/login", ..., controller.Login)
            userRoute.GET("/logout", controller.Logout)
            
            // 需要登录的接口
            selfRoute := userRoute.Group("/")
            selfRoute.Use(middleware.UserAuth())
            {
                selfRoute.GET("/self", controller.GetSelf)
                selfRoute.PUT("/self", controller.UpdateSelf)
            }
            
            // 需要管理员权限的接口
            adminRoute := userRoute.Group("/")
            adminRoute.Use(middleware.AdminAuth())
            {
                adminRoute.GET("/", controller.GetAllUsers)
                adminRoute.POST("/", controller.CreateUser)
            }
        }
        
        // 渠道管理（仅管理员）
        channelRoute := apiRouter.Group("/channel")
        channelRoute.Use(middleware.AdminAuth())
        {
            channelRoute.GET("/", controller.GetAllChannels)
            channelRoute.POST("/", controller.AddChannel)
            channelRoute.PUT("/", controller.UpdateChannel)
            channelRoute.DELETE("/:id", controller.DeleteChannel)
        }
        
        // Token 管理（普通用户）
        tokenRoute := apiRouter.Group("/token")
        tokenRoute.Use(middleware.UserAuth())
        {
            tokenRoute.GET("/", controller.GetAllTokens)
            tokenRoute.POST("/", controller.AddToken)
        }
        
        // ... 更多路由
    }
}
```

**2. Web 路由** (`router/web.go`) - `/`

用于提供前端页面和静态资源。

```go
func SetWebRouter(router *gin.Engine, buildFS embed.FS) {
    // 前端页面（SPA 单页应用）
    router.NoRoute(func(c *gin.Context) {
        c.FileFromFS("/", http.FS(buildFS))
    })
    
    // 静态资源
    router.StaticFS("/public", http.FS(buildFS))
}
```

**3. Relay 路由** (`router/relay.go`) - `/v1/*`

核心转发接口，兼容 OpenAI API 格式。

```go
func SetRelayRouter(router *gin.Engine) {
    v1Router := router.Group("/v1")
    v1Router.Use(middleware.TokenAuth())  // Token 认证
    {
        // 聊天接口
        v1Router.POST("/chat/completions", controller.RelayChatCompletions)
        
        // 补全接口
        v1Router.POST("/completions", controller.RelayCompletions)
        
        // 绘图接口
        v1Router.POST("/images/generations", controller.RelayImageGeneration)
        
        // 语音接口
        v1Router.POST("/audio/speech", controller.RelayAudioSpeech)
        v1Router.POST("/audio/transcriptions", controller.RelayAudioTranscription)
        
        // 模型列表
        v1Router.GET("/models", controller.RelayModels)
    }
}
```

#### 路由映射表

| 路径前缀 | 用途 | 认证要求 | 示例 |
|---------|------|---------|------|
| `/api/status` | 系统状态 | 无 | GET /api/status |
| `/api/user/login` | 用户登录 | 无 | POST /api/user/login |
| `/api/user/register` | 用户注册 | 无 | POST /api/user/register |
| `/api/user/self` | 获取当前用户 | User | GET /api/user/self |
| `/api/channel/*` | 渠道管理 | Admin | GET /api/channel/ |
| `/api/token/*` | Token 管理 | User | POST /api/token/ |
| `/v1/chat/completions` | 聊天接口 | Token | POST /v1/chat/completions |
| `/v1/images/generations` | 绘图接口 | Token | POST /v1/images/generations |

#### 实践任务：绘制完整路由表

创建一个 Markdown 表格，列出所有路由及其：
- HTTP 方法
- 完整路径
- 功能描述
- 所需权限
- 对应的 Controller 函数

提示：查看 `router/api.go`、`router/relay.go` 文件

---

### 3.2 中间件链分析

#### 中间件执行顺序

Gin 框架的中间件按注册顺序执行，形成责任链：

```
Request
  ↓
┌─────────────────────┐
│  Recovery           │ ← 捕获 panic，防止服务崩溃
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  Logger             │ ← 记录请求日志
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  CORS               │ ← 跨域资源共享
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  RateLimit          │ ← 限流控制
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  Auth               │ ← 身份认证
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  Handler            │ ← 业务处理函数
└──────────┬──────────┘
           ↓
Response
```

#### 关键中间件详解

##### 1. 认证中间件 (`middleware/auth.go`)

**四种认证级别：**

```go
// 用户认证（需要登录）
func UserAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        session := sessions.Default(c)
        userId := session.Get("id")
        if userId == nil {
            c.JSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "message": "无权进行此操作，请先登录",
            })
            c.Abort()
            return
        }
        c.Set("id", userId.(int))
        c.Next()
    }
}

// 管理员认证
func AdminAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 先执行用户认证
        userId := c.GetInt("id")
        if userId == 0 {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "message": "无权进行此操作，请先登录",
            })
            return
        }
        
        // 检查是否为管理员
        user, _ := model.GetUserById(userId, false)
        if user.Role < model.RoleAdminUser {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "success": false,
                "message": "无权进行此操作，权限不足",
            })
            return
        }
        
        c.Set("role", user.Role)
        c.Next()
    }
}

// Root 超级管理员认证
func RootAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        userId := c.GetInt("id")
        user, _ := model.GetUserById(userId, false)
        if user.Role != model.RoleRootUser {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "success": false,
                "message": "无权进行此操作，权限不足",
            })
            return
        }
        c.Next()
    }
}

// Token 认证（API 调用）
func TokenAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 从 Header 或 Query 中获取 Token
        key := c.Request.Header.Get("Authorization")
        key = strings.TrimPrefix(key, "Bearer ")
        
        // 验证 Token
        token, err := model.ValidateUserToken(key)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
                "success": false,
                "message": "无效的 Token",
            })
            return
        }
        
        // 检查 Token 状态
        if token.Status != model.TokenStatusEnabled {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "success": false,
                "message": "Token 已被禁用",
            })
            return
        }
        
        // 检查额度
        if token.RemainQuota <= 0 {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "success": false,
                "message": "Token 额度不足",
            })
            return
        }
        
        c.Set("id", token.UserId)
        c.Set("token_id", token.Id)
        c.Set("token_name", token.Name)
        c.Next()
    }
}
```

##### 2. 限流中间件 (`middleware/rate-limit.go`)

**全局 API 限流：**

```go
var globalApiRateLimit = rate.NewLimiter(rate.Every(time.Second), 60)

func GlobalAPIRateLimit() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !globalApiRateLimit.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "success": false,
                "message": "请求过于频繁，请稍后再试",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**关键操作限流（更严格）：**

```go
var criticalRateLimit = rate.NewLimiter(rate.Every(time.Minute), 5)

func CriticalRateLimit() gin.HandlerFunc {
    return func(c *gin.Context) {
        if !criticalRateLimit.Allow() {
            c.JSON(http.StatusTooManyRequests, gin.H{
                "success": false,
                "message": "操作过于频繁，请稍后再试",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

应用于：
- 用户注册
- 密码重置
- 邮箱验证

##### 3. 其他中间件

**CORS 跨域** (`middleware/cors.go`)：
```go
func CORS() gin.HandlerFunc {
    config := cors.DefaultConfig()
    config.AllowAllOrigins = true
    config.AllowCredentials = true
    config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
    config.AllowHeaders = []string{"*"}
    return cors.New(config)
}
```

**请求日志** (`middleware/logger.go`)：
```go
func Logger() gin.HandlerFunc {
    return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
        return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %s\n",
            param.TimeStamp.Format("2006/01/02 - 15:04:05"),
            param.StatusCode,
            param.Latency,
            param.ClientIP,
            param.Method+" "+param.Path,
        )
    })
}
```

**异常恢复** (`middleware/recover.go`)：
```go
func Recover() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                logger.SysError(fmt.Sprintf("panic recovered: %v", err))
                c.JSON(http.StatusInternalServerError, gin.H{
                    "success": false,
                    "message": "服务器内部错误",
                })
                c.Abort()
            }
        }()
        c.Next()
    }
}
```

#### 实践练习：编写自定义中间件

创建一个记录请求耗时的中间件：

**文件：** `middleware/timing.go`

```go
package middleware

import (
    "fmt"
    "time"
    
    "github.com/gin-gonic/gin"
    "github.com/songquanpeng/one-api/common/logger"
)

func Timing() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 记录开始时间
        start := time.Now()
        
        // 处理请求
        c.Next()
        
        // 计算耗时
        duration := time.Since(start)
        
        // 记录日志
        logger.SysLog(fmt.Sprintf(
            "[TIMING] %s %s - %v",
            c.Request.Method,
            c.Request.URL.Path,
            duration,
        ))
        
        // 如果耗时超过 1 秒，记录警告
        if duration > time.Second {
            logger.SysWarn(fmt.Sprintf(
                "[SLOW REQUEST] %s %s took %v",
                c.Request.Method,
                c.Request.URL.Path,
                duration,
            ))
        }
    }
}
```

在 `router/main.go` 中注册：

```go
func SetRouter(buildFS embed.FS) *gin.Engine {
    router := gin.Default()
    
    // 添加_timing_ 中间件
    router.Use(middleware.Timing())
    
    // ... 其他中间件和路由
    
    return router
}
```

重新启动服务，观察日志中的耗时信息。

---

### 3.3 权限体系设计

#### 三种角色定义

```go
// model/user.go
const (
    RoleGuestUser  = 0   // 访客（未登录）
    RoleCommonUser = 1   // 普通用户
    RoleAdminUser  = 10  // 管理员
    RoleRootUser   = 100 // 超级管理员
)
```

#### 角色权限对比

| 权限 | Guest | User | Admin | Root |
|------|-------|------|-------|------|
| 浏览首页 | ✅ | ✅ | ✅ | ✅ |
| 用户注册 | ✅ | - | - | - |
| 用户登录 | ✅ | - | - | - |
| 查看个人信息 | - | ✅ | ✅ | ✅ |
| 修改个人信息 | - | ✅ | ✅ | ✅ |
| 生成 API Token | - | ✅ | ✅ | ✅ |
| 调用 AI 接口 | - | ✅ | ✅ | ✅ |
| 查看所有用户 | - | ❌ | ✅ | ✅ |
| 创建/删除用户 | - | ❌ | ✅ | ✅ |
| 管理渠道 | - | ❌ | ✅ | ✅ |
| 管理系统配置 | - | ❌ | ❌ | ✅ |
| 查看系统日志 | - | ❌ | ✅ | ✅ |

#### 权限检查流程

```go
// 示例：删除用户接口
func DeleteUser(c *gin.Context) {
    // 1. 获取当前用户 ID（由 AdminAuth 中间件设置）
    currentUserId := c.GetInt("id")
    currentUserRole := c.GetInt("role")
    
    // 2. 获取要删除的用户 ID
    targetId, err := strconv.Atoi(c.Param("id"))
    
    // 3. 权限检查
    // 规则：不能删除自己，不能删除比自己权限高的用户
    if targetId == currentUserId {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": "不能删除自己的账户",
        })
        return
    }
    
    targetUser, _ := model.GetUserById(targetId, false)
    if targetUser.Role >= currentUserRole {
        c.JSON(http.StatusOK, gin.H{
            "success": false,
            "message": "无法删除权限高于或等于自己的用户",
        })
        return
    }
    
    // 4. 执行删除
    err = targetUser.Delete()
    // ...
}
```

#### 代码追踪：auth.go 完整分析

打开 `middleware/auth.go`，重点理解：

1. Session 是如何存储和读取的
2. 不同角色的判断逻辑
3. Token 认证的完整流程
4. 错误处理和响应格式

**实践任务：**
在 `auth.go` 中添加注释，解释每个函数的作用和处理流程。

---

## 第四章：数据库设计与模型层

### 4.1 数据模型总览

#### 核心表结构详解

##### users 表 - 用户信息

```sql
CREATE TABLE users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username VARCHAR(64) UNIQUE NOT NULL,
    password VARCHAR(64) NOT NULL,
    role INTEGER DEFAULT 1,
    status INTEGER DEFAULT 1,
    quota BIGINT DEFAULT 0,
    used_quota BIGINT DEFAULT 0,
    request_count INTEGER DEFAULT 0,
    upload_quota INTEGER DEFAULT 0,
    access_token VARCHAR(64),
    email VARCHAR(255),
    github_id VARCHAR(64),
    wechat_id VARCHAR(64),
    lark_id VARCHAR(64),
    verification_code VARCHAR(64),
    register_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login_time TIMESTAMP
);
```

**关键字段说明：**
- `quota`: 用户总额度（美元 × 500000，例如 $1 = 500000 额度）
- `used_quota`: 已使用额度
- `request_count`: 累计请求次数
- `status`: 用户状态（1=正常，2=禁用，3=已删除）
- `role`: 用户角色（1=普通，10=管理员，100=Root）

##### tokens 表 - API 令牌

```sql
CREATE TABLE tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name VARCHAR(255),
    key VARCHAR(64) UNIQUE NOT NULL,
    status INTEGER DEFAULT 1,
    remain_quota BIGINT DEFAULT -1,  -- -1 表示无限制
    expired_time BIGINT DEFAULT -1,   -- -1 表示永不过期
    unlimited_quota BOOLEAN DEFAULT false,
    used_quota BIGINT DEFAULT 0,
    allowed_ips TEXT,                 -- IP 白名单，JSON 数组
    allowed_models TEXT,              -- 允许的模型，JSON 数组
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**令牌特性：**
- 可以设置独立额度（`remain_quota`）
- 可以设置过期时间（`expired_time`，Unix 时间戳）
- 可以限制 IP 范围（`allowed_ips`）
- 可以限制可访问的模型（`allowed_models`）

##### channels 表 - AI 渠道

```sql
CREATE TABLE channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type INTEGER NOT NULL,            -- 渠道类型（1=OpenAI, 14=Claude, ...）
    key VARCHAR(255) NOT NULL,        -- API Key
    openai_organization VARCHAR(255), -- OpenAI 组织 ID
    base_url VARCHAR(255),            -- 自定义 API 地址
    other TEXT,                       -- 其他配置（JSON）
    models TEXT NOT NULL,             -- 支持的模型列表（JSON 数组）
    group_mapping TEXT,               -- 分组映射（JSON）
    status INTEGER DEFAULT 1,         -- 状态（1=正常，2=禁用）
    priority BIGINT DEFAULT 0,        -- 优先级（越高越优先）
    weight INTEGER DEFAULT 0,         -- 权重（负载均衡用）
    response_time_millis INTEGER,     -- 平均响应时间
    balance BIGINT DEFAULT 0,         -- 余额（美元 × 10000）
    test_model VARCHAR(32),           -- 测试用模型
    auto_ban BOOLEAN DEFAULT true,    -- 失败自动禁用
    other_info TEXT,                  -- 其他信息
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**渠道类型常量：**
```go
// relay/channeltype/constant.go
const (
    OpenAI = 1
    Azure = 5
    Anthropic = 14
    Baidu = 15
    Zhipu = 16
    Ali = 17
    Xunfei = 18
    Gemini = 19
    DeepSeek = 23
    Moonshot = 25
    // ... 更多
)
```

##### abilities 表 - 能力配置

这是一个关联表，定义了"哪个用户可以访问哪个渠道的哪个模型"。

```sql
CREATE TABLE abilities (
    user_id INTEGER NOT NULL,
    channel_id INTEGER NOT NULL,
    model_name VARCHAR(64) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 0,
    UNIQUE(user_id, channel_id, model_name)
);
```

**作用：**
- 实现细粒度的权限控制
- 可以为不同用户开放不同的模型
- 可以设置模型的优先级

##### logs 表 - 请求日志

```sql
CREATE TABLE logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    channel_id INTEGER,
    token_id INTEGER,
    model_name VARCHAR(64),
    prompt_tokens INTEGER DEFAULT 0,
    completion_tokens INTEGER DEFAULT 0,
    quota INTEGER DEFAULT 0,
    finish_reason VARCHAR(64),
    timestamp INTEGER NOT NULL,
    is_stream BOOLEAN DEFAULT false,
    ip VARCHAR(64),
    retry_count INTEGER DEFAULT 0
);
```

**日志字段：**
- `prompt_tokens`: 输入 Token 数
- `completion_tokens`: 输出 Token 数
- `quota`: 消耗的额度
- `finish_reason`: 结束原因（stop/length/content_filter等）
- `is_stream`: 是否为流式请求
- `retry_count`: 重试次数

##### redemptions 表 - 兑换码

```sql
CREATE TABLE redemptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER,
    key VARCHAR(64) UNIQUE NOT NULL,
    status INTEGER DEFAULT 1,
    name VARCHAR(255),
    quota BIGINT DEFAULT 0,
    created_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    redeemed_time TIMESTAMP
);
```

#### ER 关系图

```
┌──────────────┐       ┌──────────────┐
│    users     │1     N│    tokens    │
│──────────────│───────│──────────────│
│ id (PK)      │       │ id (PK)      │
│ username     │       │ user_id (FK) │
│ password     │       │ key          │
│ role         │       │ remain_quota │
│ quota        │       │ expired_time │
│ status       │       └──────────────┘
└──────┬───────┘
       │1
       │
       │N
┌──────▼───────┐       ┌──────────────┐
│  abilities   │N     1│   channels   │
│──────────────│───────│──────────────│
│ user_id (FK) │       │ id (PK)      │
│ channel(FK)  │       │ type         │
│ model_name   │       │ key          │
│ enabled      │       │ models       │
│ priority     │       │ status       │
└──────┬───────┘       └──────────────┘
       │
       │N
       │
┌──────▼───────┐
│    logs      │
│──────────────│
│ id (PK)      │
│ user_id (FK) │
│ channel(FK)  │
│ model_name   │
│ quota        │
│ timestamp    │
└──────────────┘
```

#### 实践任务

1. **查看模型定义**：
   - 打开 `model/user.go`，找到 `User` 结构体
   - 打开 `model/channel.go`，找到 `Channel` 结构体
   - 对照上面的 SQL，理解 GORM 标签的含义

2. **编写查询**：
   在 Go 代码中实现以下查询：
   ```go
   // 获取某个用户的所有 Token
   func GetUserTokens(userId int) ([]Token, error) {
       var tokens []Token
       err := DB.Where("user_id = ?", userId).Find(&tokens).Error
       return tokens, err
   }
   
   // 获取某个用户今天消耗的额度
   func GetUserTodayQuota(userId int) (int64, error) {
       var total int64
       today := time.Now().Unix() / 86400 * 86400  // 今日零点时间戳
       err := DB.Model(&Log{}).
           Where("user_id = ? AND timestamp >= ?", userId, today).
           Select("SUM(quota)").
           Scan(&total).Error
       return total, err
   }
   ```

---

### 4.2 GORM 使用实践

#### GORM 基础概念

GORM 是 Go 语言的 ORM（对象关系映射）库，让你可以用 Go 结构体操作数据库，而不用写 SQL。

#### 模型定义示例

**完整示例：** `model/user.go`

```go
package model

import (
    "golang.org/x/crypto/bcrypt"
    "gorm.io/gorm"
)

type User struct {
    Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
    Username         string `json:"username" gorm:"unique;index;not null"`
    Password         string `json:"password" gorm:"not null"`
    PasswordHash     string `json:"-" gorm:"column:password;not null"`  // 实际存储哈希值
    Role             int    `json:"role" gorm:"type:int;default:1"`
    Status           int    `json:"status" gorm:"type:int;default:1"`
    Quota            int64  `json:"quota" gorm:"type:bigint;default:0"`
    UsedQuota        int64  `json:"used_quota" gorm:"type:bigint;default:0"`
    RequestCount     int    `json:"request_count" gorm:"type:int;default:0"`
    Email            string `json:"email" gorm:"type:varchar(255)"`
    GitHubId         string `json:"github_id" gorm:"column:github_id;type:varchar(64)"`
    RegisterTime     int64  `json:"register_time" gorm:"type:bigint"`
    LastLoginTime    int64  `json:"last_login_time" gorm:"type:bigint"`
}

// TableName 指定表名
func (user User) TableName() string {
    return "users"
}

// BeforeCreate GORM Hook：创建前自动哈希密码
func (user *User) BeforeCreate(tx *gorm.DB) error {
    hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    user.Password = string(hash)
    return nil
}

// ValidatePassword 验证密码
func (user *User) ValidatePassword(password string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
    return err == nil
}
```

#### GORM 标签说明

| 标签 | 含义 | 示例 |
|------|------|------|
| `primaryKey` | 主键 | `gorm:"primaryKey"` |
| `autoIncrement` | 自增 | `gorm:"autoIncrement"` |
| `unique` | 唯一索引 | `gorm:"unique"` |
| `index` | 普通索引 | `gorm:"index"` |
| `not null` | 非空约束 | `gorm:"not null"` |
| `type` | 字段类型 | `gorm:"type:bigint"` |
| `column` | 列名 | `gorm:"column:password"` |
| `default` | 默认值 | `gorm:"default:1"` |
| `-` | 忽略字段 | `json:"-"` |

#### CRUD 操作示例

**Create - 创建记录**

```go
// 创建用户
user := User{
    Username: "testuser",
    Password: "password123",  // BeforeCreate 会自动哈希
    Role:     RoleCommonUser,
    Quota:    100000,
}
result := DB.Create(&user)
if result.Error != nil {
    return result.Error
}
fmt.Println("创建的用户 ID:", user.Id)  // 自动填充 ID
```

**Read - 查询记录**

```go
// 根据 ID 查询
var user User
DB.First(&user, 1)  // SELECT * FROM users WHERE id = 1 LIMIT 1

// 根据条件查询
var user User
DB.Where("username = ?", "testuser").First(&user)

// 查询多个
var users []User
DB.Where("role = ?", RoleAdminUser).Find(&users)

// 分页查询
var users []User
DB.Offset(0).Limit(10).Order("id DESC").Find(&users)

// 选择特定字段
var usernames []string
DB.Model(&User{}).Select("username").Where("status = ?", 1).Find(&usernames)

// 计数
var count int64
DB.Model(&User{}).Where("status = ?", 1).Count(&count)
```

**Update - 更新记录**

```go
// 更新单个字段
DB.Model(&user).Update("quota", 200000)

// 更新多个字段
DB.Model(&user).Updates(User{
    Quota:  200000,
    Status: 2,
})

// 条件更新
DB.Model(&User{}).Where("id = ?", 1).Update("status", 2)

// 原子操作（并发安全）
DB.Model(&user).UpdateColumn("quota", gorm.Expr("quota - ?", 1000))
```

**Delete - 删除记录**

```go
// 软删除（需要模型有 DeletedAt 字段）
DB.Delete(&user)

// 硬删除
DB.Unscoped().Delete(&user)

// 条件删除
DB.Where("status = ?", 3).Delete(&User{})
```

#### 高级查询

**关联查询：**

```go
// 获取用户及其所有 Token
var user User
DB.Preload("Tokens").First(&user, 1)

// 需要在 User 模型中定义关联
 type User struct {
     // ...
     Tokens []Token `gorm:"foreignKey:UserId"`
 }
```

**聚合查询：**

```go
// 统计每个用户的总消耗
 type UserQuota struct {
     UserId    int
     TotalQuota int64
 }

var results []UserQuota
DB.Model(&Log{}).
    Select("user_id, SUM(quota) as total_quota").
    Group("user_id").
    Scan(&results)
```

**原生 SQL：**

```go
// 当 GORM 无法满足需求时，可以使用原生 SQL
var result []map[string]interface{}
DB.Raw("SELECT DATE(timestamp) as date, SUM(quota) as total FROM logs GROUP BY DATE(timestamp)").Scan(&result)
```

#### 事务处理

```go
// 事务示例：转账（扣除 A 的额度，增加 B 的额度）
err := DB.Transaction(func(tx *gorm.DB) error {
    // 扣除 A 的额度
    if err := tx.Model(&userA).Update("quota", gorm.Expr("quota - ?", amount)).Error; err != nil {
        return err  // 返回错误会回滚
    }
    
    // 增加 B 的额度
    if err := tx.Model(&userB).Update("quota", gorm.Expr("quota + ?", amount)).Error; err != nil {
        return err
    }
    
    return nil  // 返回 nil 会提交
})

if err != nil {
    fmt.Println("事务失败:", err)
}
```

#### 实践练习

**任务 1：查看现有模型**

依次打开以下文件，理解每个模型的结构：
- `model/user.go` - 用户模型
- `model/token.go` - Token 模型
- `model/channel.go` - 渠道模型
- `model/log.go` - 日志模型

**任务 2：添加新字段**

为 `User` 模型添加 `avatar_url` 字段：

```go
type User struct {
    // ... 现有字段
    AvatarUrl string `json:"avatar_url" gorm:"type:varchar(255);default:''"`
}
```

重新启动服务，GORM 会自动添加该列到数据库。

验证：
```sql
-- 如果使用 SQLite
sqlite3 one-api.db
.schema users

-- 应该能看到 avatar_url 字段
```

**任务 3：编写复杂查询**

实现以下功能：
```go
// 获取最近 7 天活跃的用户（有调用记录）
func GetActiveUsersLast7Days() ([]User, error) {
    sevenDaysAgo := time.Now().AddDate(0, 0, -7).Unix()
    
    var userIds []int
    DB.Model(&Log{}).
        Distinct("user_id").
        Where("timestamp >= ?", sevenDaysAgo).
        Pluck("user_id", &userIds)
    
    var users []User
    DB.Where("id IN ?", userIds).Find(&users)
    
    return users, nil
}
```

---

### 4.3 数据库初始化流程

#### InitDB() 完整解析

**文件：** `model/main.go`

```go
func InitDB() (err error) {
    // Step 1: 判断数据库类型
    var dbType string
    sqlDSN := os.Getenv("SQL_DSN")
    
    if sqlDSN == "" {
        // 使用 SQLite
        dbType = "sqlite"
        log.Println("using SQLite as database")
    } else {
        // 判断是 MySQL 还是 PostgreSQL
        if strings.HasPrefix(sqlDSN, "postgres://") {
            dbType = "postgres"
            log.Println("using PostgreSQL as database")
        } else {
            dbType = "mysql"
            log.Println("using MySQL as database")
        }
    }
    
    // Step 2: 创建数据库连接
    switch dbType {
    case "sqlite":
        db, err = gorm.Open(sqlite.Open("one-api.db"), &gorm.Config{
            DisableForeignKeyConstraintWhenMigrating: true,
        })
    case "mysql":
        db, err = gorm.Open(mysql.Open(sqlDSN), &gorm.Config{
            DisableForeignKeyConstraintWhenMigrating: true,
        })
    case "postgres":
        db, err = gorm.Open(postgres.Open(sqlDSN), &gorm.Config{
            DisableForeignKeyConstraintWhenMigrating: true,
        })
    }
    
    if err != nil {
        return err
    }
    
    // Step 3: 配置连接池
    sqlDB, err := db.DB()
    if err != nil {
        return err
    }
    
    sqlDB.SetMaxIdleConns(10)                  // 最大空闲连接数
    sqlDB.SetMaxOpenConns(100)                 // 最大打开连接数
    sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最大生命周期
    
    // Step 4: 自动迁移表结构
    err = db.AutoMigrate(
        &User{},
        &Token{},
        &Channel{},
        &Ability{},
        &Redemption{},
        &Log{},
        &Option{},
    )
    
    if err != nil {
        return err
    }
    
    // Step 5: 创建索引
    createIndex()
    
    return nil
}
```

#### AutoMigrate 工作原理

GORM 的 `AutoMigrate` 会：
1. 检查表是否存在
2. 如果不存在，创建表
3. 如果存在，检查字段是否有变化
4. 添加新字段（不会删除旧字段）
5. 更新字段类型（如果兼容）

**注意：**
- ✅ 添加新字段
- ✅ 修改字段类型（部分情况）
- ❌ 删除字段
- ❌ 修改字段名

如果需要删除字段，需要手动执行 SQL。

#### 索引创建

```go
func createIndex() {
    // 为常用查询字段创建索引，提升查询性能
    
    // logs 表索引
    db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_user_id ON logs(user_id)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_channel_id ON logs(channel_id)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_token_id ON logs(token_id)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_logs_model_name ON logs(model_name)")
    
    // tokens 表索引
    db.Exec("CREATE INDEX IF NOT EXISTS idx_tokens_key ON tokens(key)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_tokens_user_id ON tokens(user_id)")
    
    // channels 表索引
    db.Exec("CREATE INDEX IF NOT EXISTS idx_channels_type ON channels(type)")
    db.Exec("CREATE INDEX IF NOT EXISTS idx_channels_status ON channels(status)")
}
```

#### 实践练习

**任务：查看数据库文件**

如果使用 SQLite，可以直接查看数据库：

```bash
# 安装 SQLite 命令行工具
# Windows: choco install sqlite
# macOS: brew install sqlite
# Linux: apt-get install sqlite3

# 打开数据库
sqlite3 one-api.db

# 查看所有表
.tables

# 查看 users 表结构
.schema users

# 查询数据
SELECT id, username, role, quota FROM users;

# 退出
.quit
```

---

### 4.4 缓存机制

#### Redis 缓存策略

One-API 使用 Redis 缓存热点数据，减少数据库压力。

**缓存内容：**
1. 渠道列表（频繁查询）
2. 用户配额（高频读写）
3. Token 验证结果
4. 系统配置选项

#### 缓存实现

**文件：** `model/cache.go`

```go
package model

import (
    "encoding/json"
    "time"
    
    "github.com/go-redis/redis/v8"
    "github.com/songquanpeng/one-api/common"
)

var RDB *redis.Client

// 缓存键前缀
const (
    CachePrefixChannel = "channel:"
    CachePrefixUser    = "user:"
    CachePrefixToken   = "token:"
    CachePrefixOption  = "option:"
)

// 缓存过期时间
const (
    CacheTTLChannel = 10 * time.Minute
    CacheTTLUser    = 5 * time.Minute
    CacheTTLToken   = 3 * time.Minute
    CacheTTLOption  = 15 * time.Minute
)

// 获取渠道缓存
func GetChannelCache(id int) (*Channel, error) {
    key := fmt.Sprintf("%s%d", CachePrefixChannel, id)
    
    // 尝试从 Redis 获取
    data, err := RDB.Get(common.Ctx, key).Result()
    if err == nil {
        // 缓存命中
        var channel Channel
        json.Unmarshal([]byte(data), &channel)
        return &channel, nil
    }
    
    // 缓存未命中，从数据库查询
    channel, err := GetChannelById(id, false)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    data, _ = json.Marshal(channel)
    RDB.Set(common.Ctx, key, data, CacheTTLChannel)
    
    return channel, nil
}

// 清除渠道缓存
func DeleteChannelCache(id int) {
    key := fmt.Sprintf("%s%d", CachePrefixChannel, id)
    RDB.Del(common.Ctx, key)
}

// 获取用户配额缓存
func GetUserQuotaCache(id int) (int64, error) {
    key := fmt.Sprintf("%s%d:quota", CachePrefixUser, id)
    
    quota, err := RDB.Get(common.Ctx, key).Int64()
    if err == nil {
        return quota, nil
    }
    
    // 从数据库查询
    user, err := GetUserById(id, false)
    if err != nil {
        return 0, err
    }
    
    // 写入缓存
    RDB.Set(common.Ctx, key, user.Quota, CacheTTLUser)
    
    return user.Quota, nil
}

// 更新用户配额（同时更新数据库和缓存）
func UpdateUserQuota(id int, delta int64) error {
    // 更新数据库
    err := DB.Model(&User{}).Where("id = ?", id).
        Update("quota", gorm.Expr("quota + ?", delta)).Error
    if err != nil {
        return err
    }
    
    // 更新缓存
    key := fmt.Sprintf("%s%d:quota", CachePrefixUser, id)
    RDB.IncrBy(common.Ctx, key, delta)
    
    return nil
}
```

#### 缓存一致性

**问题：** 如何保证缓存和数据库的一致性？

**解决方案：**
1. **写穿透（Write-through）**：更新数据库的同时更新缓存
2. **失效策略（Invalidate）**：更新数据库后删除缓存，下次读取时重新加载
3. **设置合理的 TTL**：即使不一致，也会在几分钟后自动恢复

One-API 采用策略 2 + 3 的组合。

#### 实践任务

**配置 Redis：**

1. 安装 Redis
   ```bash
   # Windows: 下载 https://github.com/microsoftarchive/redis/releases
   # macOS: brew install redis
   # Linux: apt-get install redis-server
   ```

2. 启动 Redis
   ```bash
   redis-server
   ```

3. 配置 One-API
   ```bash
   # .env 文件
   REDIS_CONN_STRING="redis://localhost:6379"
   ```

4. 重启服务，观察日志
   ```
   redis client initialized
   ```

5. 使用 Redis CLI 查看缓存
   ```bash
   redis-cli
   KEYS *
   GET channel:1
   TTL channel:1  # 查看过期时间
   ```

---

## 第五章：Controller 层业务逻辑

（由于篇幅限制，后续章节将在下一部分继续...）

---

**学习进度检查点：**

完成前四章后，你应该能够：
- ✅ 成功运行 One-API 服务
- ✅ 理解项目的整体架构和目录结构
- ✅ 掌握路由系统和中间件机制
- ✅ 熟练使用 GORM 进行数据库操作
- ✅ 了解缓存机制和 Redis 配置

**下一步：**
继续学习第五章，深入理解业务逻辑层的实现。

---

*教程持续更新中...*
