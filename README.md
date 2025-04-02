# 音频转换系统

一个基于Go语言开发的音频转换系统，支持将各种音频格式转换为OPUS格式。系统提供Web界面和API接口，支持文件上传和URL转换两种方式。

## 功能特点

- 支持多种音频格式转换（MP3、WAV、OGG等）
- 提供Web界面和API接口
- 支持文件上传和URL转换
- 实时显示转换进度和状态
- 自动清理临时文件
- 详细的日志记录
- 支持自定义端口和文件大小限制

## 系统架构

- 前端：HTML5 + JavaScript
- 后端：Go语言
- 音频处理：FFmpeg + OPUS编码器
- 日志系统：自定义日志模块

## 部署

#####  <a href="bushu.md">部署说明</a>

## 启动
```bash
git clone https://github.com/yanyuwangluo/audio-converter.git

cd audio-converter

./audio-converter_1.0.1_linux_amd64
# 指定端口
./audio-converter_1.0.1_linux_amd64 -port 8081
```
## 目录结构

```
.
├── main.go              # 主程序入口
├── services/            # 服务层
│   └── audio_service.go # 音频转换服务
├── utils/              # 工具包
│   ├── logger.go       # 日志工具
│   └── network.go      # 网络工具
├── static/             # 静态文件
│   └── index.html      # Web界面
├── uploads/            # 上传文件临时目录
├── outputs/            # 转换后的文件目录
└── logs/               # 日志目录
```

## 技术特性

### 1. 音频转换
- 支持多种音频格式输入
- 使用FFmpeg进行音频预处理
- 使用OPUS编码器进行最终转换
- 自动检测音频时长和采样率

### 2. 文件管理
- 自动生成唯一文件名
- 定期清理临时文件
- 支持大文件上传
- 文件类型验证

### 3. 日志系统
- 分级日志记录（DEBUG、INFO、WARN、ERROR）
- 日志文件自动轮转
- 支持彩色输出
- 详细的错误追踪

### 4. 网络功能
- 支持HTTP/HTTPS
- 自动获取本地IP
- 支持跨域请求
- 文件下载断点续传

## 配置说明

### 环境变量
- `PORT`: 服务端口（默认8080）
- `MAX_UPLOAD_SIZE`: 最大上传文件大小（默认10MB）
- `UPLOAD_DIR`: 上传文件目录
- `OUTPUT_DIR`: 输出文件目录
- `LOG_DIR`: 日志目录
- `LOG_LEVEL`: 日志级别
- `LOG_COLOR`: 是否启用彩色日志

### 日志级别
- DEBUG: 调试信息
- INFO: 一般信息
- WARN: 警告信息
- ERROR: 错误信息
- FATAL: 致命错误

## 使用说明

### Web界面
1. 访问系统首页
2. 选择上传方式（文件或URL）
3. 等待转换完成
4. 下载转换后的文件

### API接口
1. 文件上传转换
2. URL转换
3. 获取文件列表
4. 下载文件

## 注意事项

1. 文件大小限制
   - 默认最大上传大小为10MB
   - 可通过配置文件修改

2. 文件清理
   - 临时文件自动清理
   - 建议定期备份重要文件

3. 系统要求
   - 确保足够的磁盘空间
   - 建议使用SSD存储

4. 安全建议
   - 建议启用HTTPS
   - 定期更新系统和依赖
   - 配置防火墙规则

## 更新日志

### v1.0.1
- 优化日志输出格式
- 改进文件清理机制
- 修复已知问题

### v1.0.0
- 初始版本发布
- 实现基本功能

## 许可证

MIT License

## 贡献指南

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建 Pull Request

## 联系方式

如有问题或建议，请提交 Issue 或联系管理员。 