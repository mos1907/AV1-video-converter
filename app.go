package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// VideoInfo struct
type VideoInfo struct {
	FullPath   string `json:"fullPath"`
	Duration   string `json:"duration"`
	FrameCount int    `json:"frameCount"`
	Codec      string `json:"codec"`
	Size       string `json:"size"`
}

// App struct
type App struct {
	ctx             context.Context
	appDir          string
	ffmpegPath      string
	ffprobePath     string
	logFile         *os.File
	configPath      string
	lastDestination string
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	executablePath, err := os.Executable()
	if err != nil {
		log.Fatal("Error getting executable path:", err)
	}
	a.appDir = filepath.Dir(executablePath)

	// MacOS uygulaması için özel durum
	if strings.HasSuffix(a.appDir, "MacOS") {
		a.appDir = filepath.Dir(filepath.Dir(a.appDir))
	}

	log.Printf("Current working directory: %s", a.appDir)
	log.Printf("Executable path: %s", os.Args[0])

	// Setup logging
	logsDir := filepath.Join(a.appDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatal("Error creating logs directory:", err)
	}

	// Temizleme işlemlerini gerçekleştir
	a.cleanupLogs(logsDir)

	// app.log dosyasını temizle ve yeniden aç
	appLogPath := filepath.Join(logsDir, "app.log")
	if err := os.Truncate(appLogPath, 0); err != nil {
		log.Printf("Error truncating app.log: %v", err)
	}

	a.logFile, err = os.OpenFile(appLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Error opening log file:", err)
	}
	log.SetOutput(a.logFile)

	// Find FFmpeg and FFprobe
	a.ffmpegPath = a.findExecutable("ffmpeg")
	a.ffprobePath = a.findExecutable("ffprobe")
	if a.ffmpegPath == "" || a.ffprobePath == "" {
		log.Fatal("FFmpeg or FFprobe not found. Please ensure both are installed and available in the application bundle or system PATH.")
	}
	log.Printf("Using FFmpeg: %s", a.ffmpegPath)
	log.Printf("Using FFprobe: %s", a.ffprobePath)

	// Load config
	a.configPath = filepath.Join(a.appDir, "config.json")
	a.loadConfig()
}

func (a *App) findExecutable(name string) string {
	possiblePaths := []string{
		filepath.Join(filepath.Dir(os.Args[0]), name), // App bundle's MacOS directory
		filepath.Join(a.appDir, name),                 // App directory
		filepath.Join("/usr/local/bin", name),         // Common location for user-installed software
		filepath.Join("/opt/homebrew/bin", name),      // Homebrew on Apple Silicon Macs
	}

	for _, path := range possiblePaths {
		log.Printf("Checking for %s at: %s", name, path)
		if _, err := os.Stat(path); err == nil {
			log.Printf("Found %s at: %s", name, path)
			return path
		}
	}

	// If not found in the above locations, check in system PATH
	path, err := exec.LookPath(name)
	if err == nil {
		log.Printf("Found %s in PATH: %s", name, path)
		return path
	}

	log.Printf("%s not found in any expected location or system PATH", name)
	return ""
}

func (a *App) cleanupLogs(logsDir string) {
	files, err := ioutil.ReadDir(logsDir)
	if err != nil {
		log.Printf("Error reading logs directory: %v", err)
		return
	}

	now := time.Now()
	for _, file := range files {
		if file.Name() == "app.log" {
			continue // app.log'u atla, bu ayrı işleme tabi tutulacak
		}

		filePath := filepath.Join(logsDir, file.Name())
		if now.Sub(file.ModTime()) > 24*time.Hour {
			if err := os.Remove(filePath); err != nil {
				log.Printf("Error removing old log file %s: %v", filePath, err)
			} else {
				log.Printf("Removed old log file: %s", filePath)
			}
		}
	}
}

