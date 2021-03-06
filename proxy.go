package main

import (
	"io"
	"net"
)

type Proxy struct {
	id       string
	isServer bool
	inited   bool
	checked  bool
	usedChan chan bool
	from     net.Conn
	to       net.Conn
	init     func(net.Conn)
	valid    func([]byte) bool
	closed   func(string)
	onData   func(string, []byte)
}

func (proxy *Proxy) pipe(src, dst io.ReadWriter, direct bool) {
	if direct && !proxy.isServer {
		counter.Up()
	}
	buff := make([]byte, 0xffff)
	defer func() {
		if direct && !proxy.isServer {
			counter.Down()
		}
		proxy.from.Close()
		proxy.to.Close()
		if proxy.closed != nil {
			proxy.closed(proxy.id)
		}
		proxy.usedChan <- true
	}()
	for {
		if direct && !proxy.inited {
			proxy.inited = true
			if proxy.init != nil {
				proxy.init(src.(net.Conn))
			}
		}
		size, err := src.Read(buff)
		if err != nil {
			break
		}
		data := buff[:size]
		if direct && !proxy.checked {
			proxy.checked = true
			proxy.usedChan <- true
			if proxy.valid != nil && !proxy.valid(data) {
				break
			}
		}
		if _, err = dst.Write(data); err != nil {
			break
		}
		if proxy.onData != nil {
			go proxy.onData(proxy.id, data)
		}
	}
}

func (proxy *Proxy) Close() {
	TryClose(proxy.from)
	TryClose(proxy.to)
}

func (proxy *Proxy) Start(isServer bool) chan bool {
	proxy.isServer = isServer
	proxy.usedChan = make(chan bool)

	go proxy.pipe(proxy.from, proxy.to, true)
	go proxy.pipe(proxy.to, proxy.from, false)

	return proxy.usedChan
}
