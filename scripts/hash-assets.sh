#!/bin/sh
# hash-assets.sh - 为静态资源添加内容哈希（cache busting）
# 用法: sh hash-assets.sh [static_dir]
# 在 Docker 构建时运行，为 CSS/JS 文件名添加哈希后缀
set -e

STATIC_DIR="${1:-static}"

# 需要处理的文件列表（按依赖顺序）
FILES="app.js style.css background-patterns.css login.js login.css i18n.js"

echo "=== 开始处理静态资源哈希 ==="
echo "目录: $STATIC_DIR"

for file in $FILES; do
    filepath="$STATIC_DIR/$file"
    
    if [ ! -f "$filepath" ]; then
        echo "⚠ 跳过: $file (文件不存在)"
        continue
    fi
    
    # 计算 MD5 哈希（取前 8 位）
    hash=$(md5sum "$filepath" | cut -c1-8)
    
    # 构建新文件名: name.hash.ext
    base="${file%.*}"
    ext="${file##*.}"
    new_file="${base}.${hash}.${ext}"
    
    # 重命名文件（而非复制，避免冗余）
    mv "$filepath" "$STATIC_DIR/$new_file"
    
    # 更新所有 HTML 文件中的引用（处理双引号和单引号）
    for html in "$STATIC_DIR"/*.html; do
        if [ -f "$html" ]; then
            sed -i "s|/static/${file}\"|/static/${new_file}\"|g" "$html"
            sed -i "s|/static/${file}'|/static/${new_file}'|g" "$html"
        fi
    done
    
    echo "✓ $file → $new_file"
done

# 为 i18n 生成版本清单（保留原文件名，通过 ?v= 做缓存失效）
I18N_DIR="$STATIC_DIR/i18n"
MANIFEST_FILE="$I18N_DIR/manifest.json"
if [ -d "$I18N_DIR" ]; then
    tmp_manifest=$(mktemp)
    printf '{\n' > "$tmp_manifest"
    first=1

    for locale_file in "$I18N_DIR"/*.json; do
        [ -f "$locale_file" ] || continue

        locale_name=$(basename "$locale_file")
        if [ "$locale_name" = "manifest.json" ]; then
            continue
        fi

        locale="${locale_name%.json}"
        locale_hash=$(md5sum "$locale_file" | cut -c1-8)
        locale_url="/static/i18n/${locale}.json?v=${locale_hash}"

        if [ "$first" -eq 0 ]; then
            printf ',\n' >> "$tmp_manifest"
        fi
        first=0
        printf '  "%s": "%s"' "$locale" "$locale_url" >> "$tmp_manifest"
    done

    printf '\n}\n' >> "$tmp_manifest"
    mv "$tmp_manifest" "$MANIFEST_FILE"
    echo "✓ i18n manifest → $MANIFEST_FILE"
fi

echo "=== 处理完成 ==="
