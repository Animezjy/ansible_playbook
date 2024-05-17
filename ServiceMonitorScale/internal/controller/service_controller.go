package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ServiceReconciler reconciles a Service object
type ServiceReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ServiceAccount string
}

func (r *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	// 获取service
	service := &corev1.Service{}

	err := r.Get(ctx, req.NamespacedName, service)
	if err != nil {
		//TODO: 没找到对应的Service，需要删除已监听该Service的Monitor
		log.Log.WithValues("Service", service.Name).Info("Service is deleted.")
		return ctrl.Result{}, nil
	}

	// 给Service添加固定的release label,使prometheus operator可以发现该service
	if service.Labels == nil {
		service.Labels = make(map[string]string)
	}
	if _, ok := service.Labels["release"]; !ok {
		service.Labels["release"] = "kube-prometheus-stack"
		err := r.Update(ctx, service)
		if err != nil {
			log.Log.Error(err, "Failed to update Service with release label")
			return ctrl.Result{}, err
		}
		log.Log.Info("Label [release: kube-prometheus-stack] added successfully")
	}

	// 判断service的port端口名称是否未设置，如果是，那么设置为app标签的值，如果app标签也没有值，那么设置为service的名称
	r.updateServicePortName(ctx, service)

	// 创建或更新ServiceMonitor
	r.createOrUpdateServiceMonitor(ctx, service)

	return ctrl.Result{}, nil
}

func (r *ServiceReconciler) updateServicePortName(ctx context.Context, service *corev1.Service) {
	// 遍历当前service中是否配置了port name
	appName := service.Labels["app"]
	if appName == "" {
		appName = service.Name
	}
	updatedPorts := make([]corev1.ServicePort, len(service.Spec.Ports))
	for i, port := range service.Spec.Ports {
		if port.Name == "" {
			//未配置portName，需要设置为appName
			updatedPorts[i] = corev1.ServicePort{
				Name:       appName,
				Port:       port.Port,
				Protocol:   port.Protocol,
				TargetPort: port.TargetPort,
			}
		} else {
			updatedPorts[i] = port
			appName = port.Name
		}
	}
	// 创建一个新的 Service 对象来应用更改
	updatedService := service.DeepCopy()
	updatedService.Spec.Ports = updatedPorts

	// 应用补丁来更新 Service 的端口名称
	patchBytes, err := json.Marshal(updatedService)
	if err != nil {
		log.Log.Error(err, "failed to marshal updated service")
	}

	patch := client.RawPatch(types.MergePatchType, patchBytes)
	err = r.Patch(ctx, service, patch)
	// 更新失败的处理
	if err != nil {
		log.Log.Error(err, "failed to update service")
	}
	//更新成功
	log.Log.WithValues("PortName", service.Name).Info("Service PortName update success")

}

// createOrUpdateServiceMonitor 根据Service的状态创建或更新ServiceMonitor
func (r *ServiceReconciler) createOrUpdateServiceMonitor(ctx context.Context, service *corev1.Service) {

	// 检查Service是否提供了健康的/metrics端点
	isHealthy, err := r.checkMetricsEndpoint(service)
	if err != nil {
		// 如果连接失败，停止监听并返回错误
		log.Log.Info("Service Metrics is unhealthy, will not create ServiceMonitor")
		return
	}

	if !isHealthy {
		// 如果Service不健康，不创建或更新ServiceMonitor
		log.Log.Info("Service Metrics is unhealthy, will not create ServiceMonitor")
		return
	}

	// 检查当前的service是否已经有了ServiceMonitor
	// 以下情况说明service有对应的ServiceMonitor
	// serviceMonitor的标签选择器匹配了对应的service，比如：当前service的label为  app:test， 正好有一个ServiceMonitor的matchlabels也是 app:test，并且这个ServiceMonitor
	// 监听的Endpoints中对应的portName也刚好是service的PortName，说明完全匹配
	// 针对不完全匹配的情况，要做如下处理
	// 1. ServiceMonitor的标签选择器与service不匹配，那么说明这个ServiceMonitor不负责该service，跳过。如果标签选择器匹配，但是endpoint中的portName匹配不上service的portName
	// 则说明配置错误，这样无法监听到正确的服务，需要修改ServiceMonitor的这个字段。
	// 2. 如果ServiceMonitor的名字与service名字不通，但是标签选择器与endpoint字段的portname全都能匹配到一个service，说明servicdMonitor的名字需要修改（或者不修改，因为service已经有监控，这个monitor有可能是手动配置的）

	// 1. 列出default下的所有ServiceMonitor
	smList := &monitoringv1.ServiceMonitorList{}
	smMap := make(map[string]*monitoringv1.ServiceMonitor)
	if err := r.List(context.TODO(), &monitoringv1.ServiceMonitorList{}, &client.ListOptions{Namespace: "default"}); err != nil {
		log.Log.Error(err, "failed to list ServiceMonitors")
		return
	}
	for _, sm := range smList.Items {
		smMap[sm.Name] = sm.DeepCopy()
	}

	// 检查 Service 是否已经有相应的 ServiceMonitor
	createNew := true
	for _, ep := range service.Spec.Ports {
		servicePortName := ep.Name

		for _, sm := range smMap {
			// 检查 ServiceMonitor 的标签选择器是否与 Service 的标签匹配
			if r.selectorMatchesService(&sm.Spec.Selector, service.Labels) {

				// 检查 ServiceMonitor 的端口是否与 Service 的端口匹配
				for _, smEP := range sm.Spec.Endpoints {
					if smEP.Port == servicePortName {
						log.Log.WithValues("serviceMonitorName", sm.Name).Info("ServiceMonitor already exists for the service")
						createNew = false
						// 找到了，看是否要更新对应的ServiceMonitor资源
						r.updateServiceMonitor(ctx, service, sm)
						break
					}

				}
			}
		}
	}
	if createNew {
		r.createServiceMonitor(ctx, service)

	}

}

func (r *ServiceReconciler) createServiceMonitor(ctx context.Context, service *corev1.Service) {
	// 创建ServiceMonitor
	// 获取app标签的值
	appName := service.Labels["app"]
	// 创建ServiceMonitor对象
	interval := os.Getenv("Interval")
	if interval == "" {
		log.Log.Info("未配置metrics拉取时间，统一设置")
		interval = "15s"
	}
	sm := &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: "default",
			Labels: map[string]string{
				"app":     appName,
				"release": "kube-prometheus-stack",
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{service.Namespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"release": "kube-prometheus-stack",
					"app":     appName,
				},
			},
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:     appName,
					Interval: monitoringv1.Duration(interval),
					Path:     "/metrics",
				},
			},
		},
	}
	existingSm := &monitoringv1.ServiceMonitor{}
	existingSmNamespacedName := types.NamespacedName{
		Namespace: sm.Namespace,
		Name:      sm.Name,
	}

	if err := r.Get(ctx, existingSmNamespacedName, existingSm); err != nil {
		log.Log.WithValues("ServiceMonitor", sm.Name).Info("Failed to get ServiceMonitor")
		err := r.Create(ctx, sm)
		if err != nil {
			log.Log.Error(err, "Create ServiceMonitor error")
		}
		log.Log.WithValues("ServiceMonitor", sm.Name).Info("ServiceMonitor create successfully")
	}

}

func (r *ServiceReconciler) updateServiceMonitor(ctx context.Context, service *corev1.Service, serviceMonitor *monitoringv1.ServiceMonitor) (bool, error) {

	// 检查Labels是否需要更新
	needsUpdate := false

	// 检查Spec.Selector是否需要更新
	if !reflect.DeepEqual(serviceMonitor.Spec.Selector, metav1.LabelSelector{
		MatchLabels: map[string]string{
			"release": "kube-prometheus-stack",
			"app":     service.Labels["app"],
		},
	}) {
		serviceMonitor.Spec.Selector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"release": "kube-prometheus-stack",
				"app":     service.Labels["app"],
			},
		}
		needsUpdate = true
	}

	// 检查Spec.Endpoints是否需要更新
	updatedEndpoint := monitoringv1.Endpoint{
		Port:     service.Labels["app"],
		Interval: monitoringv1.Duration(os.Getenv("Interval")),
		Path:     "/metrics",
	}
	if len(serviceMonitor.Spec.Endpoints) != 1 || !reflect.DeepEqual(serviceMonitor.Spec.Endpoints[0], updatedEndpoint) {
		serviceMonitor.Spec.Endpoints = []monitoringv1.Endpoint{updatedEndpoint}
		needsUpdate = true
	}

	// 如果需要更新，则执行更新操作
	if needsUpdate {
		err := r.Update(ctx, serviceMonitor)
		if err != nil {
			return false, fmt.Errorf("failed to update ServiceMonitor: %v", err)
		}
		log.Log.Info("ServiceMonitor updated successfully")
		return true, nil
	}

	log.Log.Info("ServiceMonitor does not need to be updated")
	return false, nil
}

// selectorMatchesService 方法实现
func (r *ServiceReconciler) selectorMatchesService(selector *metav1.LabelSelector, serviceLabels map[string]string) bool {
	labelSet := labels.Set(serviceLabels)
	selectorSet, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return false
	}
	return selectorSet.Matches(labelSet)
}

// checkMetricsEndpoint 检查Service是否提供了健康的/metrics端点
func (r *ServiceReconciler) checkMetricsEndpoint(service *corev1.Service) (bool, error) {
	// 获取Service关联的所有端口
	for _, port := range service.Spec.Ports {
		// 此时port.Name一定存在
		metricsPort := int(port.Port)
		// 使用Service名称和端口号构建端点地址
		serviceDNSName := fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace)
		metricsEndpoint := fmt.Sprintf("http://%s:%d%s", serviceDNSName, metricsPort, "/metrics")
		log.Log.WithValues("service", service.Name, "metricsEndpoint", metricsEndpoint).Info("Checking metrics endpoint")

		retryCount := 3
		retryDelay := 2 * time.Second
		httpClient := &http.Client{
			Timeout: 10 * time.Second,
		}
		// 发送HTTP GET请求到/metrics端点
		for i := 0; i < retryCount; i++ {
			resp, err := httpClient.Get(metricsEndpoint)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return true, nil
				}
				log.Log.WithValues("service", service.Name, "statusCode", resp.StatusCode).Info("Metrics endpoint returned non-200 status")

				return false, nil // 返回nil错误，表示metrics端点不健康或不可访问，但不中断Reconcile过程
			}

			// If error is due to timeout or connection refused, stop retrying
			if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "connection refused") {
				log.Log.WithValues("service", service.Name).Info("Failed to reach metrics endpoint after retries")
				return false, nil // 返回nil错误，表示metrics端点不健康或不可访问，但不中断Reconcile过程
			}

			time.Sleep(retryDelay)
		}
		return false, nil // 返回nil错误，表示metrics端点不健康或不可访问，但不中断Reconcile过程

	}
	return false, nil
}

// deleteServiceMonitor 删除关联的ServiceMonitor
// func (r *ServiceReconciler) deleteServiceMonitor(ctx context.Context, namespace string) (ctrl.Result, error) {
// 	// 定义一个空的ServiceMonitor对象，使用相同的命名空间和名称
// 	sm := &monitoringv1.ServiceMonitor{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Namespace: "default",
// 			Name:      service.Name,
// 		},
// 	}
//
// 	// 删除ServiceMonitor
// 	err := r.Delete(ctx, sm)
// 	if err != nil {
// 		if apierrors.IsNotFound(err) {
// 			// ServiceMonitor已经不存在，无需进一步操作
// 			return ctrl.Result{}, nil
// 		}
// 		// 处理其他错误
// 		return ctrl.Result{}, err
// 	}
//
// 	// 成功删除ServiceMonitor
// 	return ctrl.Result{}, nil
// }

func serviceNamespacePredicate(namespaces []string) predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// 判断是否是目标命名空间的 Service
			return contains(namespaces, e.Object.GetNamespace())
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// 判断是否是目标命名空间的 Service
			return contains(namespaces, e.ObjectNew.GetNamespace())
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// 判断是否是目标命名空间的 Service
			return contains(namespaces, e.Object.GetNamespace())
		},
	}
}

func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func (r *ServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	targetNamespaces := os.Getenv("ServiceNamespaces")
	if targetNamespaces == "" {
		targetNamespaces = "demo"
	}

	namespaces := strings.Split(targetNamespaces, ",")

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}, builder.WithPredicates(serviceNamespacePredicate(namespaces))).
		//Owns(&monitoringv1.ServiceMonitor{}).
		Complete(r)
}
