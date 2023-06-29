package main

import (
	"log"
	"net"
	"net/http"
	"sync"
)

const (
	authHeader        = `X-Http2tcp-Auth`
	httpHeaderUpgrade = `http2tcp/1.0`
)

func server(listen string, token string) {
	auth := token[:3]
	key := GenerateKey(token)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != httpHeaderUpgrade {
			log.Println(r.RemoteAddr, `upgrade failed`)
			http.Error(w, `Upgrade failed`, http.StatusBadRequest)
			return
		}

		if r.Header.Get(authHeader) != auth {
			log.Println(r.RemoteAddr, `auth failed`)
			http.Error(w, `Auth failed`, http.StatusUnauthorized)
			return
		}

		target, err := DecryptFromBase64(r.URL.Query().Get("target"), key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		remote, err := net.Dial(`tcp`, target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		closeRemote := &sync.Once{}
		defer closeRemote.Do(func() { remote.Close() })

		w.Header().Add(`Content-Length`, `0`)
		w.WriteHeader(http.StatusSwitchingProtocols)

		local, bio, err := w.(http.Hijacker).Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		closeLocal := &sync.Once{}
		defer closeLocal.Do(func() { local.Close() })

		log.Println(r.RemoteAddr, `->`, target, `connected`)
		defer log.Println(r.RemoteAddr, `->`, target, `closed`)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			defer closeLocal.Do(func() { local.Close() })
			defer closeRemote.Do(func() { remote.Close() })

			if n := bio.Reader.Buffered(); n > 0 {
				buffer := make([]byte, n)
				if _, err := bio.Reader.Read(buffer); err != nil {
					return
				}

				data, err := DecryptStream(buffer, key)
				if err != nil {
					return
				}
				if _, err := remote.Write(data); err != nil {
					return
				}

			}

			buffer := make([]byte, 32*1024)

			for {
				n, err := local.Read(buffer)
				if err != nil {
					return
				}

				data, err := DecryptStream(buffer[:n], key)
				if err != nil {
					return
				}
				if _, err := remote.Write(data); err != nil {
					return
				}

			}
		}()

		go func() {
			defer wg.Done()
			defer closeLocal.Do(func() { local.Close() })
			defer closeRemote.Do(func() { remote.Close() })

			if err := bio.Writer.Flush(); err != nil {
				return
			}

			buffer := make([]byte, 32*1024)

			for {
				n, err := remote.Read(buffer)
				if err != nil {
					return
				}

				data, err := EncryptStream(buffer[:n], key)
				if err != nil {
					return
				}
				if _, err := local.Write(data); err != nil {
					return
				}

			}
		}()

		wg.Wait()
	})
	http.ListenAndServe(listen, nil)
}
