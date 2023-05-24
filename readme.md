`pd` is a local proxy dispatcher.

## Usage
Try `pd -h`.

## Dos
* Sub-domain rule first.
* Static `blocked` hosts (IPs) go proxied directly.
* Static `direct` hosts (IPs) always go direct (Never proxied). `direct` > `blocked`.
* General hosts, go direct as dynamically calculated times, if unsolved, go proxied with 3 tries using the fastest proxies in order, fall back to a direct try if directly went proxied but no proxy configured.
* Trust the specified DNS. If the DNS isn't reliable enough, improve it, or place the special host in `blocked` to go proxied directly. For DNS servers, use `0.0.0.0`/`::` or disabled domain list to block hosts, `127.0.0.1`/`::1` or other reserved IPs are legal to be a server.
* Forward going proxied requests to the same protocol proxy. `socks4a` is a super set of `socks4`.

## Don'ts
* Proxy authentication. No need for local proxies.
* Non-Global-Internet-IPs or No-domain-hosts go to upstream proxies.

## Static Host Matching Syntax

```INI
# The soal of pd is going direct or proxied on the fly.

# Some site rules can be list `direct` as solid, avoiding them go proxied ever:
## LAN
## Sensitive non-encrypted sites: login, bank, e-shopping, SNS, IP autherizing sites, etc.
## Unblocked/domestic sites, example: https://github.com/felixonmars/dnsmasq-china-list/raw/master/accelerated-domains.china.conf

# The permanently blocked sites can be placed in `blocked` file.

# I'm a comment. And an implicit comment.

# Exact match.
www.google.com

## domain match: from right to left, `.` as separator, no `*`, no separator, leading `=` means exact match.
# All `.cn` sites.
cn  # *.cn
# *.wikipedia.org
wikipedia.org
# exactly `gitlab.com` without any of `*.gitlab.com`
=gitlab.com

## ip range match: from left to right. `.` for ipv4, `:` for ipv6 as separator. Separator and `*` are required.
# 10.0.0.0-10.255.255.255
10.*
# 192.168.0.0-192.168.255.255.
192.168.*
```

## Statistics Vs Privacy

`pd` stores the dynamic statistics of hosts for the last 24h, including "host/IP:port" pairs, EWMA of the last 10 connections, visit count, and the last visit time. It is required to implement the dynamic dispatching.

You can use `-statfile` to save them to `nul`. But that will lead the application restart as a cold startup.

If you concern privacy very much, `pd` is not for you.

## Homepage

https://github.com/lifenjoiner/pd
