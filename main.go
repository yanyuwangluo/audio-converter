package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"audio-converter/services"
	"audio-converter/utils"

	"github.com/gin-gonic/gin"
)

var (
	// 命令行参数
	port     = flag.String("port", "8080", "服务器端口")
	debug    = flag.Bool("debug", true, "是否开启调试模式")
	logLevel = flag.Int("log-level", utils.LevelDebug, "日志级别: 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR")
	noColor  = flag.Bool("no-color", false, "禁用彩色日志输出")

	// 目录配置
	uploadDir = "./uploads"
	silkDir   = "./outputs"
	logsDir   = "./logs"
	cacheTime = 24 * time.Hour // 缓存时间，默认24小时

	// 服务实例
	audioService *services.AudioService
)

// 初始化服务
func initService() {
	// 确保目录存在
	os.MkdirAll(uploadDir, 0755)
	os.MkdirAll(silkDir, 0755)
	os.MkdirAll(logsDir, 0755)

	// 初始化日志
	if err := utils.InitLogger(logsDir); err != nil {
		fmt.Printf("初始化日志失败: %v\n", err)
		os.Exit(1)
	}

	// 设置日志级别
	utils.SetLevel(*logLevel)

	// 设置日志颜色
	utils.EnableColor(!*noColor)

	// 输出初始化信息
	utils.Info("==== 音频转换服务初始化 ====")
	utils.Info("调试模式: %v", *debug)
	utils.Info("日志级别: %s", utils.LevelNames[*logLevel])
	utils.Info("彩色日志: %v", !*noColor)

	// 创建音频服务实例
	audioService = services.NewAudioService(uploadDir, silkDir)

	// 启动定时清理任务
	go startCleaner()
}

// 周期性清理临时文件
func startCleaner() {
	ticker := time.NewTicker(1 * time.Hour)
	utils.Debug("启动文件清理定时任务，间隔: 1小时")

	for range ticker.C {
		cleanTempFiles()
	}
}

// 清理临时文件
func cleanTempFiles() {
	utils.Debug("开始执行清理任务")
	cleanDir(uploadDir)
	cleanDir(silkDir)
	utils.CleanOldLogs(logsDir)
	utils.Debug("清理任务完成")
}

// 清理指定目录中的旧文件
func cleanDir(dir string) {
	utils.Debug("清理目录: %s", dir)
	files, err := os.ReadDir(dir)
	if err != nil {
		utils.Error("读取目录失败: %v", err)
		return
	}

	now := time.Now()
	count := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		path := filepath.Join(dir, file.Name())
		info, err := file.Info()
		if err != nil {
			continue
		}

		if now.Sub(info.ModTime()) > cacheTime {
			if err := os.Remove(path); err != nil {
				utils.Error("删除文件失败: %s: %v", path, err)
			} else {
				count++
				utils.Debug("已删除过期文件: %s", path)
			}
		}
	}

	if count > 0 {
		utils.Info("目录 %s 中已清理 %d 个过期文件", dir, count)
	}
}

// 处理首页请求
func handleIndex(c *gin.Context) {
	utils.Debug("处理首页请求: %s", c.Request.RemoteAddr)
	c.File("static/index.html")
}

// 处理上传文件请求
func handleUpload(c *gin.Context) {
	clientIP := c.ClientIP()
	utils.Info("收到文件上传请求: %s", clientIP)

	file, err := c.FormFile("file")
	if err != nil {
		utils.Error("上传文件失败: %s: %v", clientIP, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "上传文件失败: " + err.Error()})
		return
	}

	utils.Info("上传文件: %s, 大小: %.2f KB", file.Filename, float64(file.Size)/1024)

	// 打开上传的文件
	src, err := file.Open()
	if err != nil {
		utils.Error("无法读取上传的文件: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取上传的文件: " + err.Error()})
		return
	}
	defer src.Close()

	// 读取文件内容
	content, err := io.ReadAll(src)
	if err != nil {
		utils.Error("无法读取文件内容: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法读取文件内容: " + err.Error()})
		return
	}

	utils.Info("开始转换音频: %s", file.Filename)

	// 调用音频转换服务
	startTime := time.Now()
	filename, err := audioService.ConvertToSilk(content)
	if err != nil {
		utils.Error("音频转换失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "音频转换失败: " + err.Error()})
		return
	}

	// 计算处理时间
	duration := time.Since(startTime)

	// 构建下载URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	downloadURL := fmt.Sprintf("%s://%s/download/%s", scheme, host, filename)

	utils.Info("音频转换成功: %s -> %s (耗时: %.2f秒)", file.Filename, filename, duration.Seconds())
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"url":      downloadURL,
		"filename": filename,
		"duration": fmt.Sprintf("%.2f秒", duration.Seconds()),
	})
}

// 处理URL转换请求
func handleURL(c *gin.Context) {
	clientIP := c.ClientIP()

	var req struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error("无效的URL请求参数: %s: %v", clientIP, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	utils.Info("收到URL转换请求: %s, URL: %s", clientIP, req.URL)

	// 调用音频转换服务
	startTime := time.Now()
	filename, err := audioService.ConvertToSilk(req.URL)
	if err != nil {
		utils.Error("URL音频转换失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "音频转换失败: " + err.Error()})
		return
	}

	// 计算处理时间
	duration := time.Since(startTime)

	// 构建下载URL
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	host := c.Request.Host
	downloadURL := fmt.Sprintf("%s://%s/download/%s", scheme, host, filename)

	utils.Info("URL音频转换成功: %s (耗时: %.2f秒)", filename, duration.Seconds())
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"url":      downloadURL,
		"filename": filename,
		"duration": fmt.Sprintf("%.2f秒", duration.Seconds()),
	})
}

// 处理文本转语音请求
func handleTTS(c *gin.Context) {
	clientIP := c.ClientIP()

	var req struct {
		Text string `json:"text" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.Error("无效的TTS请求参数: %s: %v", clientIP, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	utils.Info("收到TTS请求: %s, 文本长度: %d", clientIP, len(req.Text))
	utils.Warn("TTS功能尚未实现")

	// 这里可以集成第三方TTS服务
	// 暂时返回不支持
	c.JSON(http.StatusNotImplemented, gin.H{"error": "TTS功能尚未实现"})
}

// 获取本地IP地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		utils.Error("获取本地IP失败: %v", err)
		return "localhost"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}

// 处理音频转换
func handleConvert(c *gin.Context) {
	startTime := time.Now()
	var input interface{}
	var err error

	// 检查请求的Content-Type
	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		// 处理文件上传
		file, err := c.FormFile("file")
		if err != nil {
			utils.Error("获取上传文件失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "获取上传文件失败",
			})
			return
		}

		// 保存上传的文件
		filename := filepath.Base(file.Filename)
		filepath := filepath.Join(uploadDir, filename)
		if err := c.SaveUploadedFile(file, filepath); err != nil {
			utils.Error("保存上传文件失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "保存上传文件失败",
			})
			return
		}
		input = filepath
	} else if strings.Contains(contentType, "application/json") {
		// 处理URL
		var request struct {
			URL string `json:"url"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			utils.Error("解析URL请求失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "无效的URL",
			})
			return
		}
		input = request.URL
	} else {
		utils.Error("不支持的Content-Type: %s", contentType)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "不支持的请求类型",
		})
		return
	}

	// 转换音频
	filename, err := audioService.ConvertToSilk(input)
	if err != nil {
		utils.Error("音频转换失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "音频转换失败: " + err.Error(),
		})
		return
	}

	// 生成下载URL
	localIP := getLocalIP()
	downloadURL := fmt.Sprintf("http://%s:%s/download/%s", localIP, *port, filename)
	duration := time.Since(startTime).String()

	utils.Info("音频转换成功: %s", downloadURL)
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"url":      downloadURL,
		"filename": filename,
		"duration": duration,
	})
}

// 处理下载请求
func handleDownload(c *gin.Context) {
	clientIP := c.ClientIP()
	filename := c.Param("filename")

	if filename == "" {
		utils.Error("下载请求缺少文件名: %s", clientIP)
		c.JSON(http.StatusBadRequest, gin.H{"error": "未指定文件名"})
		return
	}

	utils.Debug("收到下载请求: %s, 文件: %s", clientIP, filename)

	// 安全检查：防止目录遍历攻击
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		utils.Warn("检测到不安全的文件名请求: %s, 文件: %s", clientIP, filename)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的文件名"})
		return
	}

	// 构建文件路径
	filePath := filepath.Join(silkDir, filename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		utils.Warn("请求的文件不存在: %s, 文件: %s", clientIP, filePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 设置文件名和内容类型
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Type", "application/octet-stream")

	utils.Info("提供文件下载: %s -> %s", filename, clientIP)

	// 提供文件下载
	c.File(filePath)
}

// 获取文件列表
func handleGetFiles(c *gin.Context) {
	// 获取上传目录的文件列表
	uploadFiles, err := getFileList(uploadDir)
	if err != nil {
		utils.Error("获取上传文件列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取上传文件列表失败",
		})
		return
	}

	// 获取SILK目录的文件列表
	silkFiles, err := getFileList(silkDir)
	if err != nil {
		utils.Error("获取SILK文件列表失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "获取SILK文件列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"uploads":    uploadFiles,
		"silk_files": silkFiles,
	})
}

// 获取目录下的文件列表
func getFileList(dir string) ([]map[string]interface{}, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var fileList []map[string]interface{}
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err != nil {
				continue
			}
			fileList = append(fileList, map[string]interface{}{
				"name": file.Name(),
				"time": info.ModTime().Unix() * 1000, // 转换为毫秒时间戳
			})
		}
	}
	return fileList, nil
}

// 设置路由
func setupRouter() *gin.Engine {
	// 根据调试模式设置gin模式
	if !*debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 设置静态文件路由
	r.Static("/static", "./static")

	// 配置API路由
	r.GET("/", handleIndex)
	r.POST("/upload", handleUpload)
	r.POST("/url", handleURL)
	r.POST("/tts", handleTTS)
	r.POST("/convert", handleConvert)
	r.GET("/download/:filename", handleDownload)
	r.GET("/api/files", handleGetFiles)

	return r
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 初始化服务
	initService()
	defer utils.CloseLogger()

	// 设置路由
	r := setupRouter()

	// 设置优雅关闭
	srv := &http.Server{
		Addr:    ":" + *port,
		Handler: r,
	}

	// 启动服务器
	go func() {
		utils.Info("音频转换服务启动在 http://localhost:%s", *port)
		utils.Info("上传目录: %s", uploadDir)
		utils.Info("输出目录: %s", silkDir)

		// 在调试模式下显示路由信息
		if *debug {
			utils.Debug("路由配置:")
			utils.Debug("  GET  /                - 首页")
			utils.Debug("  POST /upload          - 文件上传接口")
			utils.Debug("  POST /url             - URL转换接口")
			utils.Debug("  POST /tts             - 文本转语音接口(未实现)")
			utils.Debug("  POST /convert         - 音频转换接口")
			utils.Debug("  GET  /download/:file  - 文件下载接口")
			utils.Debug("  GET  /api/files       - 文件列表接口")
			utils.Debug("  GET  /static/*file    - 静态资源")
		}

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Fatal("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	utils.Info("正在关闭服务器...")

	// 关闭前执行清理任务
	cleanTempFiles()

	utils.Info("服务器已关闭")
}
