package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/metrics/providers/cache"
)

type MockMetricsRepository struct {
	listServicesCalls          int
	getServiceCalls            int
	getServiceCpuUsageCalls    int
	getServiceMemoryUsageCalls int
	getGlobalTotalMemoryCalls  int
	getServiceCpuRangeCalls    int
	getGraphRequestsCountCalls int

	listServicesResult          []string
	listServicesError           error
	getServiceResult            metrics.ServiceInfo
	getServiceError             error
	getServiceCpuUsageResult    float64
	getServiceCpuUsageError     error
	getServiceMemoryUsageResult float64
	getServiceMemoryUsageError  error
	getGlobalTotalMemoryResult  float64
	getGlobalTotalMemoryError   error
	getServiceCpuRangeResult    []float64
	getServiceCpuRangeError     error
	getGraphRequestsCountResult float64
	getGraphRequestsCountError  error
}

func (m *MockMetricsRepository) ListServices(ctx context.Context) ([]string, error) {
	m.listServicesCalls++
	return m.listServicesResult, m.listServicesError
}

func (m *MockMetricsRepository) GetService(ctx context.Context, serviceName string) (metrics.ServiceInfo, error) {
	m.getServiceCalls++
	return m.getServiceResult, m.getServiceError
}

func (m *MockMetricsRepository) GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceCpuUsageValue(ctx context.Context, serviceName string) (float64, error) {
	m.getServiceCpuUsageCalls++
	return m.getServiceCpuUsageResult, m.getServiceCpuUsageError
}

func (m *MockMetricsRepository) GetServiceMemoryUsageValue(ctx context.Context, serviceName string) (float64, error) {
	m.getServiceMemoryUsageCalls++
	return m.getServiceMemoryUsageResult, m.getServiceMemoryUsageError
}

