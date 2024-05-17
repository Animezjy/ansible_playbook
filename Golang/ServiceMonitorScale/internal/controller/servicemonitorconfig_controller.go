/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"net/http"
	"reflect"

	hwlv1 "ServiceMonitorScale/api/v1"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ServiceMonitorConfigReconciler reconciles a ServiceMonitorConfig object
type ServiceMonitorConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=hwl.tal.com,resources=servicemonitorconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=hwl.tal.com,resources=servicemonitorconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=hwl.tal.com,resources=servicemonitorconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ServiceMonitorConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.2/pkg/reconcile
func (r *ServiceMonitorConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	// 获取CRD
	config := &hwlv1.ServiceMonitorConfig{}
	configErr := r.Get(ctx, req.NamespacedName, config)

	if configErr != nil {
		return ctrl.Result{}, configErr
	}
	fmt.Println(config.Name)
	fmt.Println(config.Spec.NameSpaceSpec.MatchNames[0])
	fmt.Println("Hello  you are successs")

	service := &corev1.Service{}
	err := r.Get(ctx, req.NamespacedName, service)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			// 删除对应的ServiceMonitor
			return r.deleteServiceMonitor(ctx, req.NamespacedName)
		} else {
			// 处理其他错误
			return ctrl.Result{}, err
		}
	}
	// 检查Service是否具有特定的release标签
	if _, ok := service.Labels["release"]; !ok {
		// 如果没有找到release标签
		service.Labels["release"] = "kube-prometheus-stack"
		err := r.Update(ctx, service)
		if err != nil {
			// if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			//      // 这里可以添加重试逻辑，或者记录错误并返回
			//      return ctrl.Result{}, err
			// }
			// 如果更新失败，记录错误并返回
			log.Log.Error(err, "Failed to update Service with release label")
			return ctrl.Result{}, err
		}
	}
	sm, err := r.createOrUpdateServiceMonitor(ctx, service, config)
	if err != nil {
		return ctrl.Result{}, err
	}
	fmt.Println(sm)

	return ctrl.Result{}, nil
}

// createOrUpdateServiceMonitor 根据Service的状态创建或更新ServiceMonitor
func (r *ServiceMonitorConfigReconciler) createOrUpdateServiceMonitor(ctx context.Context, service *corev1.Service, config *hwlv1.ServiceMonitorConfig) (*monitoringv1.ServiceMonitor, error) {
	// 初始化端口名称为空字符串，表示未命名的端口
	portName := ""

	// 遍历Service的所有端口
	for _, port := range service.Spec.Ports {
		// 如果端口名称与app标签的值相等，使用该端口名称
		if port.Name == service.Labels["app"] {
			portName = port.Name
			break
		}
		// 如果端口没有名称，将其名称设置为app标签的值
		if port.Name == "" {
			portName = service.Labels["app"]
			break
		}
		// 如果端口已有名称，直接使用该名称
		portName = port.Name
	}
	// 检查Service是否提供了健康的/metrics端点
	isHealthy, err := r.checkMetricsEndpoint(service, portName)
	if err != nil {
		return nil, fmt.Errorf("failed to check metrics endpoint health: %v", err)
	}

	if !isHealthy {
		// 如果Service不健康，不创建或更新ServiceMonitor
		log.Log.Info("Service Metrics is unhealthy, will not create ServiceMonitor")
		return nil, nil
	}

	// 创建ServiceMonitor对象
	sm := &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			// Name:      fmt.Sprintf("%s-monitor", portName),
			Name:      portName,
			Namespace: "default", // serviceMonitor和prometheus的命名空间保持一致
			Labels: map[string]string{
				"app":     portName,
				"release": "kube-prometheus-stack",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			// NamespaceSelector: monitoringv1.NamespaceSelector{
			// 	MatchNames: []string{"glm"},
			// },
			NamespaceSelector: config.Spec.NameSpaceSpec,
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"release": "kube-prometheus-stack",
					"app":     portName,
				},
			},
		},
	}
	// 检查ServiceMonitor是否已经存在
	existingSm := &monitoringv1.ServiceMonitor{}
	existingSmNamespacedName := types.NamespacedName{
		Namespace: sm.Namespace,
		Name:      sm.Name,
	}
	if err := r.Get(ctx, existingSmNamespacedName, existingSm); err != nil {
		if apierrors.IsNotFound(err) {
			log.Log.Error(err, "Failed to get ServiceMonitor, will not create a new one")
			return nil, err
		}
		// ServiceMonitor不存在，需要创建
		err := r.Create(ctx, sm)
		if err != nil {
			// TODO: 处理创建失败的逻辑
			return nil, err
		}
	} else {
		// ServiceMonitor已存在，需要检查是否需要更新
		if !reflect.DeepEqual(existingSm.Spec, sm.Spec) || !reflect.DeepEqual(existingSm.Labels, sm.Labels) {
			existingSm.Spec = sm.Spec
			existingSm.Labels = sm.Labels
			err := r.Update(ctx, existingSm)
			if err != nil {
				// TODO: 处理更新失败的情况
				return nil, err
			}
		}
		return existingSm, nil // 返回更新后的ServiceMonitor

	}
	return existingSm, err

}

