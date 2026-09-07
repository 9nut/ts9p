//go:build plan9

//
// Use tailcat and exportfs to set up an
// adhoc secure 9P file server over the internet.
//
// See TailScale tailcat for a full description:
// https://tailscale.com/tailcat
//

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"

	"github.com/tailscale/tailcat"
	"tailscale.com/types/logger"
)

var (
	portnbr = flag.Int("p", 17007, "port number to listen on")
	patf    = flag.String("P", "", "pattern file")
	root    = flag.String("r", "./", "root of directory to serve")
	msize   = 1280 // Tailscale MTU (according the internet lore)
)

func main() {
	flag.Parse()

	s := &tailcat.Server{
		Logf: logger.Discard,
		OnTCP: func(port uint16) func(net.Conn) {
			log.Println("OnTCP called")
			if port != uint16(*portnbr) {
				log.Println("OnTCP not a port we want")
				return nil
			}
			return func(c net.Conn) {
				// make a pipe
				// connect c to p[0] and p[1] to exportfs stdin/stdout
				// start exportfs with
				// -m 1280		// MTU for tailscale
				// -P patternfile	// inclusions and exlusions
				// -R			// read-only
				// -r rootdir		// root of directory to serve

				var pip [2]int
				err := syscall.Pipe(pip[:])
				if err != nil {
					log.Fatal(err)
				}

				f0, f1 := os.NewFile(uintptr(pip[0]), "|0"), os.NewFile(uintptr(pip[1]), "|1")
				defer f0.Close()

				var attr os.ProcAttr
				attr.Files = []*os.File{f1}

				args := []string{"exportfs", "-m", fmt.Sprint("%d", msize), "-r", *root, "-R"}
				if *patf != "" {
					args = append(args, "-P", *patf)
				}

				log.Println("StartProcess exportfs")
				proc, err := os.StartProcess("/bin/exportfs", args, &attr)
				if err != nil {
					log.Fatal(err)
				}

				defer func() {
					err := proc.Kill()
					log.Printf("kill %v: %v\n", args, err)
				}()

				f1.Close()
				log.Println("Start copying...")
				go io.Copy(c, f0)
				io.Copy(f0, c)
				c.Close()
				log.Println("Done with OnTCP")
			}
		},
	}
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(s.TailcatAddr())
	select {}
}
