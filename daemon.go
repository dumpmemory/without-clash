package main

import (
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"

	"golang.org/x/sys/unix"

	"github.com/Kr328/without-clash/cgroup"
	"github.com/Kr328/without-clash/iproute2"
)

const (
	daemonArg0       = "without-clash-daemon"
	cgroupSubtree    = "without-clash"
	offsetOfMark     = 16
	withoutClashMark = 114514
)

func daemon() {
	if unix.Getuid() != 0 && unix.Geteuid() != 0 {
		abort("Daemon must run as root")
	}

	if unix.Getuid() != 0 && unix.Geteuid() == 0 {
		err := unix.Setuid(0)
		if err != nil {
			abort("Unable to evaluate to root: %s", err.Error())
		}
	}

	if !cgroup.IsVersion2() {
		abort("cgroup2 required")
	}

	has, err := cgroup.HasSubtree(cgroupSubtree)
	if err != nil {
		abort("Detect cgroup subtree: %s", err.Error())
	}

	if !has {
		err = cgroup.CreateSubtree(cgroupSubtree)
		if err != nil {
			abort("Create cgroup subtree: %s", err.Error())
		}
		defer func() {
			_ = cgroup.DeleteSubtree(cgroupSubtree)
		}()
	}

	progSpec := &ebpf.ProgramSpec{
		Name:       "without_clash",
		Type:       ebpf.CGroupSock,
		AttachType: ebpf.AttachCGroupInetSockCreate,
		Instructions: asm.Instructions{
			asm.LoadImm(asm.R2, 114514, asm.DWord),
			asm.StoreMem(asm.R1, offsetOfMark, asm.R2, asm.Word),
			asm.LoadImm(asm.R0, 1, asm.DWord),
			asm.Return(),
		},
	}

	prog, err := ebpf.NewProgram(progSpec)
	if err != nil {
		abort("Compile ebpf program: %s", err.Error())
	}
	defer prog.Close()

	lnk, err := link.AttachCgroup(link.CgroupOptions{
		Path:    filepath.Join(cgroup.CgroupPath, cgroupSubtree),
		Attach:  ebpf.AttachCGroupInetSockCreate,
		Program: prog,
	})
	if err != nil {
		abort("Link ebpf program: %s", err.Error())
	}
	defer lnk.Close()

	rules := []*iproute2.Rule{
		{
			Priority: &iproute2.Uint32Attr{Value: 8000},
			Mark:     &iproute2.Uint32Attr{Value: withoutClashMark},
			Goto:     &iproute2.Uint32Attr{Value: 10000},
		},
		{
			Priority: &iproute2.Uint32Attr{Value: 10000},
		},
	}

	cleanup := func() {
		for _, rule := range rules {
			err = nil
			for err == nil {
				err = iproute2.RuleDel(rule)
			}
		}
	}

	cleanup()
	defer cleanup()

	for _, rule := range rules {
		err = iproute2.RuleAdd(rule)
		if err != nil {
			abort("Add rule: %s", err.Error())
		}
	}

	listener, err := net.Listen("unix", "@"+daemonArg0)
	if err != nil {
		abort("Listen daemon unix: %s", err.Error())
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}

			go func() {
				defer conn.Close()

				n, err := conn.Read([]byte{0})
				if err != nil || n != 1 {
					return
				}

				uc, ok := conn.(*net.UnixConn)
				if !ok {
					return
				}

				sys, err := uc.SyscallConn()
				if err != nil {
					return
				}

				pid := -1

				err = sys.Control(func(fd uintptr) {
					cred, err := unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
					if err != nil {
						return
					}

					pid = int(cred.Pid)
				})
				if err != nil {
					return
				}

				err = cgroup.AddProcessToSubtree(cgroupSubtree, pid)
				if err != nil {
					return
				}

				_, _ = conn.Write([]byte{0})
			}()
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	<-signals
}
