//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/xproto"
)

// Pure-Go X11 clipboard. Xlibre is protocol-compatible with X.Org, so the
// regular CLIPBOARD selection works there too. No xclip/xsel/wl-copy needed.

const (
	selClipboard  = "CLIPBOARD"
	atomUTF8      = "UTF8_STRING"
	atomString    = "STRING"
	atomText      = "TEXT"
	atomTextPlain = "text/plain"
	atomPlainUTF8 = "text/plain;charset=utf-8"
	atomTargets   = "TARGETS"
	atomMultiple  = "MULTIPLE"
	propName      = "GOPASS_SELECTION"
)

type xAtoms map[string]xproto.Atom

func allAtomNames() []string {
	return []string{selClipboard, atomUTF8, atomString, atomText, atomTextPlain,
		atomPlainUTF8, atomTargets, atomMultiple, propName}
}

func internAtoms(c *xgb.Conn, names ...string) (xAtoms, error) {
	m := make(xAtoms, len(names))
	for _, n := range names {
		rep, err := xproto.InternAtom(c, false, uint16(len(n)), n).Reply()
		if err != nil {
			return nil, err
		}
		m[n] = rep.Atom
	}
	return m, nil
}

// serveClipboard takes ownership of the X11 CLIPBOARD selection and answers
// paste requests until another client takes ownership or timeout elapses
// (timeout <= 0 means "until replaced"). If ready is non-nil it is closed once
// ownership has been confirmed.
func serveClipboard(text string, timeout time.Duration, ready chan<- struct{}) error {
	c, err := xgb.NewConn()
	if err != nil {
		return fmt.Errorf("cannot connect to X server: %v", err)
	}
	defer c.Close()

	atoms, err := internAtoms(c, allAtomNames()...)
	if err != nil {
		return err
	}

	screen := xproto.Setup(c).DefaultScreen(c)
	wid, err := c.NewId()
	if err != nil {
		return err
	}
	err = xproto.CreateWindowChecked(c, 0, xproto.Window(wid), screen.Root,
		-1, -1, 1, 1, 0, 0, 0, 0, nil).Check()
	if err != nil {
		return err
	}

	xproto.SetSelectionOwner(c, xproto.Window(wid), atoms[selClipboard], xproto.TimeCurrentTime)

	rep, err := xproto.GetSelectionOwner(c, atoms[selClipboard]).Reply()
	if err != nil {
		return err
	}
	if rep.Owner != xproto.Window(wid) {
		return errors.New("failed to acquire CLIPBOARD ownership")
	}
	if ready != nil {
		close(ready)
	}

	done := make(chan error, 1)
	go func() { done <- selectionEventLoop(c, text, atoms) }()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		select {
		case err := <-done:
			return err
		case <-timer.C:
			return nil
		case <-sig:
			return nil
		}
	}
	select {
	case err := <-done:
		return err
	case <-sig:
		return nil
	}
}

func selectionEventLoop(c *xgb.Conn, text string, atoms xAtoms) error {
	for {
		ev, err := c.WaitForEvent()
		if err != nil { // connection closed or protocol error
			return err
		}
		switch e := ev.(type) {
		case xproto.SelectionClearEvent:
			return nil // another client owns the clipboard now
		case xproto.DestroyNotifyEvent:
			return nil
		case xproto.SelectionRequestEvent:
			handleSelectionRequest(c, e, text, atoms)
		}
	}
}

func handleSelectionRequest(c *xgb.Conn, e xproto.SelectionRequestEvent, text string, atoms xAtoms) {
	prop := e.Property
	if prop == xproto.AtomNone {
		prop = e.Target
	}

	ok := false
	switch e.Target {
	case atoms[atomTargets]:
		targets := []xproto.Atom{
			atoms[atomTargets], atoms[atomMultiple],
			atoms[atomUTF8], atoms[atomPlainUTF8],
			atoms[atomText], atoms[atomTextPlain], atoms[atomString],
		}
		buf := make([]byte, 4*len(targets))
		for i, t := range targets {
			binary.LittleEndian.PutUint32(buf[i*4:], uint32(t))
		}
		ok = changeProperty(c, e.Requestor, prop, xproto.AtomAtom, 32, buf)
	case atoms[atomUTF8], atoms[atomPlainUTF8], atoms[atomText], atoms[atomTextPlain], atoms[atomString]:
		// Generated passwords are plain ASCII, so serving the same bytes for
		// STRING (Latin-1) and TEXT targets is safe.
		ok = changeProperty(c, e.Requestor, prop, e.Target, 8, []byte(text))
	}
	if !ok {
		prop = xproto.AtomNone // refuse this target
	}

	// Reply with a SelectionNotify event (event code 31).
	b := make([]byte, 32)
	b[0] = 31
	binary.LittleEndian.PutUint32(b[4:], uint32(e.Time))
	binary.LittleEndian.PutUint32(b[8:], uint32(e.Requestor))
	binary.LittleEndian.PutUint32(b[12:], uint32(e.Selection))
	binary.LittleEndian.PutUint32(b[16:], uint32(e.Target))
	binary.LittleEndian.PutUint32(b[20:], uint32(prop))
	xproto.SendEvent(c, false, e.Requestor, 0, string(b))
}

