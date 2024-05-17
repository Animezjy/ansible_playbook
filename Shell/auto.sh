#!/bin/bash
# 1. 修改service
# kubernetes集群中glm命名空间下是否有新增的service，
# 一旦检测到有新的service创建出来，那么就给它打上label release: kube-prometheus-stack 并且把port的名字也改成和service同名
# 2. 渲染servicemonitor模板
# 获取到当前的命名空间、service名称、app这个label和对应的值，并用这些内容去渲染提前准备好的servicemonitor模板

# 3. 创建对应的servicemonitor资源，并检查
# 4. 创建对应应用的grafana看板
