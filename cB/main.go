package main

import (
	"context"
	"flag"
	"fmt"
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

	go listenHttp(ctx)

	// graceful terminate
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	cancel()
	fmt.Printf("graceful terminate...\n")

}

func listenHttp(ctx context.Context) {

	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_ = r.Body.Close()
			}()

			_, _ = w.Write([]byte(localHaddr.String()))
		}),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	srv, err := net.Listen("tcp", ":9333")
	if err != nil {
		log.Fatal(err)
	}

	err = server.Serve(srv)
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()

}
