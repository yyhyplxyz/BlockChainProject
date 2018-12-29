# 外部服务剖析

## 如何提供服务？
决定于应用场景 终端用户
* 智能硬件 socket/tcp 太阳能发电
* 游戏、电商、社交 web/app http
* 企业内部 rpc(grpc)

## 如何选择SDK?
* nodejs 4
* java 3
* golang 1

构造交易、发送交易、数据查询

## SDK的模块

### 区块链管理
* 通道创建&加入
* 链码的安装、实例化、升级等
* admin | 云服务提供商

### 数据查询
* 区块
* 交易
* 区块浏览器 ethscan eospark block-explorer

### 区块链交互
* 发起交易 invoke query

### 事件监听
* 业务事件 SendEvent
* 系统事件 block/trancastion