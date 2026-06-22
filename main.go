
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type MediaFile struct {
	Path    string
	Date    time.Time
	IsVideo bool
}

type LightMediaFile struct {
	Path    string `json:"path"`
	Date    string `json:"date"`
	IsVideo bool   `json:"isVideo"`
}

var (
	mediaDir     string
	port         int
	indexPath    string
	thumbsDir    string
	mediaIndex   []*MediaFile
	indexMutex   sync.RWMutex
	lastScan     time.Time
	scanInterval = time.Hour
	videoExts    = map[string]bool{".mp4": true, ".avi": true, ".mov": true, ".mkv": true, ".webm": true}
	imageExts    = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".jpeg.jpg": true}
	thumbSize    = 400
	ffmpegPath   string
)

func findFFmpeg() string {
	exePath, err := os.Executable()
	if err != nil {
		return "ffmpeg"
	}
	exeDir := filepath.Dir(exePath)

	ffmpegNames := []string{"ffmpeg", "ffmpeg.exe"}

	for _, name := range ffmpegNames {
		path := filepath.Join(exeDir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	systemPaths := []string{"/usr/bin/ffmpeg", "/usr/local/bin/ffmpeg", "/bin/ffmpeg"}
	for _, path := range systemPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "ffmpeg"
}

func selectFolderWindows() string {
	psScript := `
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = "请选择包含图片和视频的文件夹"
$dialog.ShowDialog() | Out-Null
$dialog.SelectedPath
`
	cmd := exec.Command("powershell", "-Command", psScript)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func main() {
	ffmpegPath = findFFmpeg()

	flag.StringVar(&mediaDir, "dir", "", "Media directory (required)")
	flag.IntVar(&port, "port", 8080, "HTTP port (default: 8080)")
	flag.StringVar(&indexPath, "index", "media_index.json", "Index file path")
	flag.StringVar(&thumbsDir, "thumbs", "thumbs", "Thumbnail directory")
	flag.Parse()

	if mediaDir == "" {
		fmt.Println("========================================")
		fmt.Println("      🖼️  媒体浏览器 - Media Browser")
		fmt.Println("========================================")
		fmt.Println()

		if runtime.GOOS == "windows" {
			fmt.Println("💡 正在打开文件夹选择框...")
			selectedFolder := selectFolderWindows()
			if selectedFolder != "" {
				mediaDir = selectedFolder
			}
		}

		if mediaDir == "" {
			fmt.Println("请输入图片/视频文件夹路径：")
			fmt.Print("> ")

			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			mediaDir = strings.TrimSpace(input)
		}

		if mediaDir == "" {
			fmt.Println("错误：未选择文件夹，程序退出。")
			os.Exit(1)
		}
	}

	if err := os.MkdirAll(thumbsDir, 0755); err != nil {
		log.Fatal(err)
	}

	loadIndex()

	go func() {
		scanFiles()
		go generateAllThumbnails()
		for range time.Tick(scanInterval) {
			scanFiles()
			go generateAllThumbnails()
		}
	}()

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/api/dates", datesHandler)
	http.HandleFunc("/api/files/", filesHandler)
	http.HandleFunc("/api/file/", fileHandler)
	http.HandleFunc("/api/delete", deleteHandler)
	http.HandleFunc("/thumb/", thumbHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println()
	fmt.Println("✅ 启动成功！")
	fmt.Println("📁 媒体文件夹：", mediaDir)
	fmt.Println("🌐 访问地址：http://localhost:" + strconv.Itoa(port))
	fmt.Println()
	fmt.Println("💡 提示：在浏览器中打开上面的地址即可访问")
	fmt.Println("💡 按 Ctrl+C 可停止程序")
	fmt.Println()

	log.Printf("Server starting on :%d, media dir: %s", port, mediaDir)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, "static/index.html")
}

func datesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=300")
	indexMutex.RLock()
	defer indexMutex.RUnlock()

	dates := make(map[string]bool)
	for _, f := range mediaIndex {
		dates[f.Date.Format("2006-01-02")] = true
	}

	dateList := make([]string, 0, len(dates))
	for d := range dates {
		dateList = append(dateList, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dateList)))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"dates": dateList,
		"total": len(mediaIndex),
	})
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=300")
	date := strings.TrimPrefix(r.URL.Path, "/api/files/")
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		page, _ = strconv.Atoi(pageStr)
	}

	indexMutex.RLock()
	defer indexMutex.RUnlock()

	var files []LightMediaFile
	targetDate := date
	for _, f := range mediaIndex {
		if f.Date.Format("2006-01-02") == targetDate {
			files = append(files, LightMediaFile{
				Path:    f.Path,
				Date:    f.Date.Format("2006-01-02"),
				IsVideo: f.IsVideo,
			})
		}
	}

	pageSize := 100
	total := len(files)
	pages := (total + pageSize - 1) / pageSize
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		files = []LightMediaFile{}
	} else if end > total {
		files = files[start:]
	} else {
		files = files[start:end]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"files":  files,
		"total":  total,
		"pages":  pages,
		"page":   page,
		"date":   date,
	})
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/file/")
	fullPath := filepath.Join(mediaDir, filepath.FromSlash(path))

	absMedia, _ := filepath.Abs(mediaDir)
	absFull, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFull, absMedia) {
		http.NotFound(w, r)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	if videoExts[ext] {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Accept-Ranges", "bytes")
	} else {
		w.Header().Set("Cache-Control", "max-age=86400")
	}

	http.ServeFile(w, r, fullPath)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Files []string `json:"files"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	deleted := 0
	failed := 0

	for _, fileName := range req.Files {
		filePath := filepath.Join(mediaDir, filepath.FromSlash(fileName))
		safeName := strings.ReplaceAll(fileName, "/", "_")
		thumbPath := filepath.Join(thumbsDir, safeName+".thumb.jpg")

		absMedia, _ := filepath.Abs(mediaDir)
		absFull, _ := filepath.Abs(filePath)
		if !strings.HasPrefix(absFull, absMedia) {
			continue
		}

		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err == nil {
				deleted++
				os.Remove(thumbPath)
			} else {
				failed++
				log.Printf("Failed to delete %s: %v", filePath, err)
			}
		}
	}

	go scanFiles()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"deleted": deleted,
		"failed":  failed,
	})
}

func thumbHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=86400")
	path := strings.TrimPrefix(r.URL.Path, "/thumb/")
	safeName := strings.ReplaceAll(path, "/", "_")
	thumbPath := filepath.Join(thumbsDir, safeName+".thumb.jpg")
	fullPath := filepath.Join(mediaDir, filepath.FromSlash(path))

	absMedia, _ := filepath.Abs(mediaDir)
	absFull, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFull, absMedia) {
		http.NotFound(w, r)
		return
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	needRegenerate := false
	thumbInfo, err1 := os.Stat(thumbPath)
	origInfo, err2 := os.Stat(fullPath)

	if err1 != nil || thumbInfo == nil {
		needRegenerate = true
	} else if err2 == nil && origInfo != nil && origInfo.ModTime().After(thumbInfo.ModTime()) {
		needRegenerate = true
	}

	if !needRegenerate {
		http.ServeFile(w, r, thumbPath)
		return
	}

	ext := strings.ToLower(filepath.Ext(path))

	var img image.Image
	if videoExts[ext] {
		img = generateVideoThumbnail(fullPath)
	} else {
		img = generateImageThumbnail(fullPath)
	}

	if img == nil {
		http.NotFound(w, r)
		return
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	os.WriteFile(thumbPath, buf.Bytes(), 0644)
	w.Header().Set("Content-Type", "image/jpeg")
	w.Write(buf.Bytes())
}

func generateImageThumbnail(imagePath string) image.Image {
	cmd := exec.Command(ffmpegPath,
		"-i", imagePath,
		"-vframes", "1",
		"-vf", fmt.Sprintf("scale='if(gt(iw,ih),%d,-2)':'if(gt(iw,ih),-2,%d)':flags=lanczos", thumbSize, thumbSize),
		"-q:v", "2",
		"-f", "image2pipe",
		"-")
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err == nil {
		img, _, err := image.Decode(&out)
		if err == nil {
			return img
		}
	}

	file, err := os.Open(imagePath)
	if err != nil {
		log.Printf("Failed to open image: %v", err)
		return nil
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Failed to decode image: %v, path: %s", err, imagePath)
		return nil
	}

	return resizeImage(img, thumbSize)
}

func generateVideoThumbnail(videoPath string) image.Image {
	timePoints := []string{"00:00:01", "00:00:03", "00:00:05", "00:00:10"}

	for _, timePoint := range timePoints {
		cmd := exec.Command(ffmpegPath,
			"-ss", timePoint,
			"-i", videoPath,
			"-vframes", "1",
			"-vf", fmt.Sprintf("scale='if(gt(iw,ih),%d,-2)':'if(gt(iw,ih),-2,%d)':flags=lanczos", thumbSize, thumbSize),
			"-q:v", "2",
			"-f", "image2pipe",
			"-")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr

		if err := cmd.Run(); err == nil {
			img, _, err := image.Decode(&out)
			if err == nil {
				return img
			}
		}
	}

	log.Printf("Failed to generate thumbnail for video: %s", videoPath)
	return nil
}

func resizeImage(img image.Image, maxSize int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= maxSize && height <= maxSize {
		return img
	}

	var newWidth, newHeight int
	if width > height {
		newWidth = maxSize
		newHeight = (height * maxSize) / width
	} else {
		newHeight = maxSize
		newWidth = (width * maxSize) / height
	}

	result := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := (x * width) / newWidth
			srcY := (y * height) / newHeight
			result.Set(x, y, img.At(srcX, srcY))
		}
	}

	return result
}

func generateAllThumbnails() {
	indexMutex.RLock()
	files := make([]*MediaFile, len(mediaIndex))
	copy(files, mediaIndex)
	indexMutex.RUnlock()

	workerCount := 4
	jobs := make(chan string, len(files))
	results := make(chan bool, len(files))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				safeName := strings.ReplaceAll(path, "/", "_")
				thumbPath := filepath.Join(thumbsDir, safeName+".thumb.jpg")
				fullPath := filepath.Join(mediaDir, filepath.FromSlash(path))

				needGenerate := false
				thumbInfo, err1 := os.Stat(thumbPath)
				origInfo, err2 := os.Stat(fullPath)

				if err2 != nil || origInfo == nil {
					results <- false
					continue
				}

				if err1 != nil || thumbInfo == nil {
					needGenerate = true
				} else if origInfo.ModTime().After(thumbInfo.ModTime()) {
					needGenerate = true
				}

				if !needGenerate {
					results <- false
					continue
				}

				ext := strings.ToLower(filepath.Ext(path))

				var img image.Image
				if videoExts[ext] {
					img = generateVideoThumbnail(fullPath)
				} else {
					img = generateImageThumbnail(fullPath)
				}

				if img != nil {
					f, err := os.Create(thumbPath)
					if err == nil {
						jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
						f.Close()
						results <- true
					} else {
						results <- false
					}
				} else {
					results <- false
				}
			}
		}()
	}

	for _, f := range files {
		jobs <- f.Path
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	generated := 0
	skipped := 0
	for result := range results {
		if result {
			generated++
		} else {
			skipped++
		}
	}

	log.Printf("缩略图预生成完成：新生成 %d 个，跳过 %d 个", generated, skipped)
}

func scanFiles() {
	var files []*MediaFile
	err := filepath.WalkDir(mediaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !imageExts[ext] && !videoExts[ext] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(mediaDir, path)
		files = append(files, &MediaFile{
			Path:    filepath.ToSlash(relPath),
			Date:    info.ModTime(),
			IsVideo: videoExts[ext],
		})
		return nil
	})

	if err != nil {
		log.Printf("Scan error: %v", err)
		return
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Date.After(files[j].Date)
	})

	indexMutex.Lock()
	mediaIndex = files
	lastScan = time.Now()
	indexMutex.Unlock()

	saveIndex()
	log.Printf("Scanned %d files", len(files))
}

func loadIndex() {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		log.Printf("Index file not found: %v", err)
		return
	}

	var files []*MediaFile
	if err := json.Unmarshal(data, &files); err != nil {
		log.Printf("Failed to parse index: %v", err)
		return
	}

	indexMutex.Lock()
	mediaIndex = files
	indexMutex.Unlock()

	log.Printf("Loaded index with %d files", len(files))
}

func saveIndex() {
	indexMutex.RLock()
	data, err := json.Marshal(mediaIndex)
	indexMutex.RUnlock()

	if err != nil {
		log.Printf("Failed to marshal index: %v", err)
		return
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		log.Printf("Failed to save index: %v", err)
	}
}
