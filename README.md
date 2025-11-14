# Gas Oracle

一个多链 Gas 费用预言机服务，通过实时扫描区块链数据来估算交易手续费，并提供 gRPC API 接口供外部系统调用。

## 功能特性

- **多链支持**：支持多个区块链网络的 Gas 费用估算
- **实时扫链**：定期扫描最新区块，动态计算 Gas 费用
- **代币价格集成**：集成代币市场价格，支持跨代币手续费估算
- **gRPC API**：提供高性能的 gRPC 接口
- **数据持久化**：使用 PostgreSQL 存储 Gas 费用和代币价格数据
- **大小写不敏感查询**：支持代币符号的大小写不敏感查询

## 技术栈

- **语言**：Go 1.25.1
- **数据库**：PostgreSQL
- **RPC 框架**：gRPC + Protocol Buffers
- **区块链交互**：go-ethereum (geth)
- **ORM**：GORM v2
- **HTTP 客户端**：Resty

## 项目结构

```
.
├── cmd/gas-oracle/          # 主程序入口
├── config/                  # 配置文件解析
├── database/                # 数据库模型和操作
│   └── utils/serializers/   # 自定义序列化器（支持 big.Int）
├── migrations/              # 数据库迁移脚本
├── services/grpc/           # gRPC 服务实现
│   ├── protobuf/            # Protocol Buffers 定义
│   └── gasFeePb/            # 生成的 protobuf Go 代码
├── synchronizer/            # 区块链同步器
│   ├── node/                # 节点客户端
│   └── retry/               # 重试策略
├── worker/                  # 后台任务（代币价格获取）
├── common/                  # 公共工具
└── script/                  # 编译脚本
```

## 安装指南

### 前置要求

- Go 1.25.1 或更高版本
- PostgreSQL 12 或更高版本
- protoc 编译器（用于编译 .proto 文件）

### 安装步骤

1. **克隆仓库**
```bash
git clone <repository-url>
cd gas-oracle
```

2. **安装 Go 依赖**
```bash
go mod download
```

3. **编译 Protocol Buffers**
```bash
bash script/compile.sh
```

4. **配置数据库**

创建 PostgreSQL 数据库：
```bash
createdb -U postgres gasoracle
```

运行数据库迁移：
```bash
./gas-oracle migrate -c ./gas-oracle.yaml
```

5. **编译程序**
```bash
go build -o gas-oracle ./cmd/gas-oracle
```

## 配置说明

创建配置文件 `gas-oracle.yaml` 或 `gas-oracle.local.yaml`：

```yaml
# 区块扫描配置
back_offset: 2              # 向前回溯的区块数
loop_internal: 5s           # 扫描循环间隔

# gRPC 服务配置
server:
  host: 0.0.0.0
  port: 8081

# 代币价格 API 配置
skyeye_url: http://your-price-api:port

# 支持的代币列表
symbols:
  - name: "btc"
    decimal: 6
  - name: "eth"
    decimal: 18
  - name: "usdt"
    decimal: 6

# RPC 节点配置
rpcs:
  - rpc_url: 'https://eth-sepolia.g.alchemy.com/v2/YOUR_API_KEY'
    chain_id: 11155111
    native_token: ETH
    decimal: 18

  - rpc_url: 'https://rpc-testnet.roothashpay.com'
    chain_id: 90101
    native_token: RHS
    decimal: 18

# 数据库配置
master_db:
  db_host: "127.0.0.1"
  db_port: 5432
  db_user: "postgres"
  db_password: "your_password"
  db_name: "gasoracle"
```

## 使用方法

### 1. 运行数据库迁移

```bash
./gas-oracle migrate -c ./gas-oracle.yaml
```

### 2. 启动区块扫链服务

扫描区块链并计算 Gas 费用：

```bash
./gas-oracle index -c ./gas-oracle.yaml
```

服务会：
- 连接到配置的 RPC 节点
- 定期扫描最新区块
- 计算平均 Gas 费用
- 获取代币市场价格
- 将数据存储到数据库

### 3. 启动 gRPC 服务

提供 API 查询接口：

```bash
./gas-oracle grpc -c ./gas-oracle.yaml
```

gRPC 服务默认监听在 `0.0.0.0:8081`

### 4. 同时运行两个服务

在生产环境中，通常需要同时运行扫链服务和 gRPC 服务：

```bash
# 终端 1：运行扫链服务
./gas-oracle index -c ./gas-oracle.yaml

# 终端 2：运行 gRPC 服务
./gas-oracle grpc -c ./gas-oracle.yaml
```

或使用进程管理器（如 systemd、supervisor）：

```bash
# 后台运行
nohup ./gas-oracle index -c ./gas-oracle.yaml > index.log 2>&1 &
nohup ./gas-oracle grpc -c ./gas-oracle.yaml > grpc.log 2>&1 &
```

## API 文档

### gRPC API

服务定义在 `services/grpc/protobuf/gasfee.proto`：

#### GetTokenPriceAndGasByChainId

查询指定链的 Gas 费用，并计算使用指定代币支付的预估费用。

**请求参数：**
```protobuf
message TokenGasPriceRequest {
  string consumer_token = 1;  // 消费者 token（预留）
  uint64 chain_id = 2;        // 链 ID
  string symbol = 3;          // 代币符号（如 "ETH", "USDT"）
}
```

