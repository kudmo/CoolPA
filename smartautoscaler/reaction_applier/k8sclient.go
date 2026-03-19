package reactionapplier

import (
	"context"
	"encoding/json"
	"fmt"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8sApplier struct {
	client kubernetes.Interface
}

func NewK8sApplier(client kubernetes.Interface) *K8sApplier {
	return &K8sApplier{client: client}
}

func (k *K8sApplier) ApplyHPS(
	ctx context.Context,
	namespace string,
	workload string,
	replicas int32,
) error {

	scale, err := k.client.AppsV1().
		Deployments(namespace).
		GetScale(ctx, workload, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get scale failed: %w", err)
	}

	scale.Spec.Replicas = replicas

	_, err = k.client.AppsV1().
		Deployments(namespace).
		UpdateScale(ctx, workload, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("update scale failed: %w", err)
	}

	return nil
}

func (k *K8sApplier) ApplyVPS(
	ctx context.Context,
	namespace string,
	workload string,
	cpu string,
	memory string,
) error {

	// Получаем Deployment
	deploy, err := k.client.AppsV1().
		Deployments(namespace).
		Get(ctx, workload, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment failed: %w", err)
	}

	if len(deploy.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("no containers found in deployment")
	}

	// Предполагаем 1 контейнер (можно расширить)
	container := &deploy.Spec.Template.Spec.Containers[0]

	// Парсим ресурсы
	cpuQty, err := resource.ParseQuantity(cpu)
	if err != nil {
		return fmt.Errorf("invalid cpu: %w", err)
	}

	memQty, err := resource.ParseQuantity(memory)
	if err != nil {
		return fmt.Errorf("invalid memory: %w", err)
	}

	// Обновляем requests/limits
	if container.Resources.Requests == nil {
		container.Resources.Requests = v1.ResourceList{}
	}
	if container.Resources.Limits == nil {
		container.Resources.Limits = v1.ResourceList{}
	}

	container.Resources.Requests[v1.ResourceCPU] = cpuQty
	container.Resources.Requests[v1.ResourceMemory] = memQty

	container.Resources.Limits[v1.ResourceCPU] = cpuQty
	container.Resources.Limits[v1.ResourceMemory] = memQty

	// Формируем patch (чтобы не перетирать всё)
	type patchContainer struct {
		Name      string                  `json:"name"`
		Resources v1.ResourceRequirements `json:"resources"`
	}

	patch := map[string]interface{}{
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []patchContainer{
						{
							Name:      container.Name,
							Resources: container.Resources,
						},
					},
				},
			},
		},
	}

	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return fmt.Errorf("marshal patch failed: %w", err)
	}

	_, err = k.client.AppsV1().
		Deployments(namespace).
		Patch(ctx,
			workload,
			types.StrategicMergePatchType,
			patchBytes,
			metav1.PatchOptions{},
		)

	if err != nil {
		return fmt.Errorf("patch deployment failed: %w", err)
	}

	return nil
}

func BuildApplier() (Applier, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return NewK8sApplier(clientset), nil
}
