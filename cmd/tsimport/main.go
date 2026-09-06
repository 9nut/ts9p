//go:build plan9

//
// Import a 9P file system served over tailcat
//
// See TailScale tailcat for a full description:
// https://tailscale.com/tailcat
// 

package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/tailscale/tailcat"
)

var (
	port = flag.Int("p", 17007, "port number dial")
	mntpt = flag.String("m", "/n/ts9p", "mountpoint")
)

func main() {

	flag.Parse()
	if *mntpt == "" {
		*mntpt = "/n/ts9p"
	}

	if flag.Arg(0) == "" {
		log.Fatal("tailcat address is missing")
	}

	cl := tailcat.NewClient(tailcat.Addr(flag.Arg(0)))
	defer cl.Close()
	c, err := cl.DialTCPPort(context.Background(), uint16(*port))
	if err != nil {
		log.Fatal(err)
	}

	var pip [2]int
	err = syscall.Pipe(pip[:])
	if err != nil {
		log.Fatal(err)
	}
	f0, f1 := os.NewFile(uintptr(pip[0]), "|0"), os.NewFile(uintptr(pip[1]), "|1")
	
	// connect the ts channel to one side of the pipe
	defer f0.Close()
	go io.Copy(f0, c)
	go io.Copy(c, f0)

	runtime.LockOSThread()

	// mount the other side of the pipe on the mount point
	err = syscall.Mount(int(f1.Fd()), -1, *mntpt, syscall.MCREATE|syscall.MBEFORE, "")
	if err != nil {
		log.Fatal(err)
	}
	f1.Close()

	log.Printf("The imported name space is mounted on %s\nDropping into a new rc session", *mntpt)
	cmd := exec.Command("rc", "-i")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err = cmd.Run(); err != nil {
		log.Println("rc:", err)
	}

	runtime.UnlockOSThread()
}

