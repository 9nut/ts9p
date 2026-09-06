// ramfs example from 9fans.net/go/plan9/srv9p, but over tailcat
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"slices"
	"sync"

	"9fans.net/go/plan9"
	"9fans.net/go/plan9/srv9p"
	"github.com/tailscale/tailcat"
)

var (
	verbose = flag.Bool("v", false, "print protocol trace on standard error")
	portnbr = flag.Int("p", 17007, "tcp port number")
)

func tsramfs() {
	s := &tailcat.Server{
		OnTCP: func(port uint16) func(net.Conn) {
			if port != uint16(*portnbr) {
				return nil
			}
			return func(c net.Conn) {
				srv := ramfsServer()
				log.Println("Starting server")
				srv.Serve(c, c)
				c.Close()
			}
		},
	}
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.TailcatAddr())
	select {}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: tsramfs [-v] [-p port]\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("tsramfs: ")
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() != 0 {
		usage()
	}

	tsramfs()
}

// A ramFile is the per-File storage, saved in the File's Aux field.
type ramFile struct {
	mu   sync.Mutex
	data []byte
}

func ramfsServer() *srv9p.Server {
	srv := &srv9p.Server{
		Tree: srv9p.NewTree("ram", "ram", plan9.DMDIR|0755, nil),
		Open: func(ctx context.Context, fid *srv9p.Fid, mode uint8) error {
			if mode&plan9.OTRUNC != 0 {
				rf := fid.File().Aux.(*ramFile)
				rf.mu.Lock()
				defer rf.mu.Unlock()

				rf.data = nil
			}
			return nil
		},
		Create: func(ctx context.Context, fid *srv9p.Fid, name string, perm plan9.Perm, mode uint8) (plan9.Qid, error) {
			f, err := fid.File().Create(name, "ram", perm, new(ramFile))
			if err != nil {
				return plan9.Qid{}, err
			}
			fid.SetFile(f)
			return f.Stat.Qid, nil
		},
		Read: func(ctx context.Context, fid *srv9p.Fid, data []byte, offset int64) (int, error) {
			rf := fid.File().Aux.(*ramFile)
			rf.mu.Lock()
			defer rf.mu.Unlock()

			return fid.ReadBytes(data, offset, rf.data)
		},
		Write: func(ctx context.Context, fid *srv9p.Fid, data []byte, offset int64) (int, error) {
			rf := fid.File().Aux.(*ramFile)
			rf.mu.Lock()
			defer rf.mu.Unlock()

			if int64(int(offset)) != offset || int(offset)+len(data) < 0 {
				return 0, srv9p.ErrBadOffset
			}
			end := int(offset) + len(data)
			if len(rf.data) < end {
				rf.data = slices.Grow(rf.data, end-len(rf.data))
				rf.data = rf.data[:end]
			}
			copy(rf.data[offset:], data)
			return len(data), nil
		},
	}
	if *verbose {
		srv.Trace = os.Stderr
	}

	return srv
}
