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
// Represents information about a video file
// Bir video dosyası hakkında bilgileri temsil eder
type VideoInfo struct {
	FullPath   string `json:"fullPath"`   // Full path of the video file / Video dosyasının tam yolu
	Duration   string `json:"duration"`   // Duration of the video / Videonun süresi
	FrameCount int    `json:"frameCount"` // Total number of frames / Toplam kare sayısı
	Codec      string `json:"codec"`      // Video codec / Video kodeki
	Size       string `json:"size"`       // File size / Dosya boyutu
}

// App struct
// Represents the main application structure
// Ana uygulama yapısını temsil eder
type App struct {
	ctx             context.Context // Application context / Uygulama bağlamı
	appDir          string          // Application directory / Uygulama dizini
	ffmpegPath      string          // Path to FFmpeg executable / FFmpeg yürütülebilir dosyasının yolu
	ffprobePath     string          // Path to FFprobe executable / FFprobe yürütülebilir dosyasının yolu
	logFile         *os.File        // Log file / Log dosyası
	configPath      string          // Path to config file / Yapılandırma dosyasının yolu
	lastDestination string          // Last used destination folder / Son kullanılan hedef klasör
}

// NewApp creates a new App application struct
// Creates and returns a new instance of the App struct
// App yapısının yeni bir örneğini oluşturur ve döndürür
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
// Initializes the application, sets up logging, and finds FFmpeg/FFprobe
// Uygulama başladığında çağrılır, Log kaydını ayarlar ve FFmpeg/FFprobe'u bulur
func (a *App) startup(ctx context.Context) {
	// Save the context
	// Bağlamı kaydet
	a.ctx = ctx

	// Get the executable path
	// Yürütülebilir dosya yolunu al
	executablePath, err := os.Executable()
	if err != nil {
		log.Fatal("Error getting executable path:", err)
	}
	a.appDir = filepath.Dir(executablePath)

	// Special case for MacOS application
	// MacOS uygulaması için özel durum
	if strings.HasSuffix(a.appDir, "MacOS") {
		a.appDir = filepath.Dir(filepath.Dir(a.appDir))
	}

	// Log current working directory and executable path
	// Mevcut çalışma dizinini ve yürütülebilir dosya yolunu günlüğe kaydet
	log.Printf("Current working directory: %s", a.appDir)
	log.Printf("Executable path: %s", os.Args[0])

	// Setup logging
	// Log kaydını ayarla
	logsDir := filepath.Join(a.appDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.Fatal("Error creating logs directory:", err)
	}

	// Perform cleanup operations
	// Temizleme işlemlerini gerçekleştir
	a.cleanupLogs(logsDir)

	// Clear and reopen app.log file
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
	// FFmpeg ve FFprobe'u bul
	a.ffmpegPath = a.findExecutable("ffmpeg")
	a.ffprobePath = a.findExecutable("ffprobe")
	if a.ffmpegPath == "" || a.ffprobePath == "" {
		log.Fatal("FFmpeg or FFprobe not found. Please ensure both are installed and available in the application bundle or system PATH.")
	}
	log.Printf("Using FFmpeg: %s", a.ffmpegPath)
	log.Printf("Using FFprobe: %s", a.ffprobePath)

	// Load config
	// Yapılandırmayı yükle
	a.configPath = filepath.Join(a.appDir, "config.json")
	a.loadConfig()
}

// findExecutable locates the specified executable in various paths
// Searches for the given executable in predefined paths and PATH
// Belirtilen yürütülebilir dosyayı çeşitli yollarda arar
func (a *App) findExecutable(name string) string {
	// Define possible paths for the executable
	// Yürütülebilir dosya için olası yolları tanımla
	possiblePaths := []string{
		filepath.Join(filepath.Dir(os.Args[0]), name),
		filepath.Join(a.appDir, name),
		filepath.Join("/usr/local/bin", name),
		filepath.Join("/opt/homebrew/bin", name),
	}

	// Check each possible path
	// Her olası yolu kontrol et
	for _, path := range possiblePaths {
		log.Printf("Checking for %s at: %s", name, path)
		if _, err := os.Stat(path); err == nil {
			log.Printf("Found %s at: %s", name, path)
			return path
		}
	}

	// If not found in the above locations, check in system PATH
	// Yukarıdaki konumlarda bulunamazsa, sistem PATH'inde kontrol et
	path, err := exec.LookPath(name)
	if err == nil {
		log.Printf("Found %s in PATH: %s", name, path)
		return path
	}

	log.Printf("%s not found in any expected location or system PATH", name)
	return ""
}

