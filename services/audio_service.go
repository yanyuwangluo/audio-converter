package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"audio-converter/utils"
)

type AudioService struct {
	UploadDir   string
	SilkDir     string
	FfmpegPath  string
	EncoderPath string
}

// NewAudioService 创建新的音频服务
func NewAudioService(uploadDir, silkDir string) *AudioService {
	// 确保目录使用绝对路径
	absUploadDir, _ := filepath.Abs(uploadDir)
	absSilkDir, _ := filepath.Abs(silkDir)

	// 设置ffmpeg和encoder路径
	var ffmpegPath, encoderPath string

	// 检测操作系统类型并设置相应的路径
	if runtime.GOOS == "windows" {
		// Windows环境
		ffmpegPath = "D:\\ffmpeg-7.1.1-essentials_build\\bin\\ffmpeg.exe"
		encoderPath = "D:\\silk\\encoder.exe"
	} else {
		// Linux/Unix环境
		ffmpegPath = "/usr/bin/ffmpeg"
		encoderPath = "/usr/local/bin/encoder"
	}

	// 尝试在PATH中查找ffmpeg和encoder
	if ffPath, err := exec.LookPath("ffmpeg"); err == nil {
		ffmpegPath = ffPath
		utils.Info("在PATH中找到ffmpeg: %s", ffmpegPath)
	}

	if encPath, err := exec.LookPath("encoder"); err == nil {
		encoderPath = encPath
		utils.Info("在PATH中找到encoder: %s", encoderPath)
	}

	utils.Info("音频服务初始化: 上传目录=%s, SILK目录=%s", absUploadDir, absSilkDir)
	utils.Debug("FFmpeg路径: %s", ffmpegPath)
	utils.Debug("Encoder路径: %s", encoderPath)

	return &AudioService{
		UploadDir:   absUploadDir,
		SilkDir:     absSilkDir,
		FfmpegPath:  ffmpegPath,
		EncoderPath: encoderPath,
	}
}

// 获取文件扩展名
func getFileExt(filename string) string {
	return strings.ToLower(filepath.Ext(filename))
}

// 获取文件名（不含路径）
func getFileName(filePath string) string {
	return filepath.Base(filePath)
}

