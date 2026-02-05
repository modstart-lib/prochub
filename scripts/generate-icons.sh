#!/bin/bash

# 图标生成脚本
# 将 SVG 图标转换为 Wails 项目所需的各种格式

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 源 SVG 文件路径
SVG_SOURCE="${PROJECT_ROOT}/res/logo.svg"

# 输出路径
BUILD_DIR="${PROJECT_ROOT}/build"
APPICON_PNG="${BUILD_DIR}/appicon.png"
ICONSET_DIR="${BUILD_DIR}/appicon.iconset"
ICNS_OUTPUT="${BUILD_DIR}/iconfile.icns"
ICNS_DARWIN_DIR="${BUILD_DIR}/darwin/iconfile.icns"
ICNS_OUTPUT_BUNDLE="${BUILD_DIR}/bin/ProcHub.app/Contents/Resources/iconfile.icns"

# Windows 图标路径
WINDOWS_ICO="${BUILD_DIR}/windows/icon.ico"

# Tray 图标路径
# macOS: 黑白模板图标 (PNG)
# Windows: 彩色图标 (ICO)
TRAY_ICON="${BUILD_DIR}/trayicon.png"
TRAY_ICON_WINDOWS="${BUILD_DIR}/trayicon_windows.ico"

# 检查源文件是否存在
if [ ! -f "$SVG_SOURCE" ]; then
    echo "❌ 错误: 未找到源 SVG 文件: $SVG_SOURCE"
    exit 1
fi

# 检查 magick 命令是否可用
if ! command -v magick &> /dev/null; then
    echo "❌ 错误: 需要安装 ImageMagick"
    echo "   请运行: brew install imagemagick"
    exit 1
fi

echo "🎨 开始生成图标..."
echo "   源文件: $SVG_SOURCE"

# 生成 Wails 主图标 (1024x1024 PNG)
echo "📦 生成 appicon.png (1024x1024)..."
# 生成 72% 大小的图标并居中（macOS 标准安全区域约为 80%，稍微更小一点确保视觉平衡）
# 注意：macOS Big Sur+ 的图标规范通常主体在 824px 左右，这里使用约 740px 以确保四周有明显留白
magick -background none -density 300 "$SVG_SOURCE" -resize 824x824 -gravity center -extent 1024x1024 "$APPICON_PNG"

# 生成 macOS iconset
echo "🍎 生成 macOS iconset..."
mkdir -p "$ICONSET_DIR"

# 定义所有需要的尺寸
declare -a SIZES=(
    "16:icon_16x16.png"
    "32:icon_16x16@2x.png"
    "32:icon_32x32.png"
    "64:icon_32x32@2x.png"
    "128:icon_128x128.png"
    "256:icon_128x128@2x.png"
    "256:icon_256x256.png"
    "512:icon_256x256@2x.png"
    "512:icon_512x512.png"
    "1024:icon_512x512@2x.png"
)

for item in "${SIZES[@]}"; do
    size="${item%%:*}"
    filename="${item#*:}"
    
    # Calculate inner size (72%) for padding
    inner_size=$(( size * 72 / 100 ))
    
    magick -background none -density 300 "$SVG_SOURCE" -resize "${inner_size}x${inner_size}" -gravity center -extent "${size}x${size}" "${ICONSET_DIR}/${filename}"
done

# 生成 icns 文件到 build 根目录 (Wails 会从这里读取)
echo "🔧 生成 iconfile.icns..."
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_OUTPUT"

# 同时复制到 darwin 目录备份
echo "🔧 复制到 darwin 目录..."
mkdir -p "$(dirname "$ICNS_DARWIN_DIR")"
cp "$ICNS_OUTPUT" "$ICNS_DARWIN_DIR"

# 如果 app bundle 已存在，也更新那里的图标
if [ -d "$(dirname "$ICNS_OUTPUT_BUNDLE")" ]; then
    echo "🔧 更新 app bundle 中的图标..."
    cp "$ICNS_OUTPUT" "$ICNS_OUTPUT_BUNDLE"
fi

# 生成 Windows ICO 文件
# Windows 图标需要包含多个尺寸：16, 24, 32, 48, 64, 128, 256
echo "🪟 生成 Windows icon.ico..."
mkdir -p "$(dirname "$WINDOWS_ICO")"

# 创建临时目录存放各尺寸的 PNG
WIN_ICONSET_DIR="${BUILD_DIR}/windows_iconset_temp"
mkdir -p "$WIN_ICONSET_DIR"

# Windows 图标标准尺寸
declare -a WIN_SIZES=(16 24 32 48 64 128 256)

for size in "${WIN_SIZES[@]}"; do
    # Calculate inner size (72%) for padding, same as macOS
    inner_size=$(( size * 72 / 100 ))
    magick -background none -density 300 "$SVG_SOURCE" -resize "${inner_size}x${inner_size}" -gravity center -extent "${size}x${size}" "${WIN_ICONSET_DIR}/icon_${size}.png"
done

