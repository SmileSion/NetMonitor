# 网络连接监控器 (NetMonitor)

一个跨平台的网络连接监控工具,支持Windows和Linux,提供命令行和Web界面两种使用方式。

## 功能特性

- ✅ 实时监控网络连接建立和断开
- ✅ 支持TCP/UDP协议监控
- ✅ 按进程名称、PID、协议类型、远程IP筛选
- ✅ 彩色终端输出
- ✅ Web图形界面 (实时数据展示)
- ✅ WebSocket实时推送
- ✅ 连接统计和活跃进程排行

## 系统要求

- Windows 10/11 或 Linux (内核 4.4+)
- Go 1.24.1 或更高版本

## 安装使用

### Windows

```bash
# 编译
go build -o netmonitor.exe cmd/main.go

# 运行
netmonitor.exe
```

### Linux

```bash
# 编译
go build -o netmonitor cmd/main.go

# 运行
./netmonitor
```

## 配置说明

配置文件位于 `config/config.toml`:

```toml
[log]
listener_dir = "logs/listener_logs"  # 监听端口日志目录
established_dir = "logs/established_logs"  # 已建立连接日志目录
color_enabled = true  # 是否启用彩色输出

[monitor]
interval = 1  # 检测间隔(秒)
show_stats = true  # 是否显示统计信息
log_to_console = true  # 是否输出到控制台

[filter]
# 进程筛选(留空显示全部)
process_name = ""  # 例如: "chrome.exe"
pids = []          # 例如: [1234, 5678]
protocols = ["tcp", "udp"]  # 协议类型
remote_ip = ""      # 远程IP过滤

[web]
enabled = false  # 是否启用Web界面
port = 8080      # Web服务端口
```

## 使用示例

### 命令行模式

默认监控所有连接:

```bash
netmonitor.exe
```

### Web界面模式

1. 修改配置文件启用Web界面:

```toml
[web]
enabled = true
port = 8080
```

2. 启动程序:

```bash
netmonitor.exe
```

3. 在浏览器中访问:

```
http://localhost:8080
```

### 监控特定进程

修改配置文件:

```toml
[filter]
process_name = "chrome.exe"
```

### 监控特定协议

```toml
[filter]
protocols = ["tcp"]
```

## Web界面功能

- 📊 实时统计面板 (活跃连接、监听端口、新建/关闭数)
- 📡 实时事件流 (连接建立/断开事件)
- 🔗 活跃连接列表 (完整连接信息)
- 🔍 筛选功能 (进程、协议、IP)
- 🎨 美观的渐变界面设计

## 跨平台兼容性

本项目使用以下技术确保跨平台兼容:

- `gopsutil` - 跨平台系统信息获取
- 标准库 `syscall` - 系统调用封装
- ANSI颜色代码自动检测

## 注意事项

1. 在Linux上运行可能需要root权限或sudo来获取所有进程的网络信息
2. 日志文件默认保存在 `logs/` 目录
3. Web界面端口8080可能被占用,可修改配置文件中的端口号

## 故障排除

### 权限问题 (Linux)

```bash
sudo ./netmonitor
```

### 端口被占用

修改 `config/config.toml` 中的端口号

### 活跃连接显示"加载中"

确保程序有权限访问网络连接信息

## 许可证

MIT License
