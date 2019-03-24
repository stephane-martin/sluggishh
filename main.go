package main

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/urfave/cli"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"
)

var Version string

type IdiotSet struct {
	m  map[string]int
	mu sync.Mutex
}

func NewIdiotSet() *IdiotSet {
	return &IdiotSet{
		m: make(map[string]int),
	}
}

func (s *IdiotSet) Add(idiot string) {
	s.mu.Lock()
	s.m[idiot]++
	s.mu.Unlock()
}

func (s *IdiotSet) Sub(idiot string) {
	s.mu.Lock()
	s.m[idiot]--
	s.mu.Unlock()
}

func (s *IdiotSet) Len() int {
	s.mu.Lock()
	var l int
	for _, nb := range s.m {
		if nb != 0 {
			l++
		}
	}
	s.mu.Unlock()
	return l
}

func main() {
	app := cli.NewApp()
	app.Name = "sluggissh"
	app.Usage = "slow fake SSH"
	app.Description = "fake SSH server that just make attackers lose time"
	app.Version = Version
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:   "port,p",
			Usage:  "listen port",
			Value:  2222,
			EnvVar: "SLUGGISSH_PORT",
		},
		cli.StringFlag{
			Name:   "addr,a",
			Usage:  "listen address",
			Value:  "127.0.0.1",
			EnvVar: "SLUGGISH_ADDR",
		},
		cli.IntFlag{
			Name:   "delay,d",
			Usage:  "message delay in seconds",
			Value:  10,
			EnvVar: "SLUGGISSH_DELAY",
		},
		cli.StringFlag{
			Name:  "loglevel",
			Usage: "logging level",
			Value: "debug",
		},
		cli.IntFlag{
			Name:   "length",
			Usage:  "maximum banner line length",
			Value:  32,
			EnvVar: "SLUGGISSH_LENGTH",
		},
	}
	app.Action = Sluggissh
	_ = app.Run(os.Args)
}

func Sluggissh(c *cli.Context) (e error) {
	defer func() {
		if e != nil {
			e = cli.NewExitError(e.Error(), 1)
		}
	}()
	loglevel := c.GlobalString("loglevel")
	lvl, err := log15.LvlFromString(loglevel)
	if err != nil {
		lvl = log15.LvlDebug
	}
	logger := log15.New()
	logger.SetHandler(log15.LvlFilterHandler(lvl, log15.StderrHandler))
	reporter := log15.New()
	reporter.SetHandler(log15.StreamHandler(os.Stdout, log15.JsonFormat()))
	addr := c.GlobalString("addr")
	port := c.GlobalInt("port")
	if port < 0 {
		port = 2222
	}
	delay := c.GlobalInt("delay")
	if delay < 0 {
		delay = 10
	}
	length := c.GlobalInt("length")
	if length < 3 {
		length = 3
	}
	if length > 255 {
		length = 255
	}
	go feed(uint32(length))
	logger.Debug("parameters", "addr", addr, "port", port, "delay", delay)
	ctx, cancel := context.WithCancel(context.Background())
	g, lctx := errgroup.WithContext(ctx)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for range sigChan {
			cancel()
		}
	}()
	listener, err := net.Listen("tcp", net.JoinHostPort(addr, fmt.Sprintf("%d", port)))
	if err != nil {
		return err
	}
	go func() {
		<-lctx.Done()
		_ = listener.Close()
	}()
	var nbConns atomic.Int32
	idiots := NewIdiotSet()
	g.Go(func() error {
		for {
			logger.Info("number of distinct remote addresses", "nb_addrs", idiots.Len())
			select {
			case <-lctx.Done():
				return nil
			case <-time.After(time.Minute):
			}
		}
	})
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		finished := make(chan struct{})
		go func() {
			select {
			case <-lctx.Done():
			case <-finished:
			}
			_ = conn.Close()
		}()
		g.Go(func() error {
			Handle(lctx, conn, idiots, &nbConns, time.Duration(delay)*time.Second, reporter, logger)
			close(finished)
			return nil
		})
	}
}

func Handle(ctx context.Context, conn net.Conn, idiots *IdiotSet, nbConns *atomic.Int32, delay time.Duration, reporter, logger log15.Logger) {
	remoteAddr := conn.RemoteAddr()
	remote := remoteAddr.String()
	tcpAddr, ok := remoteAddr.(*net.TCPAddr)
	if ok {
		remote = tcpAddr.IP.String()
	}
	idiots.Add(remote)
	nb := nbConns.Inc()
	logger.Info("new connection", "addr", remote, "nb_connections", nb)
	defer func() {
		nbConns.Dec()
		idiots.Sub(remote)
		logger.Info("connection closed", "addr", remote)
	}()
	select {
	case <-ctx.Done():
		return
	case <-time.After(delay / 2):
	}
	for {
		// TODO: record number of sent bytes
		_, err := conn.Write(RandomString())
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
	}
}

var randCh = make(chan []byte)

func RandomString() []byte {
	return <-randCh
}

var SSHB = []byte("SSH-")

func feed(maxLen uint32) {
	var i uint32
	r := rand.New(rand.NewSource(0))
	for {
		length := 3 + r.Uint32()%(maxLen-2)
		line := make([]byte, int(length))
		for i = 0; i < length-2; i++ {
			line[i] = byte(32 + (r.Uint32() % 95))
		}
		line[length-2] = 13
		line[length-1] = 10
		if length >= 4 {
			if bytes.Equal(line[:4], SSHB) {
				line[0] = 'X'
			}
		}
		randCh <- line
	}
}
