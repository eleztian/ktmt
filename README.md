# ktmt

keepalive tcp message transport

## Feature

1. 支持断开重连
3. 服务质量(Qos0/Qos1)最多一次/至少一次
4. 双向的业务心跳
4. 读写超时

## 整体结构

- 应用层(业务协议)

- 表示层(序列化)

- 会话层(session 报文交互协议)

  服务质量保证

- 连接层(tcp连接管理/keepalive)

### 应用层

[Example](./example)

### 表示层

采用json作为数据序列化表示

### 会话层

[协议文档](./docs/PROTO.md)

### 连接层

心跳检测，连接管理

