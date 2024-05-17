#!/bin/bash

# 获取H800机器，并配置ansible主机清单，为后续执行批量命令做准备

# 指定主机清单文件名
inventory_file="h800_hosts"
kubectl --kubeconfig=/home/test/k8s/yq-at-online.conf get node --show-labels | grep GPU | awk '{print $1}' >ip_addresses.txt

private_key="/home/test/sshkey/bd.key"

# 创建主机清单文件
echo "[web_servers]" >$inventory_file

# 并发处理 IP 地址
cat ip_addresses.txt | xargs -I {} -P 10 bash -c "echo '{} ansible_ssh_private_key_file=$private_key' >> $inventory_file"

echo "主机清单已生成：$inventory_file"
