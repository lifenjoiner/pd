`pd` 是一个本地的代理调度器。

## 特性

1. 专注做一个连接调度器，上级代理要自备；
2. 可自动选择回应最快的 IP；
3. 自动生成可用性评分，可用性低时切换用代理；
4. 支持 socks4a/socks5/http 协议
5. 支持预设“直连/被封”列表；
6. 可做为 PAC 文件服务器。

## 用法

详参 `pd -h`.

例子：

Windows 只服务本机
```bat
start pd.exe -proxies=http://127.0.0.1:1080,http://127.0.0.1:2080,http://127.0.0.1:3080,socks5://127.0.0.1:1081,socks5://127.0.0.1:2081,socks5://127.0.0.1:3081,socks4a://127.0.0.1:2081 -netprobeurl=https://www.toutiao.com
```

*nix 服务路由器下的局域网
```sh
pd -proxies=http://127.0.0.1:1080,http://127.0.0.1:2080,http://127.0.0.1:3080,socks5://127.0.0.1:1081,socks5://127.0.0.1:2081,socks5://127.0.0.1:3081,socks4a://127.0.0.1:2081 -netprobeurl=https://www.toutiao.com -listens=192.168.2.1:6699 -direct=/etc/pd/direct -blocked=/etc/pd/blocked -statfile=/tmp/stat.json
```

## 支持
* 静态规则：子域名优先。
* 静态规则：`direct` > `blocked`.
* 静态 `blocked` 主机名（IP）总是走代理。
* 静态 `direct` 主机名（IP）总是直连。
* 一般主机名（IP）：得分动态决定尝试直连次数，如果没有成功，从反应最快的代理开始尝试 3 次；如果之前直接尝试的代理，却没有提供代理，回落尝试 1 次直连。
* 信任你的 DNS。 如果它不够可靠，改进它，要不然就把那些特殊的域名直接放进 `blocked` 里。对于 DNS 服务器，建议使用 `0.0.0.0`/`::` 或者禁用域名列表来做拦截，因为 `127.0.0.1`/`::1` 或者其它保留 IP 可能正被某服务器使用。
* 使用相同协议的上游代理原始请求。

## 不支持

* 代理做身份验证。本地代理没必要。
* 非公网 IP 或者非域名通信被传递到上游代理。
* SOCKS BIND 和 UDP。

## 静态主机名匹配语法

```INI
# pd 的灵魂是使用中决定直连还是走代理。

# 一些站点规则可以放进 `direct` 里固定，避免走代理：
## 局域网
## 包含敏感信息的未加密网站：登陆页、银行、网购、社交网站、有 IP 认证的网站等。
## 未被封/国内的网站，例如：https://github.com/felixonmars/dnsmasq-china-list/raw/master/accelerated-domains.china.conf

# 永久被封的网站可以放进文件 `blocked`。

# 我是一个注解。 又一个隐式注解。

# 一个主机名
www.google.com

# 域名匹配：自右向左，`.` 是分隔符，无需 `*`，无前导分隔符, 前导 `=` 表示精确匹配。
#
# 所有 `.cn` 的网站。
cn  # *.cn
# *.wikipedia.org
wikipedia.org
# 精确匹配 `gitlab.com`，但是不匹配任何 `*.gitlab.com`。
=gitlab.com

# IP 段匹配：自左向右。`.` 和 `:` 分别是 IPv4 和 IPv6 的分隔符。 分隔符和 `*` 是必需的。
#
# 10.0.0.0-10.255.255.255
10.*
# 192.168.0.0-192.168.255.255.
192.168.*
#
# IPv6 缩写法产生的待例：
# `1111:0000:0000:4444:5555:6666:7777:8888/64`
# = `1111::4444:5555:6666:7777:8888/64`
# = `1111::4444:*` + `1111:0:0:4444:*`
#
# `1111:0:0:4444:*`
# = `1111:0:0:4444::*` + `1111:0:0:4444:not0::`
#
# `1111:0:0:4444::*`
# = `1111:0000:0000:4444:0000:0000:0000:8888/112`
#
# `1111:0:0:4444:not0::`
# = `1111:0000:0000:4444:not0:0000:0000:0000`
#
# Converter: https://github.com/lifenjoiner/iprefix
```

## 局限

网站自己限制（封禁）访问的站点或者路径并不能被识别。

## 统计 Vs 隐私

`pd` 存储主机最近 7 天（`-statvalidity` 的默认值） 的动态统计数据，包括 "host/IP:port"、最近 10 次连接的 EWMA、访问计数和时间。这是动态调度算法必需的。

