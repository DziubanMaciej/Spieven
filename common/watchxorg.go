package common

/*
#cgo LDFLAGS: -lxcb -lxcb-util
#include <stdlib.h>
#include <xcb/xcb.h>


*/
import "C"
import (
	"fmt"
	"unsafe"
)

func TryConnectXorg(displayName string) *C.xcb_connection_t {
	// Connect to the X server (NULL = getenv("DISPLAY"))
	cDisplayName := C.CString(displayName)
	defer C.free(unsafe.Pointer(cDisplayName))
	conn := C.xcb_connect(cDisplayName, nil)
	if C.xcb_connection_has_error(conn) != 0 {
		return nil
	}
	return conn
}

func WatchXorgActive(conn *C.xcb_connection_t) {
	for {
		event := C.xcb_wait_for_event(conn)
		if event == nil {
			fmt.Println("Connection closed or error")
			return
		}
		C.free(unsafe.Pointer(event))
	}
}

func DisconnectXorg(conn *C.xcb_connection_t) {
	C.xcb_disconnect(conn)
}
