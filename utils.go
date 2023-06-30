package main

import (
	"bufio"
	"io"
	"os"
	"sync"
)

type OnceCloser struct {
	io.Closer
	once sync.Once
}

func (c *OnceCloser) Close() (err error) {
	c.once.Do(func() {
		err = c.Closer.Close()
	})
	return
}

type StdReadWriteCloser struct {
	io.ReadCloser
	io.WriteCloser
}

func NewStdReadWriteCloser() *StdReadWriteCloser {
	return &StdReadWriteCloser{
		ReadCloser:  os.Stdin,
		WriteCloser: os.Stdout,
	}
}

func (c *StdReadWriteCloser) Close() error {
	err1 := c.ReadCloser.Close()
	err2 := c.WriteCloser.Close()

	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}

	return nil
}

func bridge(rwc io.ReadWriteCloser, rwcCloser *OnceCloser,
	httpConnection io.ReadWriteCloser, bodyReader *bufio.Reader, closeRemote *OnceCloser,
	key []byte) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer rwcCloser.Close()
		defer closeRemote.Close()

		if n := bodyReader.Buffered(); n > 0 {
			buffer := make([]byte, n)
			if _, err := bodyReader.Read(buffer); err != nil {
				return
			}
			data, err := EncryptStream(buffer, key)
			if err != nil {
				return
			}
			if _, err := rwc.Write(data); err != nil {
				return
			}
		}

		buffer := make([]byte, 32*1024)

		for {
			n, err := httpConnection.Read(buffer)
			if err != nil {
				return
			}
			data, err := DecryptStream(buffer[:n], key)
			if err != nil {
				return
			}
			if _, err := rwc.Write(data); err != nil {
				return
			}

		}
	}()

	go func() {
		defer wg.Done()
		defer rwcCloser.Close()
		defer closeRemote.Close()

		buffer := make([]byte, 32*1024)

		for {
			n, err := rwc.Read(buffer)
			if err != nil {
				return
			}
			data, err := EncryptStream(buffer[:n], key)
			if err != nil {
				return
			}
			if _, err := httpConnection.Write(data); err != nil {
				return
			}
		}
	}()

	wg.Wait()
}