func (a *App) loadConfig() {
	data, err := ioutil.ReadFile(a.configPath)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
		return
	}

	var config struct {
		LastDestination string `json:"lastDestination"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Error unmarshalling config: %v", err)
		return
	}

	a.lastDestination = config.LastDestination
}

func (a *App) saveConfig() {
	config := struct {
		LastDestination string `json:"lastDestination"`
	}{
		LastDestination: a.lastDestination,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("Error marshalling config: %v", err)
		return
	}

	if err := ioutil.WriteFile(a.configPath, data, 0644); err != nil {
		log.Printf("Error writing config file: %v", err)
	}
}

// shutdown is called at application termination
func (a *App) shutdown(ctx context.Context) {
	if a.logFile != nil {
		a.logFile.Close()
	}
}

// SelectVideoFiles opens a file dialog and returns the selected video files info
func (a *App) SelectVideoFiles() ([]VideoInfo, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Video Files",
		Filters: []runtime.FileFilter{
			{DisplayName: "Video Files", Pattern: "*.mp4;*.avi;*.mov;*.mkv"},
		},
	})
	if err != nil {
		log.Printf("Error selecting files: %v", err)
		return nil, err
	}

	var videoInfos []VideoInfo
	for _, file := range files {
		log.Printf("Processing file: %s", file)
		if _, err := os.Stat(file); os.IsNotExist(err) {
			log.Printf("File does not exist: %s", file)
			continue
		}
		info, err := a.getVideoInfo(file)
		if err != nil {
			log.Printf("Error getting info for %s: %v", file, err)
			continue
		}
		videoInfos = append(videoInfos, info)
		log.Printf("Successfully processed file: %s", file)
	}

	return videoInfos, nil
}

func (a *App) getVideoInfo(filePath string) (VideoInfo, error) {
	cmd := exec.Command(a.ffprobePath, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("Error running FFprobe command: %v", err)
		log.Printf("FFprobe command: %v", cmd.Args)
		log.Printf("FFprobe stderr: %s", stderr.String())
		return VideoInfo{}, fmt.Errorf("FFprobe error: %v, stderr: %s", err, stderr.String())
	}

	output := stdout.Bytes()
	var result struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			NbFrames  string `json:"nb_frames"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		log.Printf("FFprobe output: %s", output)
		return VideoInfo{}, err
	}

	if len(result.Streams) == 0 {
		return VideoInfo{}, fmt.Errorf("no streams found in the video file")
	}

	frameCount, _ := strconv.Atoi(result.Streams[0].NbFrames)
	sizeInBytes, _ := strconv.ParseFloat(result.Format.Size, 64)
	sizeInMB := sizeInBytes / 1024 / 1024

	return VideoInfo{
		FullPath:   filePath,
		Duration:   result.Format.Duration,
		FrameCount: frameCount,
		Codec:      result.Streams[0].CodecName,
		Size:       fmt.Sprintf("%.2f MB", sizeInMB),
	}, nil
}

// SelectDestinationFolder opens a directory dialog and returns the selected folder
func (a *App) SelectDestinationFolder() (string, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Destination Folder",
	})
	if err != nil {
		log.Printf("Error selecting destination folder: %v", err)
		return "", err
	}

	if folder == "" {
		// Kullanıcı bir klasör seçmedi, son seçilen klasörü kullan
		folder = a.lastDestination
		if folder == "" {
			// Eğer son seçilen klasör de yoksa, varsayılan bir klasör kullan
			folder = filepath.Join(os.Getenv("HOME"), "Desktop")
		}
	}

	// Check if the folder is writable
	testFile := filepath.Join(folder, "test_write_permission.tmp")
	f, err := os.Create(testFile)
	if err != nil {
		log.Printf("Selected folder is not writable: %v", err)
		return "", fmt.Errorf("selected folder is not writable: %v", err)
	}
	f.Close()
	os.Remove(testFile)

	a.lastDestination = folder
	a.saveConfig()

	return folder, nil
}

// GetLastDestination returns the last selected destination folder
func (a *App) GetLastDestination() string {
	return a.lastDestination
}

