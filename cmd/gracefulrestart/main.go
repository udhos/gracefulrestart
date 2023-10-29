// Package main implements the tool.
package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type config struct {
	addr    string
	message string
}

type application struct {
	server *http.Server
}

func main() {

	app := &application{
		server: &http.Server{},
	}

	go func() {
		const interval = 3 * time.Second
		for i := 1; ; i++ {
			log.Printf("reloader %d: reloading", i)
			load(app, i, "config.txt")
			log.Printf("reloader %d: sleeping for %v", i, interval)
			time.Sleep(interval)
		}
	}()

	//
	// handle graceful shutdown
	//

	shutdown(app)
}

func load(app *application, i int, path string) {
	me := fmt.Sprintf("load %d", i)

	cfg, errConf := loadConfig(path)
	if errConf != nil {
		log.Printf("%s: load config: %v", me, errConf)
		return
	}

	addr := cfg.addr

	//
	// launch new server
	//
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.NotFound)
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) { handlerHello(w, r, cfg) })
	server := http.Server{
		Addr:    addr,
		Handler: mux,
	}
	lc := net.ListenConfig{
		Control: setSocketOpt,
	}
	ln, errListen := lc.Listen(context.TODO(), "tcp", addr)
	if errListen != nil {
		log.Printf("%s: error listening on %s: %v", me, addr, errListen)
		return
	}
	log.Printf("%s: listening on %s", me, addr)
	go func() {
		errServe := server.Serve(ln)
		log.Printf("%s: error serving on %s: %v", me, addr, errServe)
	}()

	//
	// shutdown old server
	//
	httpShutdown(app.server)

	//
	// replace server
	//
	app.server = &server
}

func setSocketOpt(network, address string, c syscall.RawConn) error {
	var opErr1, opErr2 error
	err := c.Control(func(fd uintptr) {
		opErr1 = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		opErr2 = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
	})
	if err != nil {
		return err
	}
	if opErr1 != nil {
		return opErr1
	}
	if opErr2 != nil {
		return opErr2
	}
	return nil
}

func handlerHello(w http.ResponseWriter, r *http.Request, cfg config) {
	http.Error(w, cfg.message, 200)
}

func loadConfig(path string) (config, error) {
	cfg := config{}
	buf, errRead := os.ReadFile(path)
	if errRead != nil {
		return cfg, errRead
	}
	cfg.addr = ":8080"
	cfg.message = string(buf)
	return cfg, nil
}

func shutdown(app *application) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("received signal '%v', initiating shutdown", sig)

	httpShutdown(app.server)

	log.Printf("exiting")
}

func httpShutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("http server shutdown error: %v", err)
	}
}
