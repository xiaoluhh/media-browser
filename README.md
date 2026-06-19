# 🖼️ 媒体浏览器 / Media Browser

> 一个轻量级的局域网照片和视频浏览器，使用 Go 编写，无需数据库，开箱即用。

## ⚡ 快速开始

### 方式一：直接下载二进制

从 [Releases](https://github.com/YOUR_USERNAME/media-browser/releases) 下载对应系统的文件，解压后运行。

### 方式二：从源码构建

`ash
# 需要 Go 1.21+
git clone https://github.com/YOUR_USERNAME/media-browser.git
cd media-browser

# 编译
go build -o media-browser .

# 运行（Windows 会弹出文件夹选择窗口）
./media-browser -dir /path/to/你的照片和视频文件夹

# 或者指定端口（默认 8080）
./media-browser -dir /path/to/media -port 3000
`

**Windows 用户**：双击运行，或直接运行 media-browser.exe，会自动弹出文件夹选择窗口。

### 访问

打开浏览器访问 http://localhost:8080（或你指定的端口）

如果部署在局域网服务器上，同一网络下的设备通过 http://服务器IP:8080 访问。

## ✨ 功能

- 📷 按日期自动分组浏览照片和视频
- 🎬 支持视频播放（需安装 FFmpeg）
- 🔍 大图查看器，支持双指缩放、拖拽平移
- 👆 触摸滑动翻页
- 🗑️ 多选批量删除
- 📱 PWA 支持，可安装到手机/桌面主屏幕
- 🖥️ 响应式设计，适配手机/平板/电脑

## 🛠️ 命令行参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| -dir | 空（必填） | 媒体文件夹路径，不填则弹出选择窗口 |
| -port | 8080 | HTTP 服务端口 |
| -index | media_index.json | 媒体索引文件路径 |
| -thumbs | 	humbs | 缩略图缓存目录 |

## 🔧 系统要求

- **Go 1.21+**（仅构建时需要）
- **FFmpeg**（可选，用于视频缩略图生成）
- 支持 Windows / Linux / macOS

## 🏠 局域网部署（树莓派 / Linux 服务器）

`ash
# 在服务器上
git clone https://github.com/YOUR_USERNAME/media-browser.git
cd media-browser
go build -o media-browser .
./media-browser -dir /path/to/media
`

然后通过 http://服务器IP:8080 访问。

建议使用 systemd 管理服务（参考 media-browser.service）。

## 📄 协议

MIT
