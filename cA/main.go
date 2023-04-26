package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

var log = logrus.New()
var host string

func main() {
	flag.StringVar(&host, "M", "", "Get mac address")

	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	go listen(ctx, host)

	// graceful terminate
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sig:
		cancel()
		fmt.Printf("graceful terminate...\n")
	}

}

func listen(ctx context.Context, host string) {

	srvTCP, err := net.Listen("tcp", "127.0.0.1:8855")

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := srvTCP.Accept()
			if err != nil {
				log.Fatal(err)
			}

			go func(from net.Conn) {

				defer func() {
					_ = from.Close()
				}()

				log.Printf("try client http to %s", host)
				client, err := net.Dial("tcp", host)
				log.Printf("success client https to %s", host)

				if err != nil {
					return
				}

				defer func() {
					_ = client.Close()
				}()

				err = proxy(client, from)

				if err != nil && err != io.EOF {
					log.Error(err)
				}
				log.Printf("close client https to %s", host)

			}(conn)

			//_, err = conn.Write([]byte("http Computer B MAC is " + string(getMacAddress(client, host))))
		}
	}()

	srvTLS, err := net.Listen("tcp", "127.0.0.1:8854")

	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			conn, err := srvTLS.Accept()
			if err != nil {
				log.Fatal(err)
			}

			go func(from net.Conn) {

				defer func() {
					_ = from.Close()
				}()

				log.Printf("try client https to %s", host)
				client, err := net.Dial("tcp", host)
				log.Printf("success client https to %s", host)

				if err != nil {
					return
				}

				defer func() {
					_ = client.Close()
				}()

				err = proxy(client, from)

				if err != nil && err != io.EOF {
					log.Error(err)

				}
				log.Printf("close client https to %s", host)

			}(conn)

			//_, err = conn.Write([]byte("http Computer B MAC is " + string(getMacAddress(client, host))))
		}
	}()

	<-ctx.Done()

}

func proxy(w io.Writer, r io.Reader) error {
	reader, isReader := w.(io.Reader)
	writer, isWriter := r.(io.Writer)

	if isWriter && isReader {
		go func() {
			_, _ = io.Copy(writer, reader)
		}()
	}

	_, err := io.Copy(w, r)
	return err
}
