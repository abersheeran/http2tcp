package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

func client(listen string, server string, token string, to string) {
	lis, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalln(err)
	}
	defer lis.Close()

	auth := token[:3]
	key := GenerateKey(token)

	for {
		conn, err := lis.Accept()
		if err != nil {
			time.Sleep(time.Second * 5)
			continue
		}
		go func(local io.ReadWriteCloser) {
			closeLocal := &sync.Once{}
			defer closeLocal.Do(func() { local.Close() })
			remote, bodyReader, err := CreateProxyConnection(server, auth, key, to)
			if err != nil {
				log.Println(err.Error())
				return
			}
			closeRemote := &sync.Once{}
			defer closeRemote.Do(func() { remote.Close() })

			var wg sync.WaitGroup
			wg.Add(2)

			go func() {
				defer wg.Done()
				defer closeLocal.Do(func() { local.Close() })
				defer closeRemote.Do(func() { remote.Close() })

				if n := bodyReader.Buffered(); n > 0 {
					buffer := make([]byte, n)
					if _, err := bodyReader.Read(buffer); err != nil {
						return
					}
					data, err := EncryptStream(buffer, key)
					if err != nil {
						return
					}
					if _, err := local.Write(data); err != nil {
						return
					}
				}

				buffer := make([]byte, 32*1024)

				for {
					n, err := remote.Read(buffer)
					if err != nil {
						return
					}
					data, err := DecryptStream(buffer[:n], key)
					if err != nil {
						return
					}
					if _, err := local.Write(data); err != nil {
						return
					}

				}
			}()

			go func() {
				defer wg.Done()
				defer closeLocal.Do(func() { local.Close() })
				defer closeRemote.Do(func() { remote.Close() })

				buffer := make([]byte, 32*1024)

				for {
					n, err := local.Read(buffer)
					if err != nil {
						return
					}
					data, err := EncryptStream(buffer[:n], key)
					if err != nil {
						return
					}
					if _, err := remote.Write(data); err != nil {
						return
					}
				}
			}()

			wg.Wait()
		}(conn)
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
