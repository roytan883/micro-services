#push-connector

> 设计思路: 单纯类网关连接无状态微服务, 仅用于连接客户端,并提供`push`接口供内部其它微服务的消息发送给客户端. 本身不做任何逻辑, 甚至ACK也不处理,广播给外部其它微服务处理. 这样保持了与客户端连接的单纯和稳定.

* 无状态，可启动多个进程用于连接客户端
* 标准Websocket协议，客户端使用`ws://....../ws?xxx=abc&yyy=123`连接并传入参数
* 通过内部RPC call调用`auth`接口检查连接URL中的参数, 是否建立连接
* 提供`push(uids, msgId, msgBody)`RPC接口供其它服务器调用(`push-sender`)
* 对于Push消息客户端返回的ACK消息,间隔1s批量通过RPC广播给`push-sender`
* 客户端连接和断开时,间隔1s批量通过RPC广播`connect和disconnect`事件给`push-sender`
* 侦听`PushConnector.syncUsersInfo`事件, 间隔3s,每次1w的形式,将当前服务器中所有用户信息RPC广播给`push-sender`
* 提供`kick(uid, platform)`RPC接口供其它服务器调用

* 相关定义如下:
```go
//RPC定义(带Action为RPC call, 其它为PRC broadcast)
cWsConnectorActionPush       = "ws-connector.push"              //in: pushMsgStruct || out: null, err
cWsConnectorActionCount      = "ws-connector.count"             //in: null || out: count, err
cWsConnectorActionMetrics    = "ws-connector.metrics"           //in: null || out: metricsStruct, err
cWsConnectorActionUserInfo   = "ws-connector.userInfo"          //in: userIDStruct || out: []ClientInfo, err
cWsConnectorInPush           = "ws-connector.in.push"           //pushMsgStruct
cWsConnectorInKickClient     = "ws-connector.in.kickClient"     //cidStruct
cWsConnectorInKickUser       = "ws-connector.in.kickUser"       //userIDStruct
cWsConnectorOutConnect       = "ws-connector.out.connect"       //ClientInfo
cWsConnectorOutDisConnect    = "ws-connector.out.disconnect"    //ClientInfo
cWsConnectorInSyncUsersInfo  = "ws-connector.in.syncUsersInfo"  //null
cWsConnectorOutSyncUsersInfo = "ws-connector.out.syncUsersInfo" //ClientInfo
cWsConnectorOutAck           = "ws-connector.out.ack"           //ackStruct
cWsConnectorInSyncMetrics    = "ws-connector.in.syncMetrics"    //null
cWsConnectorOutSyncMetrics   = "ws-connector.out.syncMetrics"   //metricsStruct
```
