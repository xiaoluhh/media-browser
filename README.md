# 媒体浏览器

一个简单的媒体浏览应用，支持在本地网络上浏览照片和视频。

## 功能特性

- 按日期分组显示媒体文件
- 照片和视频分开查看
- 支持视频缩略图（需要 FFmpeg）
- 响应式设计，适配手机和桌面
- 支持滑动、键盘和鼠标滚轮操作
- 自动扫描新增文件

## 快速开始

### 树莓派 / Linux

```bash
./media-browser -dir /media/lat/disk2/tg -port 8080
```

或使用启动脚本：

```bash
./start.sh /media/lat/disk2/tg
```

### Windows

双击 `启动媒体浏览器.bat` 文件，选择照片文件夹即可。

或在命令行运行：

```cmd
media-browser-windows-x64.exe -dir "D:\Photos" -port 8080
```

然后在浏览器访问 http://localhost:8080

## 命令行参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| -dir | 媒体文件夹路径（必须） | - |
| -port | HTTP 端口 | 8080 |
| -index | 索引文件路径 | media_index.json |
| -thumbs | 缩略图缓存目录 | thumbs |

## 编译说明

### 编译 Linux ARM (树莓派)

```bash
GOOS=linux GOARCH=arm go build -o media-browser-linux-arm
```

### 编译 Linux x64

```bash
GOOS=linux GOARCH=amd64 go build -o media-browser-linux-x64
```

### 编译 Windows

```bash
GOOS=windows GOARCH=amd64 go build -o media-browser-windows-x64.exe
```

## 依赖

### 视频缩略图

需要安装 FFmpeg：

- Linux: `sudo apt install ffmpeg`
- Windows: 下载 ffmpeg.exe 放到程序目录

照片功能不需要 FFmpeg。

## 注意事项

- 媒体文件按修改时间排序
- 每小时自动扫描新文件
- 缩略图缓存在 thumbs 文件夹
- 索引保存在 media_index.json