你可以用 `-statfile` 将结果保存为 `nul`。但是，这样就只能冷重启。

## 主页

https://github.com/lifenjoiner/pd

---

`pd` is a local proxy dispatcher.

## Features

1. Dedicate to be a dispatcher, prepare the upstream proxies yourself,
2. May automatically choose the fatest responsed IP,
3. Automatically yield the availability score, and proxy it if with poor availability,
4. Support protocols socks4a/socks5/http,
5. Support predefined "direct/blocked" list files,
6. Serve as a PAC file server.

## Usage

Try `pd -h` for details.

Examples:

Windows serves only yourself
```bat
start pd.exe -proxies=http://127.0.0.1:1080,http://127.0.0.1:2080,http://127.0.0.1:3080,socks5://127.0.0.1:1081,socks5://127.0.0.1:2081,socks5://127.0.0.1:3081,socks4a://127.0.0.1:2081 -netprobeurl=https://www.toutiao.com
```

*nix serves for the LAN of a router
```sh
pd -proxies=http://127.0.0.1:1080,http://127.0.0.1:2080,http://127.0.0.1:3080,socks5://127.0.0.1:1081,socks5://127.0.0.1:2081,socks5://127.0.0.1:3081,socks4a://127.0.0.1:2081 -netprobeurl=https://www.toutiao.com -listens=192.168.2.1:6699 -direct=/etc/pd/direct -blocked=/etc/pd/blocked -statfile=/tmp/stat.json
```

## Dos
* Static rules: sub-domain first.
* Static rules: `direct` > `blocked`.
* Static `blocked` hosts (IPs) always go proxied.
* Static `direct` hosts (IPs) always go direct.
* General hosts (IPs): go direct for dynamically calculated times, if unsolved, go proxied with 3 tries using the fastest proxies in order; if went proxied directly but no proxy configured, fall back to a direct try.
* Trust your DNS. If the DNS isn't reliable enough, improve it, or place the special hosts in `blocked` file to go proxied directly. For DNS servers, it is suggested to use `0.0.0.0`/`::` or disabled domain list to block hosts, because `127.0.0.1`/`::1` or other reserved IPs are legal to be a server.
* Proxy the requests using the same protocol.

## Don'ts

* Proxy authentication. No need for local proxies.
* Non-Global-Internet-IPs or Non-domain-hosts go to upstream proxies.
* SOCKS BIND and UDP.

## Static Host Matching Syntax

```INI
# The soal of pd is going direct or proxied on the fly.

# Some site rules can be list `direct` as solid, avoid going proxied ever:
## LAN
## Sensitive non-encrypted sites: login, bank, e-shopping, SNS, IP authorizing sites, etc.
## Unblocked/domestic sites, example: https://github.com/felixonmars/dnsmasq-china-list/raw/master/accelerated-domains.china.conf

# The permanently blocked sites can be placed in `blocked` file.

# I'm a comment. And an implicit comment.

# A host.
www.google.com

# Domain match: from right to left, `.` as separator, no `*`, no leading separator, leading `=` means exact match.
#
# All `.cn` sites.
cn  # *.cn
# *.wikipedia.org
wikipedia.org
# Exactly `gitlab.com` without any of `*.gitlab.com`.
=gitlab.com

# IP range match: from left to right. `.` for IPv4, `:` for IPv6 as separator. Separator and `*` are required.
#
# 10.0.0.0-10.255.255.255
10.*
# 192.168.0.0-192.168.255.255.
192.168.*
#
# IPv6 shortening rules yield special cases:
# `1111:0000:0000:4444:5555:6666:7777:8888/64`
# = `1111::4444:5555:6666:7777:8888/64`
# = `1111::4444:*` + `1111:0:0:4444:*`
#
# `1111:0:0:4444:*`
# = `1111:0:0:4444::*` + `1111:0:0:4444:not0::`
#
# `1111:0:0:4444::*`
# = `1111:0000:0000:4444:0000:0000:0000:8888/112`
#
# `1111:0:0:4444:not0::`
# = `1111:0000:0000:4444:not0:0000:0000:0000`
#
# Converter: https://github.com/lifenjoiner/iprefix
```

## Limits

Sites/Pathes restricted (blocked) by the servers self are not detectable.

## Statistics Vs Privacy

`pd` stores dynamic statistics of hosts for the last 7d (`-statvalidity` default), including "host/IP:port" pairs, EWMA of the last 10 connections, visit count, and the last visit time. It is required to implement the dynamic dispatching.

You can use `-statfile` to save them to `nul`. But that will lead to a cold restart.

## Homepage

https://github.com/lifenjoiner/pd
