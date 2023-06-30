package main

import (
	"log"
	"net"
	"net/http"
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
		remoteCloser := &OnceCloser{Closer: remote}
		defer remoteCloser.Close()

		w.Header().Add(`Content-Length`, `0`)
		w.WriteHeader(http.StatusSwitchingProtocols)

		local, bio, err := w.(http.Hijacker).Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		localCloser := &OnceCloser{Closer: local}
		defer localCloser.Close()

		log.Println(r.RemoteAddr, `->`, target, `connected`)
		defer log.Println(r.RemoteAddr, `->`, target, `closed`)

		if err := bio.Writer.Flush(); err != nil {
			return
		}

		bridge(remote, remoteCloser, local, bio.Reader, localCloser, key)
	})
	http.ListenAndServe(listen, nil)
}