**响应数据：**
```protobuf
message TokenGasPriceResponse {
  uint64 return_code = 1;     // 返回码（100 表示成功）
  string message = 2;         // 返回消息
  string market_price = 3;    // 代币市场价格
  string symbol = 4;          // 代币符号
  string predict_fee = 5;     // 预估手续费（使用指定代币计价）
}
```

**费用计算公式：**
```
预估费用 = (基础 Gas 费用 / 10^小数位数) × (原生代币价格 / 目标代币价格)
```

### 使用 gRPCUI 测试

安装 gRPCUI：
```bash
go install github.com/fullstorydev/grpcui/cmd/grpcui@latest
```

启动 gRPCUI：
```bash
grpcui -plaintext localhost:8081
```

在浏览器中访问 `http://localhost:8081` 进行交互式测试。

**示例请求：**
- `chain_id`: `11155111`（Sepolia）
- `symbol`: `eth`（大小写不敏感）
- `consumer_token`: `test_token`

## 数据库结构

### gas_fee 表

存储各链的 Gas 费用数据：

| 字段 | 类型 | 说明 |
|------|------|------|
| guid | TEXT | 主键（UUID） |
| chain_id | UINT256 | 链 ID |
| token_name | VARCHAR | 原生代币名称 |
| predict_fee | VARCHAR | 预估 Gas 费用 |
| decimal | SMALLINT | 代币精度 |
| timestamp | INTEGER | 更新时间戳 |

### token_price 表

存储代币市场价格：

| 字段 | 类型 | 说明 |
|------|------|------|
| guid | TEXT | 主键（UUID） |
| token_name | VARCHAR | 代币全名 |
| token_symbol | VARCHAR | 代币符号（索引） |
| skeye_symbol | VARCHAR | Skyeye API 符号 |
| market_price | VARCHAR | 市场价格 |
| decimal | SMALLINT | 代币精度 |
| timestamp | INTEGER | 更新时间戳 |

## 开发指南

### 修改 Protocol Buffers

1. 编辑 `services/grpc/protobuf/gasfee.proto`
2. 运行编译脚本：
```bash
bash script/compile.sh
```
3. 重新编译程序

### 添加新链支持

1. 在配置文件的 `rpcs` 部分添加新链：
```yaml
rpcs:
  - rpc_url: 'https://new-chain-rpc-url'
    chain_id: 12345
    native_token: NEW
    decimal: 18
```

2. 重启服务

### 自定义序列化器

项目使用自定义序列化器处理 `big.Int` 类型：

- `database/utils/serializers/u256.go`：处理 uint256 类型
- `database/utils/serializers/uuid.go`：处理 UUID 类型
- `database/utils/serializers/bytes.go`：处理字节数组

序列化器通过 `init()` 函数自动注册到 GORM。

### 代码结构说明

**同步器（Synchronizer）**
- 负责扫描区块链
- 计算平均 Gas 费用
- 定期更新数据库

**工作器（Worker）**
- 定期获取代币市场价格
- 更新 `token_price` 表

**gRPC 服务**
- 处理客户端查询请求
- 计算跨代币手续费

## 故障排查

### 问题 1：protobuf 编译失败

**错误**：`protoc-gen-go: command not found`

**解决方案**：
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

### 问题 2：数据库连接失败

**错误**：`failed to connect to database`

**解决方案**：
1. 检查 PostgreSQL 是否运行
2. 验证配置文件中的数据库凭据
3. 确保数据库已创建

### 问题 3：GORM 序列化错误

**错误**：`invalid field found for struct ... ChainId: define a valid foreign key`

**解决方案**：
确保 `database/db.go` 导入了序列化器包：
```go
import (
    _ "github.com/Sandwichzzy/gas-oracle/database/utils/serializers"
)
```

### 问题 4：gRPC 查询失败（record not found）

**错误**：`ERROR: get token price fail, err="record not found"`

**原因**：
- 数据库中没有对应的代币价格数据
- 代币符号大小写不匹配（已修复，支持大小写不敏感）

**解决方案**：
1. 确保 `index` 服务正在运行并获取代币价格
2. 检查数据库中是否有数据：
```sql
SELECT * FROM token_price;
SELECT * FROM gas_fee;
```
3. 如果代币价格 API 不可用，可以手动插入测试数据：
```sql
INSERT INTO token_price (token_name, token_symbol, market_price, decimal, timestamp)
VALUES ('ETH', 'ETH', '3480', 18, EXTRACT(EPOCH FROM NOW()));
```

### 问题 5：RPC 连接超时

**错误**：`dial tcp: i/o timeout`

**解决方案**：
1. 检查 RPC URL 是否正确
2. 验证网络连接
3. 如果使用第三方 RPC 服务（如 Alchemy），检查 API 密钥是否有效

## 性能优化建议

1. **数据库索引**：已为 `chain_id` 和 `token_symbol` 创建索引
2. **批量操作**：GORM 配置了批量插入（batch size: 500）
3. **连接池**：使用 GORM 的连接池管理
4. **扫描间隔**：根据链的出块速度调整 `loop_internal`


## 更新日志

### v0.0.1
- 初始版本
- 支持多链 Gas 费用估算
- 提供 gRPC API
- 支持大小写不敏感的代币查询
