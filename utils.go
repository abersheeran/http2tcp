package main

import (
	"bufio"
	"io"
	"log"
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

func bridge(
	rwc io.ReadWriteCloser, rwcCloser *OnceCloser,
	httpConnection io.ReadWriteCloser, bodyReader *bufio.Reader, closeRemote *OnceCloser,
) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer rwcCloser.Close()

		if n := int64(bodyReader.Buffered()); n > 0 {
			if nc, err := io.CopyN(rwc, bodyReader, n); err != nil || nc != n {
				log.Println("io.CopyN:", nc, err)
				return
			}
		}

		_, _ = io.Copy(rwc, httpConnection)
	}()

	go func() {
		defer wg.Done()
		defer closeRemote.Close()

		_, _ = io.Copy(httpConnection, rwc)
	}()

	wg.Wait()
}
