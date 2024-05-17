package main

import "fmt"

type ServiceMonitor struct {
	Name     string
	EndPoint []string
}

type ServiceMonitorList struct {
	Name  string
	Items []*ServiceMonitor
}

func main() {
	sm := &ServiceMonitor{
		Name: "test",
	}
	smList := &ServiceMonitorList{}
}
