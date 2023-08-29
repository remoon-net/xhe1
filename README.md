# 简介

xhe 是一个基于 WireGuard 和 WebRTC 的 VPN, 最大的特点是可以在浏览器中使用, 目标是万物互联

注: xhe 并不是性能型的 VPN, 在性能较差(J1900)的机器上只能跑到 45M.

## 互联目标

- [x] [浏览器](https://github.com/remoon-net/xhe-link)
- [x] Linux
- [x] Windows 能用但是报毒, 需要折腾软件签名
- [ ] Andorid
- [ ] Mac
- [ ] iPhone

# 安装

暂时只面向 Linux 用户, 直接使用 go 安装

```sh
go install remoon.net/xhe@v0.1.4
```

在 UI 完成后, 会提供各种安装包方便各位使用

# 使用

```sh
xhe -k {key} -l https://xhe.remoon.net -p peer://pubkey
```

`-l` 指定要连接的信令服务器. `https://xhe.remoon.net` 是官方提供的测试用信令服务器,
方便各位快速测试, 不要在生产环境中使用该测试用信令服务器.

在生产环境中使用需要搭建一个属于自己的信令服务器, 实现在这: <https://github.com/remoon-net/xhe-hub>

`-p` 设置节点, 节点链接有三种模式:

- 公钥模式 `peer://{pubkey}[/preshared_key]`
- 链接模式 `https://xhe.remoon.net/path?peer={pubkey}[&preshared=preshared_key][&keepalive=15]`
- 别名模式 `peer://a-peer.remoon.net[/preshared_key]?[keepalive=15]`

### 节点链接模式详解

推荐使用 "别名模式"

注: `pubkey`, `preshared_key` 均使用 hex 编码

#### 公钥模式

将 pubkey 添加到 WireGuard 的节点中.
如何配置 AllowedIPs 呢? 不用配置, 每个 pubkey 都有一个对应的 ip, 可通过 `xhe ip {pubkey}` 获得

#### 链接模式

该链接为"信令链接", 用于交换 WebRTC SDP 以建立连接. 其中的 `peer` 参数必不可少

#### 别名模式

别名模式就是为 `a-peer.remoon.net` 添加一条 `URI` dns 记录, 内容为"信令链接": `https://a-peer.remoon.net/path?peer={pubkey}`

还推荐为 `a-peer.remoon.net` 添加一条 `AAAA` dns 记录, 内容为 `xhe ip {pubkey}` 输出的 ip,
这样访问 `a-peer.remoon.net` 就能访问对应节点了, 免去记忆超长的 IPv6 ip

相较于链接模式来说, 别名的链接短的多, 因为少了 64 字节的公钥, 简短就便于复制和分享

# Todo

点击捐赠, 助力开发

- [ ] UI [助力](https://xhe.remoon.net/sponsor/#ui)
- [ ] 完善各平台客户端支持 [助力](https://xhe.remoon.net/sponsor/#platform)
- [ ] 支持 ICE 中继服务器 [助力](https://xhe.remoon.net/sponsor/#ice)
