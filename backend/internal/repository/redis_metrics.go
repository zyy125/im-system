package repository

import "time"

// RedisMetricsRecorder 记录应用内 Redis 业务指标。
// module/op/result 必须保持低基数，禁止放入 key、user_id 等动态值。
type RedisMetricsRecorder interface {
	ObserveOperation(module, op, result string, duration time.Duration)
}

type noopRedisMetricsRecorder struct{}

func (noopRedisMetricsRecorder) ObserveOperation(string, string, string, time.Duration) {}

func withRedisMetrics(recorder RedisMetricsRecorder) RedisMetricsRecorder {
	if recorder == nil {
		return noopRedisMetricsRecorder{}
	}
	return recorder
}
