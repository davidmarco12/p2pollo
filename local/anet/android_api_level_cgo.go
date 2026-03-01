//go:build android && cgo

package anet

// #include <android/api-level.h>
import "C"

import "sync"

var (
	apiLevel int
	once     sync.Once
)

// Returns the API level of the device we're actually running on, or -1 on failure.
func androidDeviceApiLevel() int {
	once.Do(func() {
		apiLevel = int(C.android_get_device_api_level())
	})
	return apiLevel
}
