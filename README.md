
# AV1 Video Converter

- In this article, I will share information about how to write a converter program for the AV1 Codec, which I previously provided information about.
- Before we start creating our AV1 video converter application, let's get some information about the Wails framework we'll be using.

## What is Wails?

Wails is an open-source project that allows you to create desktop applications using Go and web technologies. Conceptually, it's similar to Electron, but there are some important differences:

1. **Go Backend**: Unlike Electron, which uses Node.js, Wails uses Go for the backend. This allows developers to take advantage of Go's performance and extensive standard library.

2. **Smaller Size**: Wails applications generally have a smaller file size compared to Electron applications and use less memory, as they don't include a full Chromium browser.

3. **Native Render**: Wails uses the operating system's native webview, which provides better performance and a more native feel.

4. **Multi-Platform Support**: Like Electron, Wails supports building applications for Windows, macOS, and Linux.

5. **Frontend Flexibility**: While our example uses Svelte, Wails is compatible with various frontend frameworks such as React, Vue, or pure JavaScript.

Wails bridges web technologies and Go, allowing developers to create powerful desktop applications with a modern and responsive user interface. It's particularly suitable for developers comfortable with Go who want to build desktop applications without diving into platform-specific GUI frameworks.

Now that we understand what Wails is, let's move on to creating our AV1 Video Converter application using this powerful framework.

# Let's Start Writing the AV1 Video Converter Application

Before we start writing the application, let's take a look at what we'll need. I'll assume you have the following requirements on your computer, and I won't go into their installation, but feel free to reach out to me if you need detailed information about them.

* Go programming language
* Node.js and npm
* Wails framework
* Svelte
* FFmpeg and FFprobe

- The compiled view of our application will look like this:

<div align="center">

  <br><img width="1030" alt="av1converter" src="https://github.com/user-attachments/assets/fac155e4-f860-4ba9-ab78-fc0dfea359ee">

  *Figure 1: Application view*
</div>

