package main

// #cgo CFLAGS: -Wall -Wextra
// #include <stdint.h>
// typedef struct {
//     int32_t  block_header_height;
//     int32_t  mweb_header_height;
//     int32_t  mweb_utxos_height;
//     uint32_t block_time;
// } StatusResponse;
import "C"

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ltcmweb/mwebd"
	"github.com/ltcmweb/mwebd/proto"
)

// Global registry to keep Go objects alive
var (
	serverRegistry   = make(map[uintptr]*mwebd.Server)
	serverRegistryMu sync.Mutex
	serverCounter    uintptr = 1
)

//export CreateServer
func CreateServer(chain, dataDir, peer, proxy *C.char) C.uintptr_t {
	args := &mwebd.ServerArgs{
		Chain:     C.GoString(chain),
		DataDir:   C.GoString(dataDir),
		PeerAddr:  C.GoString(peer),
		ProxyAddr: C.GoString(proxy),
	}
	server, err := mwebd.NewServer2(args)
	if err != nil {
		fmt.Print("error")
		panic(err)
	}

	serverRegistryMu.Lock()
	id := serverCounter
	serverRegistry[id] = server
	serverCounter++
	serverRegistryMu.Unlock()

	return C.uintptr_t(id)

}

//export StartServer
func StartServer(id C.uintptr_t, port C.int) C.int {
	serverRegistryMu.Lock()
	defer serverRegistryMu.Unlock()
	server := serverRegistry[uintptr(id)]
	// go waitForParent(server)
	selectedPort, err := server.Start(int(port))
	if err != nil {
		panic(err)
	}
	return C.int(selectedPort)
}

//export StopServer
func StopServer(id C.uintptr_t) {
	serverRegistryMu.Lock()
	server, exists := serverRegistry[uintptr(id)]
	if !exists {
		serverRegistryMu.Unlock()
		return // Server was already stopped.
	}
	delete(serverRegistry, uintptr(id))
	serverRegistryMu.Unlock()

	// Implement timeout-based shutdown to prevent hanging.
	done := make(chan struct{})
	go func() {
		defer close(done)
		server.Stop()
	}()

	// Wait for graceful shutdown with timeout.
	select {
	case <-done:
		// Graceful shutdown completed.
		return
	case <-time.After(2 * time.Second):
		// Timeout reached - force return to prevent FFI hanging.
		// The server goroutines will be cleaned up when the process exits.
		fmt.Printf("Warning: Server stop timed out after 2 seconds, continuing with FFI return\n")
		return
	}
}

//export Status
func Status(id C.uintptr_t, out *C.StatusResponse) {
    server := serverRegistry[uintptr(id)]
    response, err := server.Status(context.Background(), &proto.StatusRequest{})
    if err != nil {
        panic(err)
    }

    out.block_header_height = C.int32_t(response.BlockHeaderHeight)
    out.mweb_header_height = C.int32_t(response.MwebHeaderHeight)
    out.mweb_utxos_height = C.int32_t(response.MwebUtxosHeight)
    out.block_time = C.uint32_t(response.BlockTime)
}

func main() {}
