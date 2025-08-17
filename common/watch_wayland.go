package common

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <wayland-client.h>

typedef struct wl_display wl_display;
typedef struct wl_display *(*wl_display_connect_t)(const char *name);
typedef int (*wl_display_dispatch_t)(struct wl_display *display);
typedef void (*wl_display_disconnect_t)(struct wl_display *display);

static void* p_wayland_handle = NULL;
static wl_display_connect_t p_wl_display_connect = NULL;
static wl_display_dispatch_t p_wl_display_dispatch = NULL;
static wl_display_disconnect_t p_wl_display_disconnect = NULL;

int loadWaylandLibs() {
    p_wayland_handle = dlopen("libwayland-client.so", RTLD_LAZY);
    if (!p_wayland_handle) {
        return -1;
    }
    p_wl_display_connect = (wl_display_connect_t)dlsym(p_wayland_handle, "wl_display_connect");
    p_wl_display_dispatch = (wl_display_dispatch_t)dlsym(p_wayland_handle, "wl_display_dispatch");
	p_wl_display_disconnect = (wl_display_disconnect_t)dlsym(p_wayland_handle, "wl_display_disconnect");
    if (!p_wl_display_connect || !p_wl_display_dispatch || !p_wl_display_disconnect) {
        dlclose(p_wayland_handle);
		p_wayland_handle = NULL;
        return -2;
    }
    return 0;
}

wl_display* my_wl_display_connect(const char *displayName) {
	return p_wl_display_connect(displayName);
}

int my_wl_display_dispatch(wl_display *display) {
	return p_wl_display_dispatch(display);
}

void my_wl_display_disconnect(wl_display *display) {
	p_wl_display_disconnect(display);
}

int areWaylandLibsLoaded() {
	return p_wayland_handle != NULL;
}

void unloadWaylandLibs() {
	if (p_wayland_handle) {
	    dlclose(p_wayland_handle);
		p_wayland_handle = NULL;
	}
}
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

func LoadWaylandLibs() error {
	if C.loadWaylandLibs() != 0 {
		return fmt.Errorf("wayland not available on this system")
	}
	return nil
}

func UnloadWaylandLibs() {
	C.unloadWaylandLibs()
}

func TryConnectWayland(displayName string) (*C.wl_display, error) {
	if C.areWaylandLibsLoaded() == 0 {
		return nil, errors.New("wayland libs are not loaded")
	}

	cDisplayName := C.CString(displayName)
	defer C.free(unsafe.Pointer(cDisplayName))

	display := C.my_wl_display_connect(cDisplayName)
	if display == nil {
		return nil, fmt.Errorf("failed to connect to wayland display %v", displayName)
	}

	return display, nil
}

func WatchWaylandActive(display *C.wl_display) {
	for {
		numEvents := C.my_wl_display_dispatch(display)
		if numEvents == -1 {
			fmt.Println("Connection closed or error")
			return
		}
	}
}

func DisconnecWayland(display *C.wl_display) {
	C.my_wl_display_disconnect(display)
}
