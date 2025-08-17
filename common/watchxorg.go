package common

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <xcb/xcb.h>

typedef struct xcb_connection_t xcb_connection_t;
typedef xcb_connection_t* (*xcb_connect_func)(const char*, int*);
typedef int (*xcb_connection_has_error_func)(xcb_connection_t*);
typedef xcb_generic_event_t* (*xcb_wait_for_event_func)(xcb_connection_t*);
typedef void (*xcb_disconnect_func)(xcb_connection_t*);

static void* p_xcb_handle = NULL;
static xcb_connect_func p_xcb_connect = NULL;
static xcb_connection_has_error_func p_xcb_connection_has_error = NULL;
static xcb_wait_for_event_func p_xcb_wait_for_event = NULL;
static xcb_disconnect_func p_xcb_disconnect = NULL;

int loadXorgLibs() {
    p_xcb_handle = dlopen("libxcb.so.1", RTLD_LAZY);
    if (!p_xcb_handle) {
        return -1;
    }
    p_xcb_connect = (xcb_connect_func)dlsym(p_xcb_handle, "xcb_connect");
    p_xcb_connection_has_error = (xcb_connection_has_error_func)dlsym(p_xcb_handle, "xcb_connection_has_error");
	p_xcb_wait_for_event = (xcb_wait_for_event_func)dlsym(p_xcb_handle, "xcb_wait_for_event");
    p_xcb_disconnect = (xcb_disconnect_func)dlsym(p_xcb_handle, "xcb_disconnect");
    if (!p_xcb_connect || !p_xcb_connection_has_error || !p_xcb_wait_for_event || !p_xcb_disconnect) {
        dlclose(p_xcb_handle);
        return -2;
    }
    return 0;
}

int areXorgLibsLoaded() {
	return p_xcb_handle != NULL;
}

void unloadXorgLibs() {
	if (p_xcb_handle) {
	    dlclose(p_xcb_handle);
		p_xcb_handle = NULL;
	}
}

xcb_connection_t* my_xcb_connect(const char* name) {
    if (!p_xcb_connect) return NULL;
    return p_xcb_connect(name, NULL);
}

int my_xcb_connection_has_error(xcb_connection_t* c) {
    if (!p_xcb_connection_has_error) return -1;
    return p_xcb_connection_has_error(c);
}

xcb_generic_event_t* my_xcb_wait_for_event(xcb_connection_t* c) {
    if (!p_xcb_wait_for_event) return NULL;
    return p_xcb_wait_for_event(c);
}

void my_xcb_disconnect(xcb_connection_t* c) {
    if (!p_xcb_disconnect) return;
    p_xcb_disconnect(c);
}

*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

func LoadXorgLibs() error {
	if C.loadXorgLibs() != 0 {
		return fmt.Errorf("xcb not available on this system")
	}
	return nil
}

func UnloadXorgLibs() {
	C.unloadXorgLibs()
}

func TryConnectXorg(displayName string) (*C.xcb_connection_t, error) {
	if C.areXorgLibsLoaded() == 0 {
		return nil, errors.New("xorg libs are not loaded")
	}

	cDisplayName := C.CString(displayName)
	defer C.free(unsafe.Pointer(cDisplayName))

	conn := C.my_xcb_connect(cDisplayName)
	if conn == nil {
		return nil, fmt.Errorf("failed to connect to xorg display %v", displayName)
	}
	if C.my_xcb_connection_has_error(conn) != 0 {
		return nil, fmt.Errorf("connection to xorg display %v has errors", displayName)
	}
	return conn, nil
}

func WatchXorgActive(conn *C.xcb_connection_t) {
	for {
		event := C.my_xcb_wait_for_event(conn)
		if event == nil {
			fmt.Println("Connection closed or error")
			return
		}
		C.free(unsafe.Pointer(event))
	}
}

func DisconnectXorg(conn *C.xcb_connection_t) {
	C.my_xcb_disconnect(conn)
}
