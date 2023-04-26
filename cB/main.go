package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// 本机的mac地址
var localHaddr net.HardwareAddr
var iface string

func setupNetInfo(f string) {
	var ifs []net.Interface
	var err error
	if f == "" {
		ifs, err = net.Interfaces()
	} else {
		// 已经选择iface
		var it *net.Interface
		it, err = net.InterfaceByName(f)
		if err == nil {
			ifs = append(ifs, *it)
		}
	}
	if err != nil {
		log.Fatal("无法获取本地网络信息:", err)
	}
	for _, it := range ifs {
		addr, _ := it.Addrs()
		for _, a := range addr {
			if ip, ok := a.(*net.IPNet); ok && !ip.IP.IsLoopback() {
				if ip.IP.To4() != nil {
					localHaddr = it.HardwareAddr
					iface = it.Name
					goto END
				}
			}
		}
	}
END:
	if len(localHaddr) == 0 {
		log.Fatal("无法获取本地网络信息")
	}
}

func main() {
	flag.StringVar(&iface, "I", "", "Network interface name")

	flag.Parse()

	// 初始化 网络信息
	setupNetInfo(iface)

	ctx, cancel := context.WithCancel(context.Background())

	go listenProxy(ctx)
	go listenHttp()
	go listenHttps()

	// graceful terminate
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	fmt.Printf("graceful terminate...\n")

}

func listenHttps() {

	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_ = r.Body.Close()
			}()

			_, _ = w.Write([]byte("https B mac address is" + localHaddr.String()))
		}),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	srv, err := net.Listen("tcp", "127.0.0.1:8443")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("https server start")
	err = server.ServeTLS(srv, "./server.crt", "./server.key")

	if err != nil {
		log.Fatal(err)
	}

}

func listenHttp() {

	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_ = r.Body.Close()
			}()

			_, _ = w.Write([]byte("http B mac address is" + localHaddr.String()))
		}),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	srv, err := net.Listen("tcp", "127.0.0.1:8081")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("http server start")
	err = server.Serve(srv)
	if err != nil {
		log.Fatal(err)
	}

}

func listenProxy(ctx context.Context) {

	srv, err := net.Listen("tcp", ":9333")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("proxy server start")

	go func() {

		for {
			conn, err := srv.Accept()

			log.Println("accept connection")
			if err != nil {
				log.Error(err)
			}

			go func(from net.Conn) {
				defer func() {
					_ = from.Close()
				}()
				log.Println("reading")

				msg := make([]byte, 1)

				_, err := conn.Read(msg)

				if err != nil {
					log.Error(err)
				}

				log.Printf("proxy B receive msg: %s\n", msg)

				if len(msg) > 0 {

					var dest net.Conn
					if msg[0] == 0x16 {
						log.Println("try dial https")

						dest, err = net.Dial("tcp", "127.0.0.1:8443")
						log.Printf("success dial https")

						if err != nil {
							log.Error(err)
							return
						}
					} else {
						log.Println("try dial http")

						dest, err = net.Dial("tcp", "127.0.0.1:8081")
						log.Printf("success dial http")

						if err != nil {
							log.Error(err)
							return
						}

					}

					defer func() {
						_ = dest.Close()
					}()

					_, err := dest.Write(msg)
					if err != nil {
						log.Error(err)
					}

					err = proxy(dest, from)
					if err != nil && err != io.EOF {
						log.Error(err)

					}

				}
			}(conn)

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
