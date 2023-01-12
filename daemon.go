package main

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
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
	listenPath       = "@without-clash"
	cgroupSubtree    = "without-clash"
	offsetOfMark     = 16
	withoutClashMark = 'w'<<24 | 'o'<<16 | 'c'<<8 | 'h'<<0
)

func runDaemon() error {
	if unix.Getuid() != 0 || unix.Geteuid() != 0 {
		return fmt.Errorf("must run as root, current is %d", os.Getuid())
	}

	if !cgroup.IsVersion2() {
		return errors.New("unsupported cgroup version, requires version 2")
	}

	existed, err := cgroup.HasSubtree(cgroupSubtree)
	if err != nil {
		return fmt.Errorf("detect cgroup subtree: %w", err)
	}

	if !existed {
		err = cgroup.CreateSubtree(cgroupSubtree)
		if err != nil {
			return fmt.Errorf("create cgroup: %w", err)
		}
	}
	defer func() {
		_ = cgroup.DeleteSubtree(cgroupSubtree)
	}()

	programSpec := &ebpf.ProgramSpec{
		Name:       "without_clash",
		Type:       ebpf.CGroupSock,
		AttachType: ebpf.AttachCGroupInetSockCreate,
		Instructions: asm.Instructions{
			asm.LoadImm(asm.R2, withoutClashMark, asm.DWord),
			asm.StoreMem(asm.R1, offsetOfMark, asm.R2, asm.Word),
			asm.LoadImm(asm.R0, 1, asm.DWord),
			asm.Return(),
		},
	}

	program, err := ebpf.NewProgram(programSpec)
	if err != nil {
		return fmt.Errorf("compile ebpf program: %w", err)
	}
	defer closeSilent(program)

	attachment, err := link.AttachCgroup(link.CgroupOptions{
		Path:    filepath.Join(cgroup.CgroupPath, cgroupSubtree),
		Attach:  ebpf.AttachCGroupInetSockCreate,
		Program: program,
	})
	if err != nil {
		return fmt.Errorf("attach program to cgroup: %w", err)
	}
	defer closeSilent(attachment)

	rules := []*iproute2.Rule{
		// IPv4
		{
			Priority: &iproute2.Uint32Attr{Value: 8000},
			Mark:     &iproute2.Uint32Attr{Value: withoutClashMark},
			Goto:     &iproute2.Uint32Attr{Value: 10000},
		},
		{
			Priority: &iproute2.Uint32Attr{Value: 10000},
		},
		// IPv6
		{
			Priority: &iproute2.Uint32Attr{Value: 8000},
			Mark:     &iproute2.Uint32Attr{Value: withoutClashMark},
			Goto:     &iproute2.Uint32Attr{Value: 10000},
			Src:      &iproute2.IPCIDRAttr{Value: netip.MustParsePrefix("::/0")},
		},
		{
			Priority: &iproute2.Uint32Attr{Value: 10000},
			Src:      &iproute2.IPCIDRAttr{Value: netip.MustParsePrefix("::/0")},
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
			return fmt.Errorf("add iproute rule: %w", err)
		}
	}

	listener, err := net.ListenUnix("unix", &net.UnixAddr{Name: listenPath, Net: "unix"})
	if err != nil {
		return fmt.Errorf("listen unix: %w", err)
	}
	defer closeSilent(listener)

	go func() {
		for {
			conn, err := listener.AcceptUnix()
			if err != nil {
				continue
			}

			go func() {
				defer closeSilent(conn)

				n, err := conn.Read([]byte{0})
				if err != nil || n != 1 {
					return
				}

				sys, err := conn.SyscallConn()
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

				fmt.Printf("Add process %d to bypass list.\n", pid)
			}()
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	<-signals

	return nil
}