// ConvertToSilk 将音频转换为SILK格式
func (s *AudioService) ConvertToSilk(input interface{}) (string, error) {
	var inputPath string
	var err error

	switch v := input.(type) {
	case string:
		// 如果是URL
		if strings.HasPrefix(v, "http") {
			inputPath, err = s.downloadFromURL(v)
			if err != nil {
				utils.Error("下载URL失败: %v", err)
				return "", err
			}
			utils.Info("已下载文件: %s", inputPath)
		} else {
			// 如果是本地文件路径
			inputPath = v
			utils.Debug("使用本地文件: %s", inputPath)
		}
	case []byte:
		// 如果是文件内容
		inputPath, err = s.saveUploadedFile(v)
		if err != nil {
			utils.Error("保存上传文件失败: %v", err)
			return "", err
		}
		utils.Debug("已保存上传文件: %s", inputPath)
	default:
		return "", fmt.Errorf("不支持的输入类型")
	}

	// 生成输出文件名 (使用年月日时分秒格式)
	now := time.Now()
	outputFilename := fmt.Sprintf("%d%02d%02d_%02d%02d%02d.silk",
		now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), now.Second())
	outputPath := filepath.Join(s.SilkDir, outputFilename)
	utils.Debug("输出文件路径: %s", outputPath)

	// 创建临时PCM文件
	pcmFilename := fmt.Sprintf("%d.pcm", time.Now().UnixNano())
	pcmPath := filepath.Join(s.UploadDir, pcmFilename)
	utils.Debug("创建临时PCM文件: %s", pcmPath)

	// 第一步: 使用ffmpeg转换音频为PCM格式
	cmd := exec.Command(s.FfmpegPath, "-i", inputPath,
		"-f", "s16le", // 强制16位小端PCM格式
		"-acodec", "pcm_s16le", // PCM 16位有符号整数小端格式
		"-ar", "24000", // 采样率24kHz
		"-ac", "1", // 单声道
		pcmPath) // 输出到PCM文件

	// 捕获命令输出以便记录
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("创建stdout管道失败: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("创建stderr管道失败: %v", err)
	}

	utils.Debug("执行FFmpeg命令: %s", cmd.String())
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("启动ffmpeg失败: %v", err)
	}

	// 记录输出
	go s.logOutput(stdout, false)
	go s.logOutput(stderr, true)

	if err := cmd.Wait(); err != nil {
		os.Remove(pcmPath)
		return "", fmt.Errorf("PCM转换失败: %v", err)
	}
	utils.Info("FFmpeg转换为PCM完成")

	// 第二步: 使用encoder将PCM转换为SILK格式
	// 根据不同平台使用不同的命令参数
	var encoderCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		encoderCmd = exec.Command(s.EncoderPath, pcmPath, outputPath, "-tencent")
	} else {
		// Linux下的encoder可能有不同的参数格式
		encoderCmd = exec.Command(s.EncoderPath, pcmPath, outputPath, "-tencent")
	}

	// 捕获命令输出以便记录
	stdout, err = encoderCmd.StdoutPipe()
	if err != nil {
		os.Remove(pcmPath)
		return "", fmt.Errorf("创建stdout管道失败: %v", err)
	}
	stderr, err = encoderCmd.StderrPipe()
	if err != nil {
		os.Remove(pcmPath)
		return "", fmt.Errorf("创建stderr管道失败: %v", err)
	}

	utils.Debug("执行Encoder命令: %s", encoderCmd.String())
	if err := encoderCmd.Start(); err != nil {
		os.Remove(pcmPath)
		return "", fmt.Errorf("启动encoder失败: %v", err)
	}

	// 记录输出
	go s.logOutput(stdout, false)
	go s.logOutput(stderr, true)

	if err := encoderCmd.Wait(); err != nil {
		os.Remove(pcmPath)
		return "", fmt.Errorf("SILK转换失败: %v", err)
	}
	utils.Info("PCM转换为SILK完成")

	// 清理临时文件
	os.Remove(pcmPath)
	os.Remove(inputPath)
	utils.Debug("临时文件已清理")

	// 检查输出文件是否存在
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		utils.Error("输出文件未生成: %v", err)
		return "", fmt.Errorf("转换失败：输出文件未生成")
	}

	utils.Info("音频转换成功: %s", outputFilename)
	return outputFilename, nil
}

// logOutput 记录命令输出到日志
func (s *AudioService) logOutput(r io.Reader, isError bool) {
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output := strings.TrimSpace(string(buf[:n]))
			if output != "" {
				if isError {
					utils.Debug("命令错误输出: %s", output)
				} else {
					utils.Debug("命令标准输出: %s", output)
				}
			}
		}
		if err != nil {
			break
		}
	}
}

// downloadFromURL 从URL下载文件
func (s *AudioService) downloadFromURL(url string) (string, error) {
	utils.Info("开始下载文件: %s", url)

	resp, err := http.Get(url)
	if err != nil {
		utils.Error("HTTP请求失败: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), filepath.Ext(url))
	filepath := filepath.Join(s.UploadDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		utils.Error("创建文件失败: %v", err)
		return "", err
	}
	defer file.Close()

	size, err := io.Copy(file, resp.Body)
	if err != nil {
		utils.Error("保存文件失败: %v", err)
		return "", err
	}

	utils.Info("文件下载完成: %s (大小: %d 字节)", filename, size)
	return filepath, nil
}

// saveUploadedFile 保存上传的文件
func (s *AudioService) saveUploadedFile(content []byte) (string, error) {
	filename := fmt.Sprintf("%d.wav", time.Now().UnixNano())
	filepath := filepath.Join(s.UploadDir, filename)

	utils.Debug("保存上传文件: %s (大小: %d 字节)", filename, len(content))

	err := os.WriteFile(filepath, content, 0644)
	if err != nil {
		utils.Error("写入文件失败: %v", err)
		return "", err
	}

	return filepath, nil
}
