package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// NodeInfo 结构体用于保存每个节点的信息
type NodeInfo struct {
	NodeName   string    `json:"nodeName"`
	GPUTotal   int64     `json:"gpuTotal"`
	GPUUsed    int64     `json:"gpuUsed"`
	GPUPodInfo []PodInfo `json:"gpuPodInfo"`
	GPUModel   string    `json:"gpuModel"`
	GPUDriver  string    `json:"gpuDriver"`
}

// PodInfo 结构体用于保存每个Pod的信息
type PodInfo struct {
	PodName   string `json:"podName"`
	Namespace string `json:"namespace"`
	GPUUsed   int64  `json:"gpu_used"`
}

// ByGPUUsed 定义了按照 GPU 使用量排序的方式
type ByGPUUsed []NodeInfo

func (a ByGPUUsed) Len() int           { return len(a) }
func (a ByGPUUsed) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByGPUUsed) Less(i, j int) bool { return a[i].GPUUsed < a[j].GPUUsed }

// TODO: 后续要改成在集群内部通过ServiceAccount方式获取
// 获取kubeconfig配置
func getKubernetesClient() (*kubernetes.Clientset, error) {
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("HOME") + "/.kube/config"
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func getNodeGPUUsage(clientset *kubernetes.Clientset) ([]NodeInfo, error) {
	// 返回 GPU 使用量、Pod 信息和可能的错误
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	var nodesInfo []NodeInfo

	// 遍历节点列表
	for _, node := range nodes.Items {
		gpuTotal, ok := node.Status.Allocatable[corev1.ResourceName("nvidia.com/gpu")]
		if !ok {
			continue
		}

		gpuUsed, podInfos, err := getGPUUsageForNode(clientset, node.Name)
		gpuModel, gpuModelOk := node.Labels["gpu/model"]
		gpuDriver, gpuDriverOk := node.Labels["gpu/driver"]
		if !gpuModelOk || !gpuDriverOk {
			fmt.Printf("节点 %s 上没有 GPU 型号或驱动版本信息\n", node.Name)
			continue
		}

		if err != nil {
			fmt.Printf("获取节点 %s 上的 GPU 使用量时出现错误：%v\n", node.Name, err)
			continue
		}

		nodeInfo := NodeInfo{
			NodeName:   node.Name,
			GPUTotal:   gpuTotal.Value(),
			GPUUsed:    gpuUsed,
			GPUModel:   gpuModel,
			GPUDriver:  gpuDriver,
			GPUPodInfo: podInfos,
		}

		nodesInfo = append(nodesInfo, nodeInfo)
	}
	return nodesInfo, err
}

// getGPUUsageForNode 获取指定节点上已使用的 GPU 数量及相关 Pod 信息
func getGPUUsageForNode(clientset *kubernetes.Clientset, nodeName string) (int64, []PodInfo, error) {
	podList, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
	})
	if err != nil {
		return 0, nil, err
	}

	var gpuUsed int64
	podInfos := make([]PodInfo, 0) // 初始化切片

	for _, pod := range podList.Items {
		hasGPURequest := false
		totalGPUUsedPerPod := int64(0)
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if _, found := container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]; found {
					hasGPURequest = true
					gpuRequests, _ := container.Resources.Requests[corev1.ResourceName("nvidia.com/gpu")]
					totalGPUUsedPerPod += gpuRequests.Value()
				}
			}
		}

		// 只有当Pod请求了GPU资源时，才统计其GPU使用量和信息
		if hasGPURequest {
			podInfo := PodInfo{
				PodName:   pod.Name,
				Namespace: pod.Namespace,
				GPUUsed:   totalGPUUsedPerPod,
			}

			podInfos = append(podInfos, podInfo)
			gpuUsed += totalGPUUsedPerPod
		}
	}

	return gpuUsed, podInfos, nil
}

// func GenerateMarkdownTable(jsonData []byte) (string, error) {
func GenerateMarkdownTable(nodesInfo []NodeInfo) (string, error) {

	// 按照GPU使用量对节点进行排序
	sort.Sort(ByGPUUsed(nodesInfo))

	// 使用strings.Builder来构建Markdown字符串
	var markdown strings.Builder
	markdown.WriteString("| 节点名称 | GPU总数 | GPU已使用 | GPU使用详情 | GPU型号 | GPU驱动版本 \n")
	markdown.WriteString("|----------|--------|----------|------------|------------|------------|\n")

	// 遍历每个节点，生成Markdown表格行
	for _, node := range nodesInfo {
		var details []string
		for _, pod := range node.GPUPodInfo {
			details = append(details, fmt.Sprintf("%s (%s) - %d", pod.PodName, pod.Namespace, pod.GPUUsed))
		}
		detailsStr := strings.Join(details, "<br />")
		markdown.WriteString(fmt.Sprintf("| %s | %d | %d | %s | %s | %s |\n", node.NodeName, node.GPUTotal, node.GPUUsed, detailsStr, node.GPUModel, node.GPUDriver))
	}

	return markdown.String(), nil
}
func main() {
	// 获取 Kubernetes 客户端
	clientset, err := getKubernetesClient()
	if err != nil {
		log.Fatalf("无法建立 Kubernetes 客户端连接: %v", err)
	}
	// Node GPU使用详情
	nodesInfo, err := getNodeGPUUsage(clientset)
	if err != nil {
		panic(err.Error())
	}
	// 生成Markdown报告
	markdownReport, err := GenerateMarkdownTable(nodesInfo)
	if err != nil {
		log.Fatalf("生成Markdown报告出错: %v", err)
	}
	// 打印Markdown报告或进行其他操作
	fmt.Println(markdownReport)

	// 钉钉Webhook URL和Markdown内容
	// webhookURL := os.Getenv("webhook")
	// if webhookURL == "" {
	// 	webhookURL = "https://yach-oapi.zhiyinlou.com/robot/send?access_token=cmREMWJjNERSdUdESzRxTlp5ZGVJTE5rS1VwZ3EreUVybDRWVkQ1Z1NRVmpYamJsN0h6K1R3TER0OFBsS1gvcQ"
	// }
	// // 发送消息
	// sendMarkdownToDingTalk(webhookURL, "GPU使用情况", markdownReport)
	// if err != nil {
	// 	log.Fatal(err)
	// }

}
