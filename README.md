# 🖼️ 媒体浏览器 / Media Browser

一个轻量级的局域网照片和视频浏览器。选择照片文件夹，直接开用。无需数据库、无需配置。

## 快速上手

```bash
git clone https://github.com/xiaoluhh/media-browser.git
cd media-browser
go build -o media-browser .
./media-browser -dir /path/to/你的照片和视频
```

**Windows 用户**：直接双击 `media-browser.exe`，会自动弹出文件夹选择窗口。

浏览器打开 `http://localhost:8080` 即可浏览。

## 局域网共享

在同一局域网的其他设备上访问 `http://你的IP:8080` 即可。手机、平板、电脑都能用，支持 PWA 安装到主屏幕。

## 功能

- 📷 按日期自动分组
- 🎬 视频播放（需 FFmpeg）
- 🔍 大图查看器，双指缩放 + 拖拽平移
- 👆 触摸滑动翻页
- 🗑️ 多选批量删除
- 📱 PWA，可安装到手机桌面
- 🖥️ 响应式，手机/平板/电脑都适配

## 参数

| 参数 | 默认 | 说明 |
|------|------|------|
| `-dir` | 必填 | 媒体文件夹（不填则弹出选择窗口） |
| `-port` | 8080 | 端口 |
| `-thumbs` | thumbs | 缩略图缓存目录 |

## 部署到树莓派 / Linux 服务器

```bash
git clone https://github.com/xiaoluhh/media-browser.git
cd media-browser
go build -o media-browser .
./media-browser -dir /path/to/media
```

建议用 systemd 管理服务，开机自启。

## 要求

- Go 1.21+（仅构建时需要）
- FFmpeg（可选，视频缩略图用）
- Windows / Linux / macOS

## 协议

MIT