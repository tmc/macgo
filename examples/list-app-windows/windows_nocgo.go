//go:build !cgo

package main

import "fmt"

func getWindowList(includeOffscreen bool) ([]WindowInfo, error) {
	return nil, fmt.Errorf("list-app-windows requires CGo to access Core Graphics APIs\nPlease rebuild with CGo enabled: CGO_ENABLED=1 go build")
}
