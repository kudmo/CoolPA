package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	contextutil "github.com/kudmo/CoolPA/context"
	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/internal/metrics/providers/cache"
)

type MockMetricsRepository struct {
	mock.Mock
}

func (m *MockMetricsRepository) ListServices(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockMetricsRepository) GetService(ctx context.Context, serviceName string) (metrics.ServiceInfo, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(metrics.ServiceInfo), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceReplicasCountValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceCpuUsageValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceMemoryUsageValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceCpuQuota(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceMemoryQuota(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSUsageValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSWriteValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSReadValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkReceiveValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkTransmitValue(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServicePodCpuQuota(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServicePodMemoryQuota(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceAverageLatency95Value(ctx context.Context, serviceName string) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceCpuUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceMemoryUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSUsageRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSWriteRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSReadRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkReceiveRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkTransmitRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceRequestsCountRange(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo string) (float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceAverageLatency95Range(ctx context.Context, serviceName string, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalTotalMemoryLimit(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalTotalCpuLimit(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalTotalPodsLimit(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMinCpu(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMaxCpu(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMinMemory(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMaxMemory(ctx context.Context) (float64, error) {
	args := m.Called(ctx)
	return args.Get(0).(float64), args.Error(1)
}

// Вспомогательная функция для создания контекста с временем анализа
func createAnalysisContext() context.Context {
	return contextutil.WithAnalysisTime(context.Background(), time.Now())
}

func TestCachedMetricsRepository_BasicCaching(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("ListServices", mock.Anything).Return([]string{"service1", "service2"}, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := context.Background()

	services1, err := cachedRepo.ListServices(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	services2, err := cachedRepo.ListServices(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	mockRepo.AssertNumberOfCalls(t, "ListServices", 1)

	if len(services1) != len(services2) || services1[0] != services2[0] {
		t.Error("Cached result should match original result")
	}
}

func TestCachedMetricsRepository_TTLExpiration(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceCpuUsageValue", mock.Anything, "test-service").Return(42.5, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 10 * time.Millisecond})
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

	mockRepo.AssertNumberOfCalls(t, "GetServiceCpuUsageValue", 2)

	if value1 != value2 {
		t.Errorf("Values should match: %v vs %v", value1, value2)
	}
}

func TestCachedMetricsRepository_DifferentKeys(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceCpuUsageValue", mock.Anything, "service1").Return(10.0, nil)
	mockRepo.On("GetServiceCpuUsageValue", mock.Anything, "service2").Return(20.0, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := context.Background()

	cachedRepo.GetServiceCpuUsageValue(ctx, "service1")
	cachedRepo.GetServiceCpuUsageValue(ctx, "service2")

	mockRepo.AssertNumberOfCalls(t, "GetServiceCpuUsageValue", 2)
}

func TestCachedMetricsRepository_ErrorHandling(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceMemoryUsageValue", mock.Anything, "test-service").
		Return(0.0, errors.New("test error"))

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := context.Background()

	_, err1 := cachedRepo.GetServiceMemoryUsageValue(ctx, "test-service")
	if err1 == nil {
		t.Fatal("Expected error, got nil")
	}

	_, err2 := cachedRepo.GetServiceMemoryUsageValue(ctx, "test-service")
	if err2 == nil {
		t.Fatal("Expected error, got nil")
	}

	mockRepo.AssertNumberOfCalls(t, "GetServiceMemoryUsageValue", 2)
}

func TestCachedMetricsRepository_RangeValues(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceCpuUsageRange",
		mock.Anything, "test-service", 5*time.Minute).
		Return([]float64{1.0, 2.0, 3.0}, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := createAnalysisContext()

	rangeWindow := 5 * time.Minute

	result1, err := cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", rangeWindow)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	result2, err := cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", rangeWindow)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	mockRepo.AssertNumberOfCalls(t, "GetServiceCpuUsageRange", 1)

	if len(result1) != len(result2) {
		t.Error("Cached range result should match original")
	}
}

func TestCachedMetricsRepository_RangeValuesDifferentWindows(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceCpuUsageRange",
		mock.Anything, "test-service", 5*time.Minute).
		Return([]float64{1.0, 2.0, 3.0}, nil)
	mockRepo.On("GetServiceCpuUsageRange",
		mock.Anything, "test-service", 10*time.Minute).
		Return([]float64{1.0, 2.0, 3.0, 4.0, 5.0}, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := createAnalysisContext()

	// Разные окна должны создавать разные ключи
	cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", 5*time.Minute)
	cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", 10*time.Minute)

	mockRepo.AssertNumberOfCalls(t, "GetServiceCpuUsageRange", 2)
}

func TestCachedMetricsRepository_ClearCache(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetGraphRequestsCountValue", mock.Anything, "service1", "service2").
		Return(100.0, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := context.Background()

	cachedRepo.GetGraphRequestsCountValue(ctx, "service1", "service2")

	cachedRepo.ClearCache()

	cachedRepo.GetGraphRequestsCountValue(ctx, "service1", "service2")

	mockRepo.AssertNumberOfCalls(t, "GetGraphRequestsCountValue", 2)
}

func TestCachedMetricsRepository_ConcurrentAccess(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceCpuUsageValue", mock.Anything, "test-service").
		Return(30.0, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
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

	mockRepo.AssertNumberOfCalls(t, "GetServiceCpuUsageValue", 1)
}

func TestCachedMetricsRepository_ConcurrentRangeAccess(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("GetServiceCpuUsageRange",
		mock.Anything, "test-service", 5*time.Minute).
		Return([]float64{1.0, 2.0, 3.0}, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
	ctx := createAnalysisContext()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			value, err := cachedRepo.GetServiceCpuUsageRange(ctx, "test-service", 5*time.Minute)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(value) != 3 {
				t.Errorf("Expected 3 values, got %d", len(value))
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	mockRepo.AssertNumberOfCalls(t, "GetServiceCpuUsageRange", 1)
}

func TestCachedMetricsRepository_GetCacheSize(t *testing.T) {
	mockRepo := new(MockMetricsRepository)
	mockRepo.On("ListServices", mock.Anything).Return([]string{"service1"}, nil)
	mockRepo.On("GetServiceCpuUsageValue", mock.Anything, "service1").Return(15.0, nil)

	cachedRepo := cache.NewCachedMetricsRepository(mockRepo, cache.CachedMetricsProviderConfig{MaxCacheSize: 100, TTL: 5 * time.Minute})
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
