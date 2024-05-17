# 读取文本文件内容
with open("disk_data.txt", "r") as file:
    data = file.readlines()
# 将文本分割成行
lines = data.strip().split('\n')

# 计算每组的行数
lines_per_group = len(lines) // 24

# 将行分成24组
groups = [lines[i:i + lines_per_group] for i in range(0, len(lines), lines_per_group)]

# 打印每组
for i, group in enumerate(groups, start=1):
    print(f"组 {i}:\n{''.join(group)}\n")