// ConvertVideo converts the input video to SVTAV1 format
func (a *App) ConvertVideo(inputPath, outputFolder string, totalFrames int) error {
	outputFileName := filepath.Base(inputPath)
	outputFileName = strings.TrimSuffix(outputFileName, filepath.Ext(outputFileName))
	outputFileName = sanitizeFileName(outputFileName)
	outputPath := filepath.Join(outputFolder, outputFileName+"_av1.mp4")

	if err := os.MkdirAll(outputFolder, os.ModePerm); err != nil {
		log.Printf("Failed to create output directory: %v", err)
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	logFileName := outputFileName + "_ffmpeg.log"
	logFilePath := filepath.Join(a.appDir, "logs", logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return fmt.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()

	cmd := exec.Command(a.ffmpegPath,
		"-i", inputPath,
		"-c:v", "libsvtav1",
		"-crf", "30",
		"-preset", "6",
		"-svtav1-params", "tune=0",
		"-c:a", "copy", "-y",
		outputPath)

	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	done := make(chan bool)
	go func() {
		a.monitorProgress(logFilePath, totalFrames, done)
	}()

	if err := cmd.Wait(); err != nil {
		close(done)
		log.Printf("FFmpeg error: %v", err)
		runtime.EventsEmit(a.ctx, "conversion:error", err.Error())
		return fmt.Errorf("FFmpeg error: %v", err)
	}

	close(done)
	time.Sleep(time.Second) // İlerleme çubuğunun %100'e ulaşması için kısa bir bekleme
	runtime.EventsEmit(a.ctx, "conversion:complete", outputPath)
	log.Printf("Conversion completed: %s", outputPath)

	// Sıradaki videoyu işleme almak için event gönder
	runtime.EventsEmit(a.ctx, "conversion:next")

	return nil
}

func (a *App) monitorProgress(logPath string, totalFrames int, done chan bool) {
	file, err := os.Open(logPath)
	if err != nil {
		log.Printf("Error opening log file: %v", err)
		return
	}
	defer file.Close()

	// Frame ve speed için regex'ler
	frameRegex := regexp.MustCompile(`frame=\s*(\d+)`)
	speedRegex := regexp.MustCompile(`speed=(\S+)`)

	var lastProgress float64
	for {
		select {
		case <-done:
			runtime.EventsEmit(a.ctx, "conversion:progress", map[string]interface{}{
				"progress": 100,
				"speed":    "",
			})
			return
		default:
			file.Seek(-1024, 2) // Son 1024 byte'ı oku
			scanner := bufio.NewScanner(file)
			var lastLine string
			for scanner.Scan() {
				lastLine = scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				log.Printf("Error scanning file: %v", err)
				continue
			}

			//fmt.Println("Son satır:", lastLine) // Debug için

			if strings.Contains(lastLine, "frame=") {
				frameMatch := frameRegex.FindStringSubmatch(lastLine)
				speedMatch := speedRegex.FindStringSubmatch(lastLine)

				//fmt.Println("Frame eşleşmesi:", frameMatch) // Debug için
				//fmt.Println("Speed eşleşmesi:", speedMatch) // Debug için

				if len(frameMatch) > 1 && len(speedMatch) > 1 {
					currentFrame, err := strconv.ParseFloat(frameMatch[1], 64)
					if err != nil {
						log.Printf("Error parsing frame: %v", err)
						continue
					}

					speed := strings.TrimSpace(speedMatch[1])

					progress := (currentFrame / float64(totalFrames)) * 100
					if progress > 100 {
						progress = 100
					}

					if progress > lastProgress {
						lastProgress = progress
						fmt.Printf("İlerleme: %.2f%%, Hız: %s\n", progress, speed) // Debug için
						runtime.EventsEmit(a.ctx, "conversion:progress", map[string]interface{}{
							"progress": progress,
							"speed":    speed,
						})
					}
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func sanitizeFileName(fileName string) string {
	// Remove or replace invalid characters
	fileName = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, fileName)

	// Truncate filename if it's too long
	if len(fileName) > 200 {
		fileName = fileName[:200]
	}

	return fileName
}
