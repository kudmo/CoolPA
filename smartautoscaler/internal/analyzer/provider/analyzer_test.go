package analyzerprovider

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/kudmo/CoolPA/internal/core"
	"github.com/kudmo/CoolPA/internal/metrics"
	"github.com/kudmo/CoolPA/utils/welchtest"
	"github.com/kudmo/toporank/types"
)

type MockMetricsRepository struct {
	mock.Mock
}

func (m *MockMetricsRepository) ListServices(ctx context.Context, namespace string) ([]string, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockMetricsRepository) GetService(ctx context.Context, serviceName core.ServiceID) (metrics.ServiceInfo, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(metrics.ServiceInfo), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceReplicasCountValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceCpuUsageValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceMemoryUsageValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceCpuQuota(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceMemoryQuota(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSUsageValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSWriteValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSReadValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkReceiveValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkTransmitValue(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServicePodCpuQuota(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServicePodMemoryQuota(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceAverageLatency95Value(ctx context.Context, serviceName core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceName)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceCpuUsageRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceMemoryUsageRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSUsageRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSWriteRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceFSReadRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkReceiveRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceNetworkTransmitRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceRequestsCountRange(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphRequestsCountValue(ctx context.Context, serviceFrom, serviceTo core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP95Value(ctx context.Context, serviceFrom, serviceTo core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP50Value(ctx context.Context, serviceFrom, serviceTo core.ServiceID) (float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphRequestsCountRange(ctx context.Context, serviceFrom, serviceTo core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP95Range(ctx context.Context, serviceFrom, serviceTo core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGraphLatencyP50Range(ctx context.Context, serviceFrom, serviceTo core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceFrom, serviceTo, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetServiceAverageLatency95Range(ctx context.Context, serviceName core.ServiceID, rangeWindow time.Duration) ([]float64, error) {
	args := m.Called(ctx, serviceName, rangeWindow)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalTotalMemoryLimit(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalTotalCpuLimit(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalTotalPodsLimit(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMinCpu(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMaxCpu(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMinMemory(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockMetricsRepository) GetGlobalServiceMaxMemory(ctx context.Context, namespace string) (float64, error) {
	args := m.Called(ctx, namespace)
	return args.Get(0).(float64), args.Error(1)
}

func createTestContext() (context.Context, time.Time) {
	now := time.Now()
	ctx := core.WithAnalysisTime(context.Background(), now)
	return ctx, now
}

func newTestAnalyzer(mockRepo metrics.MetricsRepository) *TopoRankAnalyzer {
	return &TopoRankAnalyzer{
		metricsProvider: mockRepo,
		config: TopoRankAnalyzerConfig{
			Window: 1 * time.Hour,
			SLO:    500.0, // milliseconds
			Alpha:  0.1,
		},
	}
}

func TestFindAbnormalCalls_SLOViolation(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := core.ServiceID{Namespace: "test-namespace-1", Name: "service-a"}
	svcB := core.ServiceID{Namespace: "test-namespace-2", Name: "service-b"}
	mockRepo.On("ListServices", mock.Anything, svcA.Namespace).Return([]string{svcA.Name}, nil)

	svcAInfo := metrics.ServiceInfo{
		ID:            svcA,
		OutboundCalls: []core.ServiceID{svcB},
	}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcAInfo, nil)

	lat95 := []float64{600.0}
	lat50 := []float64{100.0}
	mockRepo.On("GetGraphLatencyP95Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat95, nil)
	mockRepo.On("GetGraphLatencyP50Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat50, nil)

	a := newTestAnalyzer(mockRepo)
	calls := a.findAbnormalCalls(ctx, svcA.Namespace)

	assert.Len(t, calls, 1)
	assert.Equal(t, svcA, calls[0].from)
	assert.Equal(t, svcB, calls[0].to)
}

func TestFindAbnormalCalls_RatioViolation(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := core.ServiceID{Namespace: "test-namespace-1", Name: "service-a"}
	svcB := core.ServiceID{Namespace: "test-namespace-2", Name: "service-b"}
	mockRepo.On("ListServices", mock.Anything, svcA.Namespace).Return([]string{svcA.Name}, nil)

	svcAInfo := metrics.ServiceInfo{
		ID:            svcA,
		OutboundCalls: []core.ServiceID{svcB},
	}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcAInfo, nil)

	lat95 := []float64{130.0}
	lat50 := []float64{10.0}
	mockRepo.On("GetGraphLatencyP95Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat95, nil)
	mockRepo.On("GetGraphLatencyP50Range", mock.Anything, svcA, svcB, 1*time.Hour).
		Return(lat50, nil)

	a := newTestAnalyzer(mockRepo)
	calls := a.findAbnormalCalls(ctx, svcA.Namespace)

	assert.Len(t, calls, 1)
}

func TestComputeAnomalyDegree(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := core.ServiceID{Namespace: "test-namespace-1", Name: "service-a"}
	svcB := core.ServiceID{Namespace: "test-namespace-1", Name: "service-b"}
	abnormalCalls := []call{{from: svcB, to: svcA}}

	svcAInfo := metrics.ServiceInfo{
		ID:           svcA,
		InboundCalls: []core.ServiceID{svcB},
	}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcAInfo, nil)

	svcBInfo := metrics.ServiceInfo{
		ID:            svcB,
		OutboundCalls: []core.ServiceID{svcA},
	}
	mockRepo.On("GetService", mock.Anything, svcB).Return(svcBInfo, nil)

	latSeries := []float64{400.0, 600.0}
	mockRepo.On("GetGraphLatencyP95Range", ctx, svcB, svcA, 1*time.Hour).
		Return(latSeries, nil)

	a := newTestAnalyzer(mockRepo)
	anomalyMap := a.computeAnomalyDegree(ctx, svcA.Namespace, abnormalCalls)

	assert.Equal(t, float64(1), anomalyMap[svcA])
	assert.Equal(t, float64(0), anomalyMap[svcB])
}

func TestBuildCorrelationGraphFromCalls(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)

	svcA := core.ServiceID{Namespace: "test-namespace-1", Name: "service-a"}
	svcB := core.ServiceID{Namespace: "test-namespace-2", Name: "service-b"}
	abnormalCalls := []call{{from: svcA, to: svcB}}
	serviceAnomaly := map[core.ServiceID]float64{svcA: 1.0, svcB: 2.0}

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
	graph, err := a.buildCorrelationGraphFromCalls(ctx, svcA.Namespace, abnormalCalls, serviceAnomaly)

	assert.NoError(t, err)
	assert.NotNil(t, graph)

	_, existsA := graph.Nodes[svcA.ToString()]
	assert.True(t, existsA, "Node %s should exist", svcA)
	_, existsB := graph.Nodes[svcB.ToString()]
	assert.True(t, existsB, "Node %s should exist", svcB)

	edges := graph.Edges[svcA.ToString()]
	assert.NotEmpty(t, edges, "Edges from %s should not be empty", svcA)

	var targetEdge *types.Edge
	for _, e := range edges {
		if e.To == svcB.ToString() {
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

	svcA := core.ServiceID{Namespace: "test-namespace-1", Name: "svc-a"}
	mockRepo.On("ListServices", mock.Anything, svcA.Namespace).Return([]string{svcA.Name}, nil)
	svcInfo := metrics.ServiceInfo{ID: svcA, OutboundCalls: []core.ServiceID{}}
	mockRepo.On("GetService", mock.Anything, svcA).Return(svcInfo, nil)

	a := newTestAnalyzer(mockRepo)
	graph, err := a.buildAbnormalCorrelationGraph(ctx, svcA.Namespace)

	assert.NoError(t, err)
	assert.Nil(t, graph)
}

func TestAnalyzeRPSlowing(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)
	serviceID := core.ServiceID{Namespace: "test-namespace", Name: "svc1"}

	mockRepo.On("ListServices", mock.Anything, serviceID.Namespace).Return([]string{serviceID.Name}, nil)

	requests := make([]float64, 100)
	for i := 0; i < 50; i++ {
		requests[i] = 40.0
	}
	for i := 50; i < 100; i++ {
		requests[i] = 60.0
	}
	mockRepo.On("GetServiceRequestsCountRange", mock.Anything, serviceID, 1*time.Hour).
		Return(requests, nil)

	mockRepo.On("GetServiceCpuQuota", mock.Anything, serviceID).Return(2.0, nil)
	mockRepo.On("GetServiceMemoryQuota", mock.Anything, serviceID).Return(1024.0, nil)
	mockRepo.On("GetServiceReplicasCountValue", mock.Anything, serviceID).Return(3.0, nil)

	cpuUsage := []float64{0.5, 0.5, 0.5}
	mockRepo.On("GetServiceCpuUsageRange", mock.Anything, serviceID, 1*time.Hour).
		Return(cpuUsage, nil)

	memUsage := []float64{100.0, 100.0, 100.0}
	mockRepo.On("GetServiceMemoryUsageRange", mock.Anything, serviceID, 1*time.Hour).
		Return(memUsage, nil)

	mockRepo.On("GetGlobalTotalCpuLimit", mock.Anything, serviceID.Namespace).Return(10.0, nil)
	mockRepo.On("GetGlobalTotalMemoryLimit", mock.Anything, serviceID.Namespace).Return(4096.0, nil)

	BETA := 0.8
	prevStats := &welchtest.Stats{
		N:    100,
		Mean: 100 / BETA,
		M2:   10000 / (BETA * BETA),
	}

	a := &TopoRankAnalyzer{
		metricsProvider: mockRepo,
		config: TopoRankAnalyzerConfig{
			Window:               time.Hour,
			Confidence:           0.05,
			AnomalyServicesCount: 1,
			Beta:                 BETA,
		},
		previousStatistics: map[core.ServiceID]*welchtest.Stats{serviceID: prevStats},
	}

	results := a.analyzeRPSlowing(ctx, serviceID.Namespace)

	assert.Len(t, results, 1)
	assert.Equal(t, serviceID, results[0].Service)

	expectedRate := math.Max((2.0-0.5)*3.0/10.0, (1024.0-100.0)*3.0/4096.0)
	assert.InDelta(t, expectedRate, results[0].Rate, 0.0001)
}

func TestAnalyzeUnderutilization(t *testing.T) {
	ctx, _ := createTestContext()

	mockRepo := new(MockMetricsRepository)
	serviceID := core.ServiceID{Namespace: "test-namespace", Name: "svc1"}

	mockRepo.On("ListServices", mock.Anything, serviceID.Namespace).Return([]string{serviceID.Name}, nil)

	requests := make([]float64, 100)
	for i := 0; i < 50; i++ {
		requests[i] = 40.0
	}
	for i := 50; i < 100; i++ {
		requests[i] = 60.0
	}
	mockRepo.On("GetServiceRequestsCountRange", mock.Anything, serviceID, 1*time.Hour).
		Return(requests, nil)

	mockRepo.On("GetServiceCpuQuota", mock.Anything, serviceID).Return(2.0, nil)
	mockRepo.On("GetServiceMemoryQuota", mock.Anything, serviceID).Return(1024.0, nil)
	mockRepo.On("GetServiceReplicasCountValue", mock.Anything, serviceID).Return(3.0, nil)

	cpuUsage := []float64{0.5, 0.5, 0.5}
	mockRepo.On("GetServiceCpuUsageRange", mock.Anything, serviceID, 1*time.Hour).
		Return(cpuUsage, nil)

	memUsage := []float64{100.0, 100.0, 100.0}
	mockRepo.On("GetServiceMemoryUsageRange", mock.Anything, serviceID, 1*time.Hour).
		Return(memUsage, nil)

	mockRepo.On("GetGlobalTotalCpuLimit", mock.Anything, serviceID.Namespace).Return(10.0, nil)
	mockRepo.On("GetGlobalTotalMemoryLimit", mock.Anything, serviceID.Namespace).Return(4096.0, nil)

	BETA := 0.8
	prevStats := &welchtest.Stats{
		N:    100,
		Mean: 100 / BETA,
		M2:   10000 / (BETA * BETA),
	}

	a := &TopoRankAnalyzer{
		metricsProvider: mockRepo,
		config: TopoRankAnalyzerConfig{
			Window:               time.Hour,
			Confidence:           0.05,
			AnomalyServicesCount: 1,
			Beta:                 BETA,
		},
		previousStatistics: map[core.ServiceID]*welchtest.Stats{serviceID: prevStats},
	}

	result := a.analyzeUnderutilization(ctx, serviceID.Namespace)

	assert.Len(t, result, 1)
	assert.Equal(t, serviceID.Name, result[0])
}