// checkMetricsEndpoint 检查Service是否提供了健康的/metrics端点
func (r *ServiceMonitorConfigReconciler) checkMetricsEndpoint(service *corev1.Service, portName string) (bool, error) {
	// 查找名为"metrics"的端口
	var metricsPort int
	for _, port := range service.Spec.Ports {
		if port.Name == portName {
			metricsPort = int(port.Port)
			break
		}
	}
	if metricsPort == 0 {
		// 如果没有找到名为"metrics"的端口，返回错误
		return false, fmt.Errorf("metrics port not found in service")
	}

	// 构建用于健康检查的URL
	metricsPath := "/metrics"
	serviceIP := service.Spec.ClusterIP // 假设使用ClusterIP进行访问
	if serviceIP == "" {
		// 如果没有ClusterIP，可能需要使用其他方法来获取Pod的IP地址
		return false, fmt.Errorf("service does not have a ClusterIP")
	}
	serviceURL := fmt.Sprintf("http://%s:%d%s", serviceIP, metricsPort, metricsPath)

	// 发送HTTP GET请求到/metrics端点
	resp, err := http.Get(serviceURL)
	if err != nil {
		return false, fmt.Errorf("failed to reach the metrics endpoint: %v", err)
	}
	defer resp.Body.Close()

	// 如果响应状态码是200，表示健康
	return resp.StatusCode == http.StatusOK, nil
}

// deleteServiceMonitor 删除关联的ServiceMonitor
func (r *ServiceMonitorConfigReconciler) deleteServiceMonitor(ctx context.Context, reqNamespacedName types.NamespacedName) (ctrl.Result, error) {
	// 定义一个空的ServiceMonitor对象，使用相同的命名空间和名称
	sm := &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: reqNamespacedName.Namespace,
			Name:      reqNamespacedName.Name,
		},
	}

	// 删除ServiceMonitor
	err := r.Delete(ctx, sm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// ServiceMonitor已经不存在，无需进一步操作
			return ctrl.Result{}, nil
		}
		// 处理其他错误
		return ctrl.Result{}, err
	}

	// 成功删除ServiceMonitor
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ServiceMonitorConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// 设置监听corev1.Service资源的控制器
	if err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Complete(r); err != nil {
		return err
	}

	// 设置监听hwlv1.ServiceMonitorConfig资源的控制器
	// if err := ctrl.NewControllerManagedBy(mgr).
	// 	For(&hwlv1.ServiceMonitorConfig{}).
	// 	Complete(r); err != nil {
	// 	return err
	// }

	return nil
}