func changeProperty(c *xgb.Conn, win xproto.Window, prop, typ xproto.Atom, format byte, data []byte) bool {
	units := uint32(len(data))
	if format == 32 {
		units /= 4
	}
	return xproto.ChangePropertyChecked(c, xproto.PropModeReplace, win, prop,
		typ, format, units, data).Check() == nil
}

// copyViaX11 copies through the built-in clipboard server. It reports false
// when no X server is reachable, so the caller can fall back to external
// clipboard tools.
func copyViaX11(text string, clearAfter time.Duration) (bool, error) {
	if os.Getenv("DISPLAY") == "" {
		return false, nil
	}
	probe, err := xgb.NewConn()
	if err != nil {
		return false, nil
	}
	probe.Close()
	return true, spawnClipboardDaemon(text, clearAfter)
}

// spawnClipboardDaemon re-executes this binary detached (new session) so the
// clipboard keeps working after the terminal is closed. The password travels
// over a pipe; it is never in argv or the environment.
func spawnClipboardDaemon(text string, timeout time.Duration) error {
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}

	cmd := exec.Command(selfPath(), "--serve-clipboard", "--clear-after="+timeout.String())
	cmd.ExtraFiles = []*os.File{r}
	cmd.Env = os.Environ() // keep DISPLAY for the child
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		r.Close()
		w.Close()
		return err
	}
	if _, err := io.WriteString(w, text); err != nil {
		return err
	}
	w.Close()
	r.Close()
	return nil
}

// selfPath returns the absolute path of this running executable, so the
// detached clipboard daemon can re-exec itself reliably.
func selfPath() string {
	if p, err := os.Executable(); err == nil {
		return p
	}
	return os.Args[0]
}

// runClipboardDaemon is the detached child process: it receives the password
// over the pipe passed as fd 3 and serves the X11 clipboard until it is
// replaced or the timeout elapses.
func runClipboardDaemon(timeout time.Duration) error {
	f := os.NewFile(uintptr(3), "pipe")
	if f == nil {
		return errors.New("no pipe on fd 3")
	}
	data := make([]byte, 0, 64)
	buf := make([]byte, 64)
	for {
		n, err := f.Read(buf)
		data = append(data, buf[:n]...)
		if err != nil {
			break
		}
	}
	f.Close()
	if len(data) == 0 {
		return errors.New("empty clipboard payload")
	}
	return serveClipboard(string(data), timeout, nil)
}

// readClipboardSelection requests the current CLIPBOARD contents as
// UTF8_STRING (equivalent of "xclip -o -selection clipboard").
func readClipboardSelection() (string, error) {
	c, err := xgb.NewConn()
	if err != nil {
		return "", fmt.Errorf("cannot connect to X server: %v", err)
	}
	defer c.Close()

	atoms, err := internAtoms(c, allAtomNames()...)
	if err != nil {
		return "", err
	}

	screen := xproto.Setup(c).DefaultScreen(c)
	wid, err := c.NewId()
	if err != nil {
		return "", err
	}
	if err := xproto.CreateWindowChecked(c, 0, xproto.Window(wid), screen.Root,
		-1, -1, 1, 1, 0, 0, 0, 0, nil).Check(); err != nil {
		return "", err
	}

	xproto.ConvertSelection(c, xproto.Window(wid), atoms[selClipboard],
		atoms[atomUTF8], atoms[propName], xproto.TimeCurrentTime)

	// Wait for the SelectionNotify reply.
	var prop xproto.Atom
	found := false
	deadline := time.Now().Add(5 * time.Second)
	for !found {
		if time.Now().After(deadline) {
			return "", errors.New("timed out waiting for the clipboard owner")
		}
		ev, err := c.WaitForEvent()
		if err != nil {
			return "", err
		}
		if sn, ok := ev.(xproto.SelectionNotifyEvent); ok {
			if sn.Selection != atoms[selClipboard] {
				continue
			}
			if sn.Property == xproto.AtomNone {
				return "", errors.New("clipboard owner refused the request")
			}
			prop = sn.Property
			found = true
		}
	}

	rep, err := xproto.GetProperty(c, false, xproto.Window(wid), prop,
		xproto.GetPropertyTypeAny, 0, 1<<20).Reply()
	if err != nil {
		return "", err
	}
	return string(rep.Value), nil
}
