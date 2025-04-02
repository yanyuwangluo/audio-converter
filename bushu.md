# 音频转换系统部署指南

## 1. 系统要求

### 1.1 硬件要求
- CPU: 1核以上
- 内存: 1GB以上
- 硬盘: 10GB以上可用空间

### 1.2 软件要求
- 操作系统: CentOS 7/8 或 Ubuntu 18.04/20.04
- Go 1.16或更高版本
- FFmpeg 4.0或更高版本
- Git

## 2. 安装步骤

### 2.1 安装基础软件

#### CentOS系统：
```bash
# 安装EPEL源
sudo yum install -y epel-release

# 安装基础工具
sudo yum install -y git gcc make

# 安装FFmpeg
sudo yum install -y ffmpeg ffmpeg-devel

# 安装libopus（用于音频编码）
sudo yum install -y opus opus-devel
```

#### Ubuntu系统：
```bash
# 更新软件源
sudo apt update

# 安装基础工具
sudo apt install -y git gcc make

# 安装FFmpeg
sudo apt install -y ffmpeg

# 安装libopus
sudo apt install -y libopus-dev
```

### 2.2 安装Go环境

```bash
# 下载Go安装包（以1.16.14为例）
wget https://golang.org/dl/go1.16.14.linux-amd64.tar.gz

# 解压到/usr/local
sudo tar -C /usr/local -xzf go1.16.14.linux-amd64.tar.gz

# 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
source ~/.bashrc

# 验证安装
go version
```

### 2.3 编译SILK编码器

```bash
# 克隆SILK编码器代码
git clone https://github.com/kn007/silk-v3-decoder.git
cd silk-v3-decoder/silk

# 编译编码器
make encoder
cp encoder /usr/local/bin/
chmod +x /usr/local/bin/encoder
encoder

```

### 2.4 部署应用

```bash
# 克隆应用代码
git clone https://github.com/yanyuwangluo/audio-converter.git

cd audio-converter

# 编译应用
go build -o audio-converter_1.0.1_linux_amd64

# 设置权限
chmod +x audio-converter_1.0.1_linux_amd64
```

## 3. 配置说明

### 3.1 目录结构
```
.
├── audio-converter    # 主程序
├── uploads/          # 上传文件临时目录
├── outputs/          # 转换后的文件目录
├── logs/             # 日志目录
└── static/           # 静态文件目录
```

## 4. 启动服务

### 4.1 直接启动
```bash
./audio-converter_1.0.1_linux_amd64

./audio-converter_1.0.1_linux_amd64 -port 8081
```

### 4.2 使用systemd管理（推荐）

创建服务文件 `/etc/systemd/system/audio-converter.service`：
```ini
[Unit]
Description=Audio Converter Service
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/path/to/your/app
ExecStart=/path/to/your/app/audio-converter
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable audio-converter
sudo systemctl start audio-converter
```

## 5. 使用说明

### 5.1 API接口

1. 文件上传转换：
```bash
curl -X POST -F "file=@/path/to/your/audio.mp3" http://localhost:8080/convert
```

2. URL转换：
```bash
curl -X POST -H "Content-Type: application/json" -d '{"url":"http://example.com/audio.mp3"}' http://localhost:8080/convert
```

3. 获取文件列表：
```bash
curl http://localhost:8080/api/files
```

4. 下载文件：
```bash
curl -O http://localhost:8080/download/filename.silk
```

### 5.2 Web界面
访问 `http://localhost:8080` 使用Web界面进行文件转换。

## 6. 常见问题

### 6.1 FFmpeg相关
- 如果遇到FFmpeg找不到的问题，检查FFmpeg是否正确安装：
```bash
ffmpeg -version
```

### 6.2 权限问题
- 确保应用有权限访问相关目录：
```bash
sudo chown -R your-user:your-group uploads outputs logs
```

### 6.3 端口占用
- 如果8080端口被占用，可以在启动时指定其他端口：
```bash
./audio-converter -port 8081
```

## 7. 维护说明

### 7.1 日志管理
- 日志文件位于 `logs` 目录
- 默认保留7天的日志
- 可以通过修改 `config.env` 调整日志级别

### 7.2 文件清理
- 定期清理 `uploads` 目录中的临时文件
- 定期备份 `outputs` 目录中的重要文件

### 7.3 性能优化
- 根据实际需求调整 `MAX_UPLOAD_SIZE` 限制
- 监控系统资源使用情况，必要时进行扩容

## 8. 安全建议

1. 配置防火墙：
```bash
# CentOS
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# Ubuntu
sudo ufw allow 8080
```

2. 使用HTTPS：
- 配置SSL证书
- 修改配置文件启用HTTPS

3. 定期更新：
- 及时更新系统和依赖包
- 关注安全漏洞公告 