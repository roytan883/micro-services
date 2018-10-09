#ws-online

> 设计思路: 记录用户在线状态微服务, 并提供查询在线状态和分配`ws`链路功能. 这里的online是指一定时间段(30min)内在线, 屏蔽了移动环境下设备的频繁断线状态. 提供RPC接口供其它服务器查询用户在线状态.

* 一般情况下单进程微服务即可
* 启动时通过广播`ws-connector.in.syncUsersInfo`来通知`ws-connector`将它们已存在的连接信息以`ws-connector.out.syncUsersInfo`广播出来, 自己接收并存储
* 平时接收`ws-connector.out.connect/disconnect`事件, 建立新在线状态和删除在线状态(延时30min后删除disconnect的用户信息)
* 用户在线状态分为`online, tempOffline, offline`, `offline`和初次`online`时要广播给其它微服务
* 提供`usersOnlineInfos`RPC接口供其它微服务查询使用
* 内部间隔获取所有`ws-connector`的`syncMetrics`状态(10s), 将超时(15s)的`ws-connector`设为不可用(TODO:报警)
* 提供`dispatch(uid)`查询到当前连接数最少的可用`ws-connector`
* 若该uid当前已分配给任意`ws-connector`, 则继续分配到该`ws-connector`(保证用户多设备只会分配到同一服务器)
* 间隔一定时间(60s), 将有变化的在线状态用户信息存入DB中. 用于实现活跃用户(30天)功能. 业务层推送前可以通过DB中表关联查询, 过滤出要推送的活跃用户, 通过`ws-sender`微服务推送消息.

* 相关定义如下:
```go
//RPC定义(带Action为RPC call, 其它为PRC broadcast)
cWsOnlineActionUsersOnlineInfos = "ws-online.usersOnlineInfos" //in: idsStruct `json:"ids"` || out:onlineStatusBulkStruct, err
cWsOnlineOutOnline              = "ws-online.out.online"       //ClientInfo
cWsOnlineOutOffline             = "ws-online.out.offline"      //ClientInfo
```
