//go:build linux

package singleinstance

import (
	"context"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

// The first instance owns this name on the session bus for as long as it
// runs; later launches find it taken and ask the owner to come forward.
const (
	busName    = "io.github.netrasad.NetRasad"
	objectPath = dbus.ObjectPath("/io/github/netrasad/NetRasad")
	iface      = "io.github.netrasad.NetRasad"
)

var (
	mu         sync.Mutex
	conn       *dbus.Conn
	onActivate func()
)

type activator struct{}

func (activator) Activate() *dbus.Error {
	mu.Lock()
	fn := onActivate
	mu.Unlock()
	if fn != nil {
		go fn()
	}
	return nil
}

// OnActivate sets what to do when another launch asks this instance to show
// itself.
func OnActivate(fn func()) {
	mu.Lock()
	onActivate = fn
	mu.Unlock()
}

// CheckAndLock reports whether another instance is already running, in which
// case it has been asked to show its window. Without a session bus there is
// nothing to coordinate through, so startup is never blocked.
func CheckAndLock() bool {
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return false
	}
	if err := c.Export(activator{}, objectPath, iface); err != nil {
		c.Close()
		return false
	}
	reply, err := c.RequestName(busName, dbus.NameFlagDoNotQueue)
	if err != nil {
		c.Close()
		return false
	}
	if reply == dbus.RequestNameReplyPrimaryOwner {
		mu.Lock()
		conn = c
		mu.Unlock()
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = c.Object(busName, objectPath).CallWithContext(ctx, iface+".Activate", 0).Err
	c.Close()
	return true
}