## Step 1: Creating the Project Structure
1. Assuming Go is installed on your computer, we'll start with the Wails installation. If you haven't installed the Go programming language yet, you can install it by following the installation instructions on the [Go Official Page](https://go.dev/ "Go Programming Language").
2. Open the command line and install the Wails framework with the following command:

``` SH
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

3. Let's create our Wails project:

``` SH
wails init -n AV1-video-converter -t svelte
```

4. Let's navigate to the project directory:

``` SH
cd AV1-video-converter
```

- Now our project is created in this directory, let's start writing our code

## Step 2: Writing Go Codes
I will write code inside 3 files in the project files. The main files of our application will consist of "app.go", "main.go" and "app.svelte"

### Part 1: Backend (app.go)
#### 1.1 Package and Import Definitions
 ``` GO
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
 ```

- In this section, the Go packages required by the application are imported. Important packages include os, exec, filepath, and Wails runtime.

#### 1.2 Data Structures
 ``` GO
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

 ``` 

- The VideoInfo struct represents the properties of a video file. The App struct contains the general state and configuration of the application.

#### 1.3 Application Startup and Shutdown
``` GO
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

``` 

- The startup function runs when the application is launched and makes the necessary configurations. The shutdown function performs cleanup operations when the application is closed.

#### 1.4 Finding FFmpeg and FFprobe Locations
```  GO
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

```

- Searches for the locations of FFmpeg and FFprobe installations.

#### 1.5 Log and Config Operations
```  GO
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
```

#### 1.6 File and Folder Operations
```  GO
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
// Uygulamanın configinden son kullanılan hedef klasörü alır
func (a *App) GetLastDestination() string {
	// Return the last destination to the frontend
	// Son hedefi Frontend'e gönder
	return a.lastDestination
}
``` 

- These functions allow the user to select video files and the destination folder. The GetLastDestination function retrieves the last used destination folder from the application's config.

#### 1.7 Video Conversion
``` GO
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

``` 

- The ConvertVideo function converts the selected video file to AV1 format.
- You can adjust bitrate, resolution, speed, codec, etc. by changing the content of the "cmd" variable.
- Here I converted to AV1 codec using libsvtav1, you can make it h265, h264, mpeg-2 if you wish.

#### 1.8 Progress Monitoring
``` GO
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

``` 

- This function monitors the progress of the conversion process and notifies the frontend (Svelte side).

#### 1.9 Our Main.go File
``` GO
package main

import (
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:         "MD-AV1-Converter",
		Width:         1024,
		Height:        680,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}


``` 

- You can change the name and resolution of the application from our main.go file.

## Step 2: Writing Frontend Svelte Codes (app.svelte)

#### 2.1 Script Section
``` JavaScript
<script>
  // Import necessary Svelte functions and modules
  // Gerekli Svelte fonksiyonlarını ve modüllerini içe aktar
  import { onMount } from 'svelte';
  import { flip } from 'svelte/animate';
  import { quintOut } from 'svelte/easing';
  import Icon from "./Icon.svelte";

  // Initialize state variables for the component
  // Bileşen için durum değişkenlerini başlat
  let selectedVideos = [];  // Array to store selected video information / Seçilen video bilgilerini saklamak için dizi
  let progressVideo = null;  // Currently processing video / Şu anda işlenen video
  let contextMenu = { show: false, x: 0, y: 0, index: -1 };  // Context menu state / Bağlam menüsü durumu
  let draggedOverIndex = -1;  // Index of the item being dragged over / Üzerine sürüklenen öğenin indeksi
  let destinationFolder = '';  // Selected destination folder / Seçilen hedef klasör
  let conversionProgress = 0;  // Current conversion progress / Mevcut dönüşüm ilerlemesi
  let conversionSpeed = '';  // Current conversion speed / Mevcut dönüşüm hızı
  let errorMessage = '';  // Error message to display / Görüntülenecek hata mesajı
  let showErrorPopup = false;  // Whether to show the error popup / Hata Pop'u gösterilip gösterilmeyeceği

  // Define table headers with tooltips
  // Araç ipuçları ile tablo başlıklarını tanımla
  const tableHeaders = [
    { label: "#", tooltip: "Index" },
    { label: "File Path", tooltip: "Full path of the video file" },
    { label: "Duration", tooltip: "Video duration (HH:MM:SS:FF)" },
    { label: "Frames", tooltip: "Total number of frames" },
    { label: "Codec", tooltip: "Video codec" },
    { label: "Size", tooltip: "File size" }
  ];

  // Set up event listeners and initialize data when the component mounts
  // Bileşen monte edildiğinde olay dinleyicilerini ayarla ve verileri başlat
  onMount(async () => {
    // Listen for Wails runtime loaded event
    // Wails çalışma zamanı yüklendi olayını dinle
    window.runtime.EventsOn("wails:loaded", () => {
      console.log("Wails runtime Loaded");
    });

    // Add click event listener to close context menu
    // Bağlam menüsünü kapatmak için tıklama olay dinleyicisi ekle
    document.addEventListener('click', closeContextMenu);

    // Listen for conversion progress updates from Go backend
    // Go Bakcend'den dönüşüm ilerleme güncellemelerini dinle
    window.runtime.EventsOn("conversion:progress", (data) => {
      console.log("Progress update:", data);
      conversionProgress = data.progress;
      conversionSpeed = data.speed;
    });

    // Listen for conversion completion event from Go backend
    // Go Bakcend'den dönüşüm tamamlanma olayını dinle
    window.runtime.EventsOn("conversion:complete", (outputPath) => {
      console.log("Conversion completed:", outputPath);
      progressVideo = null;
      updateProgressVideo();
    });

    // Listen for conversion error event from Go backend
    // Go Bakcend'den dönüşüm hata olayını dinle
    window.runtime.EventsOn("conversion:error", (error) => {
      console.error("Conversion error:", error);
      errorMessage = error;
      showErrorPopup = true;
      progressVideo = null;
      updateProgressVideo();
    });

    // Listen for next video conversion event from Go backend
    // Go Bakcend'den sonraki video dönüşüm olayını dinle
    window.runtime.EventsOn("conversion:next", () => {
      updateProgressVideo();
    });

    // Get the last destination folder from Go backend
    // Go Bakcend'den son hedef klasörü al
    destinationFolder = await window.go.main.App.GetLastDestination();
  });

  // Function to handle selecting video files
  // Video dosyalarını seçme işlemini yöneten fonksiyon
  async function handleSelectFiles() {
    try {
      // Call Go backend to open file dialog and get video info
      // Dosya iletişim kutusunu açmak ve video bilgilerini almak için Go Bakcend'i çağır
      const videoInfos = await window.go.main.App.SelectVideoFiles();
      if (videoInfos && videoInfos.length > 0) {
        selectedVideos = [...selectedVideos, ...videoInfos];
        updateProgressVideo();
      }
    } catch (err) {
      console.error("Selected File Error:", err);
      showError("Selected File Error: " + err.message);
    }
  }

  // Function to handle selecting destination folder
  // Hedef klasör seçme işlemini yöneten fonksiyon
  async function handleSelectDestination() {
    try {
      // Call Go backend to open folder dialog
      // Klasör iletişim kutusunu açmak için Go Bakcend'i çağır
      const folder = await window.go.main.App.SelectDestinationFolder();
      if (folder) {
        destinationFolder = folder;
      }
    } catch (err) {
      console.error("Destination folder selection error:", err);
      showError("Destination folder selection error: " + err.message);
    }
  }

  // Function to update the current video being processed
  // İşlenen mevcut videoyu güncelleyen fonksiyon
  function updateProgressVideo() {
    if (!progressVideo && selectedVideos.length > 0) {
      progressVideo = selectedVideos.shift();
      selectedVideos = [...selectedVideos];
      startConversion();
    }
  }

  // Function to start the video conversion process
  // Video dönüşüm sürecini başlatan fonksiyon
  async function startConversion() {
    if (progressVideo && destinationFolder) {
      conversionProgress = 0;
      conversionSpeed = '';
      try {
        // Call Go backend to start video conversion
        // Video dönüşümünü başlatmak için Go Bakcend'i çağır
        await window.go.main.App.ConvertVideo(progressVideo.fullPath, destinationFolder, progressVideo.frameCount);
      } catch (err) {
        console.error("Conversion Error:", err);
        showError("Conversion Error: " + err.message);
        progressVideo = null;
        updateProgressVideo();
      }
    }
  }

  // Function to handle drag start event
  // Sürükleme başlangıç olayını yöneten fonksiyon
  function dragStart(event, index) {
    event.dataTransfer.setData('text/plain', index);
    event.target.classList.add('dragging');
  }

  // Function to handle drag end event
  // Sürükleme bitiş olayını yöneten fonksiyon
  function dragEnd(event) {
    event.target.classList.remove('dragging');
    draggedOverIndex = -1;
  }

  // Function to handle drag over event
  // Sürükleme üzerinde olayını yöneten fonksiyon
  function dragOver(event, index) {
    event.preventDefault();
    draggedOverIndex = index;
  }

  // Function to handle drag leave event
  // Sürüklemeden ayrılma olayını yöneten fonksiyon
  function dragLeave() {
    draggedOverIndex = -1;
  }

  // Function to handle drop event
  // Bırakma olayını yöneten fonksiyon
  function drop(event, targetIndex) {
    event.preventDefault();
    const sourceIndex = parseInt(event.dataTransfer.getData('text/plain'));
    if (sourceIndex !== targetIndex) {
      const items = Array.from(selectedVideos);
      const [reorderedItem] = items.splice(sourceIndex, 1);
      items.splice(targetIndex, 0, reorderedItem);
      selectedVideos = items;
    }
    draggedOverIndex = -1;
  }

  // Function to show context menu
  // Bağlam menüsünü gösteren fonksiyon
  function showContextMenu(event, index) {
    event.preventDefault();
    contextMenu = {
      show: true,
      x: event.clientX,
      y: event.clientY,
      index: index
    };
  }

  // Function to close context menu
  // Bağlam menüsünü kapatan fonksiyon
  function closeContextMenu() {
    contextMenu.show = false;
  }

  // Function to delete an item from the video list
  // Video listesinden bir öğeyi silen fonksiyon
  function deleteItem() {
    if (contextMenu.index > -1) {
      selectedVideos = selectedVideos.filter((_, index) => index !== contextMenu.index);
      closeContextMenu();
    }
  }

  // Function to show error message
  // Hata mesajını gösteren fonksiyon
  function showError(message) {
    errorMessage = message;
    showErrorPopup = true;
  }

  // Function to close error popup
  // Hata açılır penceresini kapatan fonksiyon
  function closeErrorPopup() {
    showErrorPopup = false;
    errorMessage = '';
  }
</script>

``` 

- In the script section, we do the following operations;

- import statements and variable definitions
- lifecycle functions
- file and folder selection functions
- conversion and progress monitoring functions
- user interaction functions

#### 2.2 HTML Section
```  HTML
<main on:contextmenu|preventDefault>
  <h1>AV1 Video Converter</h1>

  <!-- Destination folder selector -->
  <!-- Hedef klasör seçici -->
  <div class="destination-selector">
    <button on:click={handleSelectDestination}>Select Destination</button>
    <input type="text" bind:value={destinationFolder} readonly placeholder="No destination selected">
  </div>

  <!-- Progress display for current video conversion -->
  <!-- Mevcut video dönüşümü için ilerleme göstergesi -->
  <div class="progress-container">
    <h2>Progress</h2>
    {#if progressVideo}
      <table>
        <thead>
        <tr>
          {#each tableHeaders as header}
            <th title={header.tooltip}>{header.label}</th>
          {/each}
        </tr>
        </thead>
        <tbody>
        <tr>
          <td>-</td>
          <td>{progressVideo.fullPath}</td>
          <td>{progressVideo.duration}</td>
          <td>{progressVideo.frameCount}</td>
          <td>{progressVideo.codec}</td>
          <td>{progressVideo.size}</td>
        </tr>
        </tbody>
      </table>
      <div class="conversion-progress">
        <progress value={conversionProgress} max="100"></progress>
        <span>{conversionProgress.toFixed(2)}%</span>
      </div>
      <div class="conversion-speed">
        <span>Speed: {conversionSpeed}</span>
      </div>
    {:else}
      <p>No video in progress</p>
    {/if}
  </div>

  <!-- Button to add new videos -->
  <!-- Yeni videolar eklemek için düğme -->
  <button class="add-video-btn" on:click={handleSelectFiles}>
    <i class="fas fa-plus"></i>
    <i class="fas fa-video"></i>
    Add Video(s)
  </button>

  <!-- Table displaying selected videos -->
  <!-- Seçilen videoları gösteren tablo -->
  <div class="table-container">
    <table>
      <thead>
      <tr>
        {#each tableHeaders as header}
          <th title={header.tooltip}>{header.label}</th>
        {/each}
      </tr>
      </thead>
      <tbody>
      {#each selectedVideos as video, index (video.fullPath)}
        <tr
                animate:flip={{duration: 300, easing: quintOut}}
                draggable={true}
                on:dragstart={(event) => dragStart(event, index)}
                on:dragend={dragEnd}
                on:dragover={(event) => dragOver(event, index)}
                on:dragleave={dragLeave}
                on:drop={(event) => drop(event, index)}
                on:contextmenu={(event) => showContextMenu(event, index)}
                class:drag-over={draggedOverIndex === index}
        >
          <td>{index + 1}</td>
          <td>{video.fullPath}</td>
          <td>{video.duration}</td>
          <td>{video.frameCount}</td>
          <td>{video.codec}</td>
          <td>{video.size}</td>
        </tr>
      {/each}
      </tbody>
    </table>
  </div>

  <!-- Instructions for users -->
  <!-- Kullanıcılar için talimatlar -->
  <div class="instructions">
    <p>Right-click on a video to remove it from the list | Drag and drop to reorder videos</p>
    <p>Duration format: Hours:Minutes:Seconds:Frames (HH:MM:SS:FF)</p>
    <p>For customization or advanced features, please contact: <a href="mailto:murat@muratdemirci.com.tr">murat@muratdemirci.com.tr</a></p>
  </div>

  <!-- Context menu for video list items -->
  <!-- Video listesi öğeleri için bağlam menüsü -->
  {#if contextMenu.show}
    <div class="context-menu" style="top: {contextMenu.y}px; left: {contextMenu.x}px;">
      <button on:click={deleteItem}>Delete</button>
    </div>
  {/if}

  <!-- Error popup -->
  <!-- Hata açılır penceresi -->
  {#if showErrorPopup}
    <div class="error-popup">
      <div class="error-content">
        <h3>Error</h3>
        <p>{errorMessage}</p>
        <button on:click={closeErrorPopup}>Close</button>
      </div>
    </div>
  {/if}
</main>

``` 

- In this section, html definitions related to the user interface are made, the following main headings
- Destination folder selector
- Progress indicator
- Video file add button
- Video list table
- Context menu
- Error popup

#### 2.3 CSS Section
``` css
<style>
  @font-face {
    font-family: 'Roboto';
    src: url('assets/fonts/Roboto-Regular.woff2') format('woff2'),
    url('assets/fonts/Roboto-Regular.woff') format('woff');
    font-weight: 400;
    font-style: normal;
  }

  @font-face {
    font-family: 'Roboto';
    src: url('assets/fonts/Roboto-Bold.woff2') format('woff2'),
    url('assets/fonts/Roboto-Bold.woff') format('woff');
    font-weight: 700;
    font-style: normal;
  }

  :root {
    --primary-color: #4a90e2;
    --secondary-color: #357abD;
    --background-color: #343434;
    --text-color: #e0e0e0;
    --hover-color: #4a4a4a;
    --table-bg-color: #2a2a2a;
    --table-border-color: #4a4a4a;
  }

  main {
    font-family: 'Roboto', sans-serif;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background-color: var(--background-color);
    color: var(--text-color);
    font-size: 14px;
  }

  h1, h2 {
    color: var(--primary-color);
    text-align: center;
    font-weight: 500;
  }

  h1 {
    font-size: 24px;
  }

  h2 {
    font-size: 20px;
  }

  button {
    font-family: 'Roboto', sans-serif;
    background-color: var(--primary-color);
    border: none;
    color: white;
    padding: 8px 16px;
    text-align: center;
    text-decoration: none;
    display: inline-block;
    font-size: 14px;
    margin: 10px 0;
    cursor: pointer;
    border-radius: 4px;
    transition: background-color 0.3s;
    font-weight: 500;
  }

  button:hover {
    background-color: var(--secondary-color);
  }

  .add-video-btn {
    padding: 8px 16px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 10px auto;
    width: auto;
    height: auto;
    border-radius: 4px;
  }

  .add-video-btn i {
    margin-right: 8px;
    font-size: 16px; /* İkon boyutunu artır */
  }

  .destination-selector {
    display: flex;
    align-items: center;
    margin-bottom: 20px;
    gap: 10px;
  }

  .destination-selector button {
    flex-shrink: 0;
  }

  .destination-selector input {
    flex-grow: 1;
    padding: 8px;
    border-radius: 4px;
    border: 1px solid var(--table-border-color);
    background-color: var(--table-bg-color);
    color: var(--text-color);
    font-family: 'Roboto', sans-serif;
    font-size: 14px;
  }

  .progress-container, .table-container {
    background-color: var(--table-bg-color);
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
    overflow: hidden;
    margin-bottom: 20px;
  }

  .progress-container table, .table-container table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
  }

  .progress-container th, .progress-container td,
  .table-container th, .table-container td {
    padding: 8px 12px;
    text-align: left;
    border-bottom: 1px solid var(--table-border-color);
    font-size: 12px;
  }

  .progress-container p {
    text-align: center;
    padding: 16px;
    color: var(--text-color);
    font-size: 14px;
  }

  .table-container {
    flex: 1;
    overflow-y: auto;
    max-height: calc(100vh - 500px); /* Açıklama için daha fazla alan bırak */
  }

  thead {
    position: sticky;
    top: 0;
    background-color: var(--primary-color);
    color: white;
    z-index: 1;
  }

  th {
    font-weight: 500;
    text-transform: uppercase;
    font-size: 12px;
    letter-spacing: 0.5px;
  }

  tbody tr {
    background-color: var(--table-bg-color);
    transition: background-color 0.2s;
  }

  tbody tr:hover {
    background-color: var(--hover-color);
    cursor: move;
  }

  tbody tr.dragging {
    opacity: 0.5;
    background-color: var(--hover-color);
  }

  td:first-child, th:first-child {
    border-left: none;
    width: 30px;
    text-align: center;
  }

  .context-menu {
    position: fixed;
    background-color: var(--table-bg-color);
    border-radius: 4px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
    z-index: 1000;
    overflow: hidden;
  }

  .context-menu button {
    display: block;
    width: 100%;
    padding: 8px 12px;
    background-color: var(--table-bg-color);
    border: none;
    text-align: left;
    cursor: pointer;
    font-size: 12px;
    color: var(--text-color);
    transition: background-color 0.2s;
    margin: 0;
  }

  .context-menu button:hover {
    background-color: var(--hover-color);
  }

  tbody tr.drag-over {
    background-color: rgba(74, 144, 226, 0.2);
    box-shadow: 0 0 10px rgba(74, 144, 226, 0.5);
  }

  tbody tr.drag-over td {
    border-top: 2px solid var(--primary-color);
    border-bottom: 2px solid var(--primary-color);
  }

  tbody tr.drag-over td:first-child {
    border-left: 2px solid var(--primary-color);
  }

  tbody tr.drag-over td:last-child {
    border-right: 2px solid var(--primary-color);
  }

  .instructions {
    margin-top: 16px;
    font-size: 12px;
    color: var(--text-color);
    text-align: center;
    background-color: var(--table-bg-color);
    padding: 10px;
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
  }

  .instructions p {
    margin: 4px 0;
  }

  .instructions a[href^="mailto:"] {
    color: var(--primary-color);
    text-decoration: none;
    font-weight: bold;
    transition: color 0.3s ease, text-decoration 0.3s ease;
  }

  .instructions a[href^="mailto:"]:hover {
    color: var(--secondary-color);
    text-decoration: underline;
  }

  .instructions a[href^="mailto:"]:before {
    content: "\f0e0"; /* Font Awesome envelope icon */
    font-family: "Font Awesome 5 Free";
    font-weight: 900;
    margin-right: 5px;
  }

  .conversion-progress {
    display: flex;
    align-items: center;
    margin-top: 10px;
  }

  .conversion-progress progress {
    flex-grow: 1;
    height: 20px;
  }

  .conversion-progress span {
    margin-left: 10px;
    font-size: 14px;
  }

  .conversion-speed {
    margin-top: 5px;
    font-size: 14px;
  }

  progress {
    -webkit-appearance: none;
    appearance: none;
    width: 100%;
    height: 20px;
  }

  progress::-webkit-progress-bar {
    background-color: var(--table-border-color);
    border-radius: 4px;
  }

  progress::-webkit-progress-value {
    background-color: var(--primary-color);
    border-radius: 4px;
  }

  progress::-moz-progress-bar {
    background-color: var(--primary-color);
    border-radius: 4px;
  }

  .error-popup {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .error-content {
    background-color: var(--table-bg-color);
    padding: 20px;
    border-radius: 8px;
    max-width: 400px;
    width: 100%;
  }

  .error-content h3 {
    color: #ff4444;
    margin-top: 0;
  }

  .error-content p {
    margin-bottom: 20px;
  }
</style>
<svelte:head>
  <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500&display=swap" rel="stylesheet">
  <link href="src/assets/css/all.min.css" rel="stylesheet">
</svelte:head>
``` 

- The style section customizes the appearance of the application.

## Step 3: Compiling and Running the Application

- To run the application in developer mode and make the final appearance and adjustments before compiling, type the following command

``` sh
wails dev
``` 

- The application is now ready and if it's as you want, you can compile and start using it.
``` sh
wails build
``` 

## Step 4: Installation Recommendations for the Application
- If you are using macOS operating system, you will need ffmpeg installation, and I recommend you to install it with the following command

``` sh
brew install ffmpeg
``` 

- If you are using Linux Ubuntu, you will first need to install local webview and then ffmpeg installation. Let's make the installations by typing the following commands

``` sh
sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev
``` 
``` sh
sudo apt install ffmpeg
``` 

- If you are using Windows, all you need to do is download ffmpeg.exe and ffprobe.exe and copy them into the "bin" folder where your application is running, or it will be sufficient to define ffmpeg globally in environment variables. For ffmpeg installation files [FFmmpeg Official Page](https://ffmpeg.org/ "Ffmepg installations")

# Conclusion
- With this application, we have written an application that converts any video file to AV1 (libsvvtav1) format using FFmpeg
- While writing the application, we gained knowledge about the Wails framework
- You can access the complete source code of the project from this link [Github Av1-video-eoncoder](https://github.com/mos1907/AV1-video-converter "Github Av1-video-eoncoder")
