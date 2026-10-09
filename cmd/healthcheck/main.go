// Command healthcheck dials the game server port and exits non-zero when it is not
// accepting connections. It is used by the container HEALTHCHECK so that orchestrators
// (docker compose, Kubernetes) can tell a live server from a wedged one.
package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	addr := os.Getenv("HEALTHCHECK_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8999"
	}

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		fmt.Fprintf(os.Stderr, "healthcheck: %v\n", err)
		os.Exit(1)
	}
	_ = conn.Close()
	fmt.Printf("healthcheck: %s is accepting connections\n", addr)
}