# 使用 ImageMagick 将多个 PNG 合并为 ICO 文件
magick "${WIN_ICONSET_DIR}/icon_16.png" "${WIN_ICONSET_DIR}/icon_24.png" "${WIN_ICONSET_DIR}/icon_32.png" "${WIN_ICONSET_DIR}/icon_48.png" "${WIN_ICONSET_DIR}/icon_64.png" "${WIN_ICONSET_DIR}/icon_128.png" "${WIN_ICONSET_DIR}/icon_256.png" "$WINDOWS_ICO"

# 清理临时目录
rm -rf "$WIN_ICONSET_DIR"

echo "   - icon.ico: $WINDOWS_ICO"

# 生成 macOS Tray 图标（黑白模板图像）
# macOS 菜单栏图标标准：
# - 使用黑色作为前景色，透明背景
# - 推荐尺寸：22x22 像素（@1x），44x44 像素（@2x）
# - 这里生成 44x44 以支持 Retina 显示屏
echo "🖤 生成 macOS Tray 图标（黑白模板）..."

# 创建一个临时的纯图标 SVG（去掉背景矩形）
TRAY_SVG_TEMP="${BUILD_DIR}/trayicon_temp.svg"
cat > "$TRAY_SVG_TEMP" << 'EOF'
<?xml version="1.0" standalone="no"?>
<svg viewBox="0 0 1024 1024" version="1.1" xmlns="http://www.w3.org/2000/svg" width="1024" height="1024">
    <g transform="translate(128, 128) scale(0.75)">
        <path d="M128 128h768v768H128z" fill="#000000"></path>
        <path d="M256 256h512v512H256V256zM0 256h128v96H0V256z m0 192h128v96H0v-96z m0 192h128v96H0v-96zM896 256h128v96h-128V256z m0 192h128v96h-128v-96z m0 192h128v96h-128v-96zM768 0v128h-96V0h96z m-192 0v128h-96V0h96z m-192 0v128H288V0h96z m352 896v128h-96v-128h96z m-192 0v128h-96v-128h96z m-192 0v128H256v-128h96z"
              fill="#000000"></path>
    </g>
</svg>
EOF

# 将临时 SVG 转换为 PNG（黑色图标 + 透明背景）
magick -background none -density 300 "$TRAY_SVG_TEMP" -resize 44x44 "$TRAY_ICON"

# 清理临时 SVG
rm -f "$TRAY_SVG_TEMP"

echo "   - trayicon.png: $TRAY_ICON"

# 生成 Windows Tray 图标（彩色 ICO 格式）
# Windows 托盘图标标准：
# - 彩色图标，支持透明背景
# - 推荐尺寸：16x16, 24x24, 32x32, 48x48
echo "🪟 生成 Windows Tray 图标（彩色 ICO）..."

# 创建临时目录存放各尺寸的 PNG
WIN_TRAY_ICONSET_DIR="${BUILD_DIR}/windows_tray_iconset_temp"
mkdir -p "$WIN_TRAY_ICONSET_DIR"

# Windows 托盘图标标准尺寸
declare -a WIN_TRAY_SIZES=(16 24 32 48)

for size in "${WIN_TRAY_SIZES[@]}"; do
    # 托盘图标不需要太多留白，使用 85% 的尺寸
    inner_size=$(( size * 85 / 100 ))
    magick -background none -density 300 "$SVG_SOURCE" -resize "${inner_size}x${inner_size}" -gravity center -extent "${size}x${size}" "${WIN_TRAY_ICONSET_DIR}/icon_${size}.png"
done

# 使用 ImageMagick 将多个 PNG 合并为 ICO 文件
magick "${WIN_TRAY_ICONSET_DIR}/icon_16.png" "${WIN_TRAY_ICONSET_DIR}/icon_24.png" "${WIN_TRAY_ICONSET_DIR}/icon_32.png" "${WIN_TRAY_ICONSET_DIR}/icon_48.png" "$TRAY_ICON_WINDOWS"

# 清理临时目录
rm -rf "$WIN_TRAY_ICONSET_DIR"

echo "   - trayicon_windows.ico: $TRAY_ICON_WINDOWS"

# 清理临时文件
echo "🧹 清理临时文件..."
rm -rf "$ICONSET_DIR"

echo "   - darwin backup: $ICNS_DARWIN_DIR"
echo ""
echo "✅ 图标生成完成!"
echo "   - appicon.png: $APPICON_PNG"
echo "   - iconfile.icns: $ICNS_OUTPUT"
echo "   - icon.ico: $WINDOWS_ICO"
echo "   - trayicon.png: $TRAY_ICON"
echo "   - trayicon_windows.ico: $TRAY_ICON_WINDOWS"
if [ -f "$ICNS_OUTPUT_BUNDLE" ]; then
    echo "   - app bundle icns: $ICNS_OUTPUT_BUNDLE"
    echo ""
    echo "🚨 注意：如果图标没有变化，可能是 macOS 图标缓存导致的。"
    echo "   请尝试运行以下命令刷新缓存："
    echo "   rm -rf /var/folders/*/*/*/com.apple.dock.iconcache; killall Dock"
    echo "   或者手动将应用移出 Applications 文件夹再移回去。"
fi
