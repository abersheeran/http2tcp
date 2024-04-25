package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"
)

func client(listen string, server string, token string, to string) {
	auth := token[:3]
	key := GenerateKey(token)

	if listen == `-` {
		local := NewStdReadWriteCloser()
		localCloser := &OnceCloser{Closer: local}
		defer localCloser.Close()

		remote, bodyReader, err := CreateProxyConnection(server, auth, key, to)
		if err != nil {
			log.Println(err.Error())
			return
		}
		remoteCloser := &OnceCloser{Closer: remote}
		defer remoteCloser.Close()

		bridge(local, localCloser, remote, bodyReader, remoteCloser)
	} else {
		lis, err := net.Listen("tcp", listen)
		if err != nil {
			log.Fatalln(err)
		}
		defer lis.Close()

		for {
			conn, err := lis.Accept()
			if err != nil {
				time.Sleep(time.Second * 5)
				continue
			}

			go func(local net.Conn) {
				localCloser := &OnceCloser{Closer: local}
				defer localCloser.Close()

				remote, bodyReader, err := CreateProxyConnection(server, auth, key, to)
				if err != nil {
					log.Println(err.Error())
					return
				}
				remoteCloser := &OnceCloser{Closer: remote}
				defer remoteCloser.Close()

				bridge(local, localCloser, remote, bodyReader, remoteCloser)
			}(conn)
		}
	}
}

func CreateProxyConnection(server string, auth string, key []byte, target string) (net.Conn, *bufio.Reader, error) {
	u, err := url.Parse(server)
	if err != nil {
		return nil, nil, err
	}
	host := u.Hostname()
	port := u.Port()
	if port == `` {
		switch u.Scheme {
		case `http`:
			port = "80"
		case `https`:
			port = `443`
		default:
			return nil, nil, fmt.Errorf(`unknown scheme: %s`, u.Scheme)
		}
	}
	serverAddr := net.JoinHostPort(host, port)

	var remote net.Conn
	switch u.Scheme {
	case `http`:
		remote, err = net.Dial(`tcp`, serverAddr)
		if err != nil {
			return nil, nil, err
		}
	case `https`:
		remote, err = tls.Dial(`tcp`, serverAddr, nil)
		if err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, fmt.Errorf(`unknown scheme: %s`, u.Scheme)
	}

	v := u.Query()
	to, err := EncryptAndBase64(target, key)
	if err != nil {
		return nil, nil, err
	}
	v.Set(`target`, to)
	u.RawQuery = v.Encode()

	req, err := http.NewRequest(http.MethodPost, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Add(`Connection`, `upgrade`)
	req.Header.Add(`Upgrade`, httpHeaderUpgrade)
	req.Header.Add(authHeader, auth)
	req.Header.Add(`User-Agent`, `http2tcp`)

	if err := req.Write(remote); err != nil {
		return nil, nil, err
	}
	bior := bufio.NewReader(remote)
	resp, err := http.ReadResponse(bior, req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		buf := bytes.NewBuffer(nil)
		resp.Write(buf)
		return nil, nil, fmt.Errorf("status code != 101:\n%s", buf.String())
	}

	return remote, bior, nil
}
