package redisqueue

import "sync/atomic"

var usageStatisticsEnabled atomic.Bool

func init() {
	usageStatisticsEnabled.Store(true)
}

// SetUsageStatisticsEnabled toggles whether usage records are enqueued.
func SetUsageStatisticsEnabled(enabled bool) { usageStatisticsEnabled.Store(enabled) }

// UsageStatisticsEnabled reports whether the usage queue plugin should publish records.
func UsageStatisticsEnabled() bool { return usageStatisticsEnabled.Load() }