func (m *MockMetricsRepository) GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceFSUsageValue(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceFSWriteValue(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceFSReadValue(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceNetworkReceiveValue(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceNetworkTransmitValue(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServicePodCpuQuota(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServicePodMemoryQuota(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetServiceCpuUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	m.getServiceCpuRangeCalls++
	return m.getServiceCpuRangeResult, m.getServiceCpuRangeError
}

func (m *MockMetricsRepository) GetServiceMemoryUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceFSUsageRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceFSWriteRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceFSReadRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceRequestsCountRange(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	m.getGraphRequestsCountCalls++
	return m.getGraphRequestsCountResult, m.getGraphRequestsCountError
}

func (m *MockMetricsRepository) GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetServiceAverageLatency95Range(ctx context.Context, serviceName string, from, to time.Time) ([]float64, error) {
	return nil, nil
}

func (m *MockMetricsRepository) GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error) {
	m.getGlobalTotalMemoryCalls++
	return m.getGlobalTotalMemoryResult, m.getGlobalTotalMemoryError
}

func (m *MockMetricsRepository) GetGlobalTotalCpuLimit(ctx context.Context) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGlobalTotalPodsLimit(ctx context.Context) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGlobalServiceMinCpu(ctx context.Context) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGlobalServiceMaxCpu(ctx context.Context) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGlobalServiceMinMemory(ctx context.Context) (float64, error) {
	return 0, nil
}

func (m *MockMetricsRepository) GetGlobalServiceMaxMemory(ctx context.Context) (float64, error) {
	return 0, nil
}

func TestCachedMetricsRepository_BasicCaching(t *testing.T) {
	mock := &MockMetricsRepository{
		listServicesResult: []string{"service1", "service2"},
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	services1, err := cachedRepo.ListServices(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	services2, err := cachedRepo.ListServices(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mock.listServicesCalls != 1 {
		t.Errorf("Expected 1 call to ListServices, got %d", mock.listServicesCalls)
	}

	if len(services1) != len(services2) || services1[0] != services2[0] {
		t.Error("Cached result should match original result")
	}
}

func TestCachedMetricsRepository_TTLExpiration(t *testing.T) {
	mock := &MockMetricsRepository{
		getServiceCpuUsageResult: 42.5,
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 10 * time.Millisecond})
	ctx := context.Background()

	value1, err := cachedRepo.GetServiceCpuUsageValue(ctx, "test-service")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	value2, err := cachedRepo.GetServiceCpuUsageValue(ctx, "test-service")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mock.getServiceCpuUsageCalls != 2 {
		t.Errorf("Expected 2 calls after TTL expiration, got %d", mock.getServiceCpuUsageCalls)
	}

	if value1 != value2 {
		t.Errorf("Values should match: %v vs %v", value1, value2)
	}
}

func TestCachedMetricsRepository_DifferentKeys(t *testing.T) {
	mock := &MockMetricsRepository{
		getServiceCpuUsageResult: 10.0,
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	cachedRepo.GetServiceCpuUsageValue(ctx, "service1")
	cachedRepo.GetServiceCpuUsageValue(ctx, "service2")

	if mock.getServiceCpuUsageCalls != 2 {
		t.Errorf("Expected 2 calls for different services, got %d", mock.getServiceCpuUsageCalls)
	}
}

func TestCachedMetricsRepository_ErrorHandling(t *testing.T) {
	mock := &MockMetricsRepository{
		getServiceMemoryUsageError: errors.New("test error"),
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	_, err1 := cachedRepo.GetServiceMemoryUsageValue(ctx, "test-service")
	if err1 == nil {
		t.Fatal("Expected error, got nil")
	}

	_, err2 := cachedRepo.GetServiceMemoryUsageValue(ctx, "test-service")
	if err2 == nil {
		t.Fatal("Expected error, got nil")
	}

	if mock.getServiceMemoryUsageCalls != 2 {
		t.Errorf("Expected 2 calls (errors should not be cached), got %d", mock.getServiceMemoryUsageCalls)
	}
}

func TestCachedMetricsRepository_RangeValues(t *testing.T) {
	mock := &MockMetricsRepository{
		getServiceCpuRangeResult: []float64{1.0, 2.0, 3.0},
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	from := time.Now().Add(-1 * time.Hour)
	to := time.Now()

	result1, err := cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", from, to)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result2, err := cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", from, to)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if mock.getServiceCpuRangeCalls != 1 {
		t.Errorf("Expected 1 call for same range, got %d", mock.getServiceCpuRangeCalls)
	}

	if len(result1) != len(result2) {
		t.Error("Cached range result should match original")
	}
}

func TestCachedMetricsRepository_ClearCache(t *testing.T) {
	mock := &MockMetricsRepository{
		getGraphRequestsCountResult: 100.0,
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	cachedRepo.GetGraphRequestsCountValue(ctx, "service1", "service2")

	cachedRepo.ClearCache()

	cachedRepo.GetGraphRequestsCountValue(ctx, "service1", "service2")

	if mock.getGraphRequestsCountCalls != 2 {
		t.Errorf("Expected 2 calls after cache clear, got %d", mock.getGraphRequestsCountCalls)
	}
}

func TestCachedMetricsRepository_ConcurrentAccess(t *testing.T) {
	mock := &MockMetricsRepository{
		getServiceCpuUsageResult: 30.0,
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			value, err := cachedRepo.GetServiceCpuUsageValue(ctx, "test-service")
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if value != 30.0 {
				t.Errorf("Expected 30.0, got %v", value)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if mock.getServiceCpuUsageCalls > 10 || mock.getServiceCpuUsageCalls < 1 {
		t.Errorf("Expected between 1 and 10 calls, got %d", mock.getServiceCpuUsageCalls)
	}
}

func TestCachedMetricsRepository_GetCacheSize(t *testing.T) {
	mock := &MockMetricsRepository{
		listServicesResult:       []string{"service1"},
		getServiceCpuUsageResult: 15.0,
	}

	cachedRepo := cache.NewCachedMetricsRepository(mock, cache.CachedMetricsProviderConfig{TTL: 5 * time.Minute})
	ctx := context.Background()

	if size := cachedRepo.GetCacheSize(); size != 0 {
		t.Errorf("Expected empty cache, got size %d", size)
	}

	cachedRepo.ListServices(ctx)
	cachedRepo.GetServiceCpuUsageValue(ctx, "service1")

	if size := cachedRepo.GetCacheSize(); size != 2 {
		t.Errorf("Expected cache size 2, got %d", size)
	}
}
