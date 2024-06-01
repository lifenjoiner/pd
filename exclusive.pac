function FindProxyForURL(url, host) {
    /* make the special sites go to the exclusive networks */
    var i = host.lastIndexOf('.');
    if (i == -1) return 'DIRECT';
    switch (host.slice(i).toLowerCase()) {
    case '.onion':
        return 'SOCKS5 127.0.0.1:9050';
    case '.i2p':
        return 'HTTP 127.0.0.1:4444';
    }
    return 'PROXY 127.0.0.1:6699';
}
