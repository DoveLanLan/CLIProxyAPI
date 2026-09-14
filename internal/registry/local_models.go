package registry

import "sync/atomic"

var localModelsOnly atomic.Bool

// SetLocalModelsOnly disables provider-native catalog discovery for this process.
func SetLocalModelsOnly(local bool) { localModelsOnly.Store(local) }

// LocalModelsOnly reports whether native providers must use configured models only.
func LocalModelsOnly() bool { return localModelsOnly.Load() }
