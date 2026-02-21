package launchservices

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// WaitForExit uses kqueue/kevent to block until the given process exits.
// This is the same mechanism /usr/bin/open uses for -W (--wait-apps).
func WaitForExit(pid int) error {
	return WaitForExitMultiple([]int{pid})
}

// WaitForExitMultiple uses kqueue/kevent to block until all given processes exit.
func WaitForExitMultiple(pids []int) error {
	if len(pids) == 0 {
		return nil
	}

	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("kqueue: %w", err)
	}
	defer unix.Close(kq)

	changes := make([]unix.Kevent_t, len(pids))
	for i, pid := range pids {
		changes[i] = unix.Kevent_t{
			Ident:  uint64(pid),
			Filter: unix.EVFILT_PROC,
			Flags:  unix.EV_ADD | unix.EV_ENABLE,
			Fflags: unix.NOTE_EXIT,
		}
	}

	events := make([]unix.Kevent_t, len(pids))
	n, err := unix.Kevent(kq, changes, events, nil)
	if err != nil {
		return fmt.Errorf("kevent register: %w", err)
	}

	remaining := len(pids)
	for i := 0; i < n; i++ {
		if events[i].Fflags&unix.NOTE_EXIT != 0 {
			remaining--
		}
	}

	for remaining > 0 {
		n, err := unix.Kevent(kq, nil, events, nil)
		if err != nil {
			if err == unix.EINTR {
				continue
			}
			return fmt.Errorf("kevent wait: %w", err)
		}
		for i := 0; i < n; i++ {
			if events[i].Fflags&unix.NOTE_EXIT != 0 {
				remaining--
			}
		}
	}

	return nil
}