// cleanupLogs removes old log files
// Deletes log files older than 24 hours, except for app.log
// 24 saatten eski Log dosyalarını siler, app.log hariç
func (a *App) cleanupLogs(logsDir string) {
	// Read all files in the logs directory
	// Log dizinindeki tüm dosyaları oku
	files, err := ioutil.ReadDir(logsDir)
	if err != nil {
		log.Printf("Error reading logs directory: %v", err)
		return
	}

	now := time.Now()
	for _, file := range files {
		if file.Name() == "app.log" {
			continue // Skip app.log / app.log'u atla
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

// loadConfig reads the configuration file
// Loads the last used destination folder from the config file
// Yapılandırma dosyasından son kullanılan hedef klasörü yükler
func (a *App) loadConfig() {
	// Read the config file
	// Yapılandırma dosyasını oku
	data, err := ioutil.ReadFile(a.configPath)
	if err != nil {
		log.Printf("Error reading config file: %v", err)
		return
	}

	// Unmarshal the JSON data
	// JSON verisini çöz
	var config struct {
		LastDestination string `json:"lastDestination"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("Error unmarshalling config: %v", err)
		return
	}

	// Set the last destination
	// Son hedefi ayarla
	a.lastDestination = config.LastDestination
}

// saveConfig writes the current configuration to file
// Saves the current destination folder to the config file
// Mevcut hedef klasörü yapılandırma dosyasına kaydeder
func (a *App) saveConfig() {
	// Prepare the config data
	// Yapılandırma verisini hazırla
	config := struct {
		LastDestination string `json:"lastDestination"`
	}{
		LastDestination: a.lastDestination,
	}

	// Marshal the config to JSON
	// Yapılandırmayı JSON'a dönüştür
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		log.Printf("Error marshalling config: %v", err)
		return
	}

	// Write the config to file
	// Yapılandırmayı dosyaya yaz
	if err := ioutil.WriteFile(a.configPath, data, 0644); err != nil {
		log.Printf("Error writing config file: %v", err)
	}
}

// shutdown is called at application termination
// Performs cleanup operations when the application is closing
// Uygulama kapanırken temizleme işlemlerini gerçekleştirir
func (a *App) shutdown(ctx context.Context) {
	// Close the log file if it's open
	// Log dosyası açıksa kapat
	if a.logFile != nil {
		a.logFile.Close()
	}
}

// SelectVideoFiles opens a file dialog and returns the selected video files info
// Allows user to select multiple video files and returns their information
// Kullanıcının birden çok video dosyası seçmesine izin verir ve bilgilerini döndürür
func (a *App) SelectVideoFiles() ([]VideoInfo, error) {
	// Open file dialog for selecting video files
	// Video dosyaları seçmek için dosya iletişim kutusunu aç
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

	// Process selected files
	// Seçilen dosyaları işle
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

	// Return the video information to the frontend
	// Video bilgilerini Frontend'e gönder
	return videoInfos, nil
}

// getVideoInfo retrieves detailed information about a video file
// Uses FFprobe to get video metadata such as duration, frame count, codec, and size
// FFprobe kullanarak video meta verilerini (süre, kare sayısı, kodek, boyut) alır
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

	var result struct {
		Streams []struct {
			CodecName    string `json:"codec_name"`
			NbFrames     string `json:"nb_frames"`
			AvgFrameRate string `json:"avg_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
		} `json:"format"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		log.Printf("Error unmarshalling JSON: %v", err)
		log.Printf("FFprobe output: %s", stdout.String())
		return VideoInfo{}, err
	}

	if len(result.Streams) == 0 {
		return VideoInfo{}, fmt.Errorf("no streams found in the video file")
	}

	durationInSeconds, _ := strconv.ParseFloat(result.Format.Duration, 64)
	frameRate, _ := strconv.ParseFloat(result.Streams[0].AvgFrameRate, 64)

	hours := int(durationInSeconds) / 3600
	minutes := (int(durationInSeconds) % 3600) / 60
	seconds := int(durationInSeconds) % 60
	frames := int((durationInSeconds - float64(int(durationInSeconds))) * frameRate)

	timecode := fmt.Sprintf("%02d:%02d:%02d:%02d", hours, minutes, seconds, frames)

	frameCount, _ := strconv.Atoi(result.Streams[0].NbFrames)
	sizeInBytes, _ := strconv.ParseFloat(result.Format.Size, 64)
	sizeInMB := sizeInBytes / 1024 / 1024

	return VideoInfo{
		FullPath:   filePath,
		Duration:   timecode,
		FrameCount: frameCount,
		Codec:      result.Streams[0].CodecName,
		Size:       fmt.Sprintf("%.2f MB", sizeInMB),
	}, nil
}

// SelectDestinationFolder opens a directory dialog and returns the selected folder
// Allows user to choose a destination folder for converted videos
// Kullanıcının dönüştürülen videolar için bir hedef klasör seçmesine izin verir
func (a *App) SelectDestinationFolder() (string, error) {
	// Open directory dialog
	// Dizin seçim penceresini aç
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Destination Folder",
	})
	if err != nil {
		log.Printf("Error selecting destination folder: %v", err)
		return "", err
	}

	// If no folder selected, use last destination or default
	// Eğer klasör seçilmediyse, son hedefi veya varsayılanı kullan
	if folder == "" {
		folder = a.lastDestination
		if folder == "" {
			folder = filepath.Join(os.Getenv("HOME"), "Desktop")
		}
	}

	// Check if the folder is writable
	// Klasörün yazılabilir olup olmadığını kontrol et
	testFile := filepath.Join(folder, "test_write_permission.tmp")
	f, err := os.Create(testFile)
	if err != nil {
		log.Printf("Selected folder is not writable: %v", err)
		return "", fmt.Errorf("selected folder is not writable: %v", err)
	}
	f.Close()
	os.Remove(testFile)

	// Save the selected folder as last destination
	// Seçilen klasörü son hedef olarak kaydet
	a.lastDestination = folder
	a.saveConfig()

	// Return the selected folder path to the frontend
	// Seçilen klasör yolunu Frontend'e gönder
	return folder, nil
}

// GetLastDestination returns the last selected destination folder
// Retrieves the last used destination folder from the app's state
// Uygulamanın durumundan son kullanılan hedef klasörü alır
func (a *App) GetLastDestination() string {
	// Return the last destination to the frontend
	// Son hedefi Frontend'e gönder
	return a.lastDestination
}

// ConvertVideo converts the input video to SVTAV1 format
// Performs the video conversion using FFmpeg and emits progress events
// FFmpeg kullanarak video dönüşümünü gerçekleştirir ve ilerleme olayları yayar
func (a *App) ConvertVideo(inputPath, outputFolder string, totalFrames int) error {
	// Prepare output file name
	// Çıktı dosya adını hazırla
	outputFileName := filepath.Base(inputPath)
	outputFileName = strings.TrimSuffix(outputFileName, filepath.Ext(outputFileName))
	outputFileName = sanitizeFileName(outputFileName)
	outputPath := filepath.Join(outputFolder, outputFileName+"_av1.mp4")

	// Create output directory if it doesn't exist
	// Çıktı dizini yoksa oluştur
	if err := os.MkdirAll(outputFolder, os.ModePerm); err != nil {
		log.Printf("Failed to create output directory: %v", err)
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Prepare log file for FFmpeg output
	// FFmpeg çıktısı için log dosyasını hazırla
	logFileName := outputFileName + "_ffmpeg.log"
	logFilePath := filepath.Join(a.appDir, "logs", logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return fmt.Errorf("failed to create log file: %v", err)
	}
	defer logFile.Close()

	// Prepare FFmpeg command
	// FFmpeg komutunu hazırla
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

	// Start FFmpeg process
	// FFmpeg işlemini başlat
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start FFmpeg: %v", err)
		return fmt.Errorf("failed to start FFmpeg: %v", err)
	}

	// Monitor progress in a separate goroutine
	// İlerlemeyi ayrı bir goroutine'de izle
	done := make(chan bool)
	go func() {
		a.monitorProgress(logFilePath, totalFrames, done)
	}()

	// Wait for FFmpeg to finish
	// FFmpeg'in bitmesini bekle
	if err := cmd.Wait(); err != nil {
		close(done)
		log.Printf("FFmpeg error: %v", err)
		runtime.EventsEmit(a.ctx, "conversion:error", err.Error())
		return fmt.Errorf("FFmpeg error: %v", err)
	}

	close(done)
	time.Sleep(time.Second) // Short wait for progress bar to reach 100% / İlerleme çubuğunun %100'e ulaşması için kısa bir bekleme
	runtime.EventsEmit(a.ctx, "conversion:complete", outputPath)
	log.Printf("Conversion completed: %s", outputPath)

	// Emit event to process next video
	// Sıradaki videoyu işlemek için olay yayınla
	runtime.EventsEmit(a.ctx, "conversion:next")

	return nil
}

// monitorProgress tracks the conversion progress and emits update events
// Monitors the FFmpeg log file and sends progress updates to the frontend
// FFmpeg Log dosyasını izler ve ilerleme güncellemelerini Frontend'e gönderir
func (a *App) monitorProgress(logPath string, totalFrames int, done chan bool) {
	// Open the log file
	// Log dosyasını aç
	file, err := os.Open(logPath)
	if err != nil {
		log.Printf("Error opening log file: %v", err)
		return
	}
	defer file.Close()

	// Prepare regular expressions for parsing
	// Ayrıştırma için düzenli ifadeleri hazırla
	frameRegex := regexp.MustCompile(`frame=\s*(\d+)`)
	speedRegex := regexp.MustCompile(`speed=(\S+)`)

	var lastProgress float64
	for {
		select {
		case <-done:
			// Conversion finished, send 100% progress
			// Dönüşüm bitti, %100  bilgisini gönder
			runtime.EventsEmit(a.ctx, "conversion:progress", map[string]interface{}{
				"progress": 100,
				"speed":    "",
			})
			return
		default:
			// Read the last 1024 bytes of the log file
			// Log dosyasının son 1024 baytını oku
			file.Seek(-1024, 2)
			scanner := bufio.NewScanner(file)
			var lastLine string
			for scanner.Scan() {
				lastLine = scanner.Text()
			}
			if err := scanner.Err(); err != nil {
				log.Printf("Error scanning file: %v", err)
				continue
			}

			// Parse progress information
			// İlerleme bilgisini ayrıştır
			if strings.Contains(lastLine, "frame=") {
				frameMatch := frameRegex.FindStringSubmatch(lastLine)
				speedMatch := speedRegex.FindStringSubmatch(lastLine)

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

					// Send progress update to frontend if progress has increased
					// İlerleme artmışsa Frontend'e ilerleme güncellemesi gönder
					if progress > lastProgress {
						lastProgress = progress
						fmt.Printf("İlerleme: %.2f%%, Hız: %s\n", progress, speed)
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

// sanitizeFileName removes or replaces invalid characters in a filename
// Ensures the output filename is valid for the file system
// Çıktı dosya adının dosya sistemi için geçerli olmasını sağlar
func sanitizeFileName(fileName string) string {
	// Remove or replace invalid characters
	// Geçersiz karakterleri kaldır veya değiştir
	fileName = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
			return '_'
		}
		return r
	}, fileName)

	// Truncate filename if it's too long
	// Dosya adı çok uzunsa kısalt
	if len(fileName) > 200 {
		fileName = fileName[:200]
	}

	return fileName
}
