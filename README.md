# Introduction

xhe is a VPN based on WireGuard and WebRTC, the biggest feature is that it can be used in the browser.

## supported platforms

- [x] [Browser](https://github.com/remoon-net/xhe-link)
- [x] Linux

# Install

## From Source

```sh
go install remoon.net/xhe@v0.1.6
```

# Usage

```sh
xhe -k {key} -l https://xhe.remoon.net -p peer://pubkey
```

`-l` set signaler server link. `https://xhe.remoon.net` is a test signaler server.

In production environment you need selfhost signaler server for yourself. signaler server source code: <https://github.com/remoon-net/xhe-hub>

`-p` set peer, peer link has three link mode:

- pubkey link `peer://{pubkey}[/preshared_key]`
- signaler link `https://xhe.remoon.net/path?peer={pubkey}[&preshared=preshared_key][&keepalive=15]`
- cname link `peer://a-peer.remoon.net[/preshared_key]?[keepalive=15]`

### peer link details

recommend: cname link mode

ps: `pubkey`, `preshared_key` both use hex encode

#### pubkey link

add pubkey to WireGuard peers.
Why don't need config peer AllowedIPs? Because each pubkey has a corresponding ip, which can be obtained through `xhe ip {pubkey}`

#### signaler link

the link will exchange WebRTC Session Description via http post to connect.

the query param `peer={pubkey}` is required

#### cname link

cname link is set `signaler link` to `a-peer.remoon.net` URI dns record.

recommend set ip to `a-peer.remoon.net` AAAA dns record, the ip is from `xhe ip {pubkey}`
if set, you can access peer by `a-peer.remoon.net`, not longer IPv6 ip

and cname link is easily copy and share it to your friend, because it is not included 64 string length pubkey

# Todo

- [ ] UI
- [ ] Other platform client support
  - [ ] Windows
  - [ ] Andorid
  - [ ] Mac
  - [ ] iPhone
- [ ] ICE relay

Sponsors can expedite todo development.

I am an individual developer, if open source project has paid me, I will spend more time at open source project,
if not I have to do some side project to keep open source project development

❤ Open Source ❤
