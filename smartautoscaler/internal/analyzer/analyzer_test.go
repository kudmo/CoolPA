package analyzer

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	contextutil "github.com/kudmo/CoolPA/context"
	"github.com/kudmo/CoolPA/internal/analyzer/interfaces"
	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/utils/welchtest"
	"github.com/kudmo/toporank/types"
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

func createTestContext() (context.Context, time.Time) {
	now := time.Now()
	ctx := contextutil.WithAnalysisTime(context.Background(), now)
	return ctx, now
}

func newTestAnalyzer(mockRepo interfaces.MetricsRepository) *Analyzer {
	return &Analyzer{
		metricsProvider: mockRepo,
		config: AnalyzerConfig{
			Window: 1 * time.Hour,
			SLO:    500.0, // milliseconds
			Alpha:  0.1,
		},
	}
}

func TestFindAbnormalCalls_SLOViolation(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := "service-a"
	svcB := "service-b"
	mockRepo.On("ListServices", mock.Anything).Return([]string{svcA}, nil)

	svcAInfo := metrics.ServiceInfo{
		Name:         svcA,
		OuboundCalls: []string{svcB},
	}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcAInfo, nil)

	lat95 := []float64{600.0}
	lat50 := []float64{100.0}
	mockRepo.On("GetGraphLatencyP95Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat95, nil)
	mockRepo.On("GetGraphLatencyP50Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat50, nil)

	a := newTestAnalyzer(mockRepo)
	calls := a.findAbnormalCalls(ctx)

	assert.Len(t, calls, 1)
	assert.Equal(t, svcA, calls[0].from)
	assert.Equal(t, svcB, calls[0].to)
}

func TestFindAbnormalCalls_RatioViolation(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := "svc-a"
	svcB := "svc-b"
	mockRepo.On("ListServices", mock.Anything).Return([]string{svcA}, nil)

	svcAInfo := metrics.ServiceInfo{Name: svcA, OuboundCalls: []string{svcB}}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcAInfo, nil)

	lat95 := []float64{130.0}
	lat50 := []float64{10.0}
	mockRepo.On("GetGraphLatencyP95Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat95, nil)
	mockRepo.On("GetGraphLatencyP50Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat50, nil)

	a := newTestAnalyzer(mockRepo)
	calls := a.findAbnormalCalls(ctx)

	assert.Len(t, calls, 1)
}

func TestComputeAnomalyDegree(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := "svc-a"
	svcB := "svc-b"
	abnormalCalls := []call{{from: svcB, to: svcA}}

	svcAInfo := metrics.ServiceInfo{Name: svcA, InboundCalls: []string{svcB}}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcAInfo, nil)

	svcBInfo := metrics.ServiceInfo{Name: svcB, InboundCalls: []string{}}
	mockRepo.On("GetService", mock.Anything, svcB).Return(svcBInfo, nil)

	latSeries := []float64{400.0, 600.0}
	mockRepo.On("GetGraphLatencyP95Range", ctx, svcB, svcA, 1*time.Hour).
		Return(latSeries, nil)

	a := newTestAnalyzer(mockRepo)
	anomalyMap := a.computeAnomalyDegree(ctx, abnormalCalls)

	assert.Equal(t, float64(1), anomalyMap[svcA])
	assert.Equal(t, float64(0), anomalyMap[svcB])
}

func TestBuildCorrelationGraphFromCalls(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := "svc-a"
	svcB := "svc-b"
	abnormalCalls := []call{{from: svcA, to: svcB}}
	serviceAnomaly := map[string]float64{svcA: 1.0, svcB: 2.0}

	latSeries := []float64{1, 2, 3, 4, 5}
	mockRepo.On("GetGraphLatencyP95Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(latSeries, nil)

	cpuSeries := []float64{2, 4, 6, 8, 10}
	mockRepo.On("GetServiceCpuUsageRange", mock.Anything, svcB, 1*time.Hour).
		Return(cpuSeries, nil)

	mockRepo.On("GetServiceMemoryUsageRange", mock.Anything, svcB, 1*time.Hour).
		Return([]float64{}, nil)
	mockRepo.On("GetServiceFSUsageRange", mock.Anything, svcB, 1*time.Hour).
		Return([]float64{}, nil)
	mockRepo.On("GetServiceFSWriteRange", mock.Anything, svcB, 1*time.Hour).
		Return([]float64{}, nil)
	mockRepo.On("GetServiceFSReadRange", mock.Anything, svcB, 1*time.Hour).
		Return([]float64{}, nil)
	mockRepo.On("GetServiceNetworkReceiveRange", mock.Anything, svcB, 1*time.Hour).
		Return([]float64{}, nil)
	mockRepo.On("GetServiceNetworkTransmitRange", mock.Anything, svcB, 1*time.Hour).
		Return([]float64{}, nil)

	a := newTestAnalyzer(mockRepo)
	graph, err := a.buildCorrelationGraphFromCalls(ctx, abnormalCalls, serviceAnomaly)

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	_, existsA := graph.Nodes[svcA]
	assert.True(t, existsA, "Node %s should exist", svcA)
	_, existsB := graph.Nodes[svcB]
	assert.True(t, existsB, "Node %s should exist", svcB)

	edges := graph.Edges[svcA]
	assert.NotEmpty(t, edges, "Edges from %s should not be empty", svcA)

	var targetEdge *types.Edge
	for _, e := range edges {
		if e.To == svcB {
			targetEdge = e
			break
		}
	}
	assert.NotNil(t, targetEdge, "Edge from %s to %s should exist", svcA, svcB)
	assert.InDelta(t, 1.0, targetEdge.Weight, 0.001)
}

func TestBuildAbnormalCorrelationGraph_NoAbnormal(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	mockRepo.On("ListServices", mock.Anything).Return([]string{"svc-a"}, nil)
	svcInfo := metrics.ServiceInfo{Name: "svc-a", OuboundCalls: []string{}}
	mockRepo.On("GetService", mock.Anything, "svc-a").Return(svcInfo, nil)

	a := newTestAnalyzer(mockRepo)
	graph, err := a.buildAbnormalCorrelationGraph(ctx)

	assert.NoError(t, err)
	assert.Nil(t, graph)
}

func TestAnalyzeRPSlowing(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)
	serviceName := "svc1"

	mockRepo.On("ListServices", mock.Anything).Return([]string{serviceName}, nil)

	requests := make([]float64, 100)
	for i := 0; i < 50; i++ {
		requests[i] = 40.0
	}
	for i := 50; i < 100; i++ {
		requests[i] = 60.0
	}
	mockRepo.On("GetServiceRequestsCountRange", mock.Anything, serviceName, 1*time.Hour).
		Return(requests, nil)

	mockRepo.On("GetServiceCpuQuota", mock.Anything, serviceName).Return(2.0, nil)
	mockRepo.On("GetServiceMemoryQuota", mock.Anything, serviceName).Return(1024.0, nil)
	mockRepo.On("GetServiceReplicasCountValue", mock.Anything, serviceName).Return(3.0, nil)

	cpuUsage := []float64{0.5, 0.5, 0.5}
	mockRepo.On("GetServiceCpuUsageRange", mock.Anything, serviceName, 1*time.Hour).
		Return(cpuUsage, nil)

	memUsage := []float64{100.0, 100.0, 100.0}
	mockRepo.On("GetServiceMemoryUsageRange", mock.Anything, serviceName, 1*time.Hour).
		Return(memUsage, nil)

	mockRepo.On("GetGlobalTotalCpuLimit", mock.Anything).Return(10.0, nil)
	mockRepo.On("GetGlobalTotalMemoryLimit", mock.Anything).Return(4096.0, nil)

	prevStats := &welchtest.Stats{
		N:    100,
		Mean: 100 / BETA,
		M2:   10000 / (BETA * BETA),
	}

	a := &Analyzer{
		metricsProvider: mockRepo,
		config: AnalyzerConfig{
			Window:               time.Hour,
			Confidence:           0.05,
			AnomalyServicesCount: 1,
		},
		previousStatistics: map[string]*welchtest.Stats{serviceName: prevStats},
	}

	results := a.analyzeRPSlowing(ctx)

	assert.Len(t, results, 1)
	assert.Equal(t, serviceName, results[0].Service)

	expectedRate := math.Max((2.0-0.5)*3.0/10.0, (1024.0-100.0)*3.0/4096.0)
	assert.InDelta(t, expectedRate, results[0].Rate, 0.0001)
}

func TestAnalyzeUnderutilization(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)
	serviceName := "svc1"

	mockRepo.On("ListServices", mock.Anything).Return([]string{serviceName}, nil)

	requests := make([]float64, 100)
	for i := 0; i < 50; i++ {
		requests[i] = 40.0
	}
	for i := 50; i < 100; i++ {
		requests[i] = 60.0
	}
	mockRepo.On("GetServiceRequestsCountRange", mock.Anything, serviceName, 1*time.Hour).
		Return(requests, nil)

	mockRepo.On("GetServiceCpuQuota", mock.Anything, serviceName).Return(2.0, nil)
	mockRepo.On("GetServiceMemoryQuota", mock.Anything, serviceName).Return(1024.0, nil)
	mockRepo.On("GetServiceReplicasCountValue", mock.Anything, serviceName).Return(3.0, nil)

	cpuUsage := []float64{0.5, 0.5, 0.5}
	mockRepo.On("GetServiceCpuUsageRange", mock.Anything, serviceName, 1*time.Hour).
		Return(cpuUsage, nil)

	memUsage := []float64{100.0, 100.0, 100.0}
	mockRepo.On("GetServiceMemoryUsageRange", mock.Anything, serviceName, 1*time.Hour).
		Return(memUsage, nil)

	mockRepo.On("GetGlobalTotalCpuLimit", mock.Anything).Return(10.0, nil)
	mockRepo.On("GetGlobalTotalMemoryLimit", mock.Anything).Return(4096.0, nil)

	prevStats := &welchtest.Stats{
		N:    100,
		Mean: 100 / BETA,
		M2:   10000 / (BETA * BETA),
	}

	a := &Analyzer{
		metricsProvider: mockRepo,
		config: AnalyzerConfig{
			Window:               time.Hour,
			Confidence:           0.05,
			AnomalyServicesCount: 1,
		},
		previousStatistics: map[string]*welchtest.Stats{serviceName: prevStats},
	}

	result := a.analyzeUnderutilization(ctx)

	assert.Len(t, result, 1)
	assert.Equal(t, serviceName, result[0])
}
