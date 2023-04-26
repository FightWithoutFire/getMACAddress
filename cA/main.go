package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var log = logrus.New()
var host string

func main() {
	flag.StringVar(&host, "M", "", "Get mac address")

	flag.Parse()
	// 初始化 代理
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	go listenHttps(ctx, host, client)
	go listenHttp(ctx, host, client)

	// graceful terminate
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sig:
		cancel()
		fmt.Printf("graceful terminate...\n")
	}

}

func listenHttp(ctx context.Context, host string, client *http.Client) {

	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_ = r.Body.Close()
			}()

			_, _ = w.Write([]byte(fmt.Sprintf("http Computer B MAC is %s", getMacAddress(client, host))))
		}),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	srv, err := net.Listen("tcp", "127.0.0.1:8855")
	if err != nil {
		log.Fatal(err)
	}

	err = server.Serve(srv)
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()

}

func getMacAddress(client *http.Client, host string) []byte {
	resp, err := client.Get(fmt.Sprintf("http://%s", host))
	if err != nil {
		log.Error(err)
		return nil
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	all, err := io.ReadAll(resp.Body)

	if err != nil {
		log.Error(err)
		return nil
	}
	return all
}

func listenHttps(ctx context.Context, host string, client *http.Client) {

	server := http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				_ = r.Body.Close()
			}()

			_, _ = w.Write([]byte(fmt.Sprintf("https Computer B MAC is %s", getMacAddress(client, host))))
		}),
		IdleTimeout:       5 * time.Minute,
		ReadHeaderTimeout: time.Minute,
	}

	srvHttps, err := net.Listen("tcp", "127.0.0.1:8854")
	if err != nil {
		log.Fatal(err)
	}

	err = server.ServeTLS(srvHttps, "./domain.crt", "./domain.key")
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()

}
