#!/bin/bash
# check-markdown-similarity.sh
#
# Markdownファイル間の類似度を検出するCIスクリプト

set -euo pipefail

# 設定
SIMILARITY_THRESHOLD=80  # 類似度閾値（%）
MAX_DUPLICATE_LINES=30   # 許容重複行数
IGNORE_FILE=".markdown-similarity-ignore"

# テキスト正規化関数
normalize_text() {
    local file="$1"
    cat "$file" | \
        # コードブロックを除外
        sed '/```/,/```/d' | \
        # リンクを除外
        sed 's/\[.*\](.*)//g' | \
        # 空行を除外
        sed '/^$/d' | \
        # 連続する空白を正規化
        sed 's/[ \t]\+/ /g' | \
        # 行頭空白を除外
        sed 's/^[ \t]*//' | \
        # 先頭のタイトル行を除外（最初の10行はチェック対象外）
        sed '1,10d'
}

# 類似度計算関数
calculate_similarity() {
    local file1="$1"
    local file2="$2"

    local text1=$(normalize_text "$file1")
    local text2=$(normalize_text "$file2")

    local lines1_count=$(echo "$text1" | wc -l | awk '{print $1}')
    local lines2_count=$(echo "$text2" | wc -l | awk '{print $1}')

    # 連続する同一行を検出
    local duplicate_lines=$(diff -y <(echo "$text1") <(echo "$text2") | \
        grep -E '^\S+\s+\t\s+\S+$' | wc -l | awk '{print $1}')

    if [ "$lines1_count" -eq 0 ] || [ "$lines2_count" -eq 0 ]; then
        echo "0"
        return
    fi

    # 小さい方の行数を基準に類似度を計算
    local base_count=$((lines1_count < lines2_count ? lines1_count : lines2_count))
    local similarity=$((duplicate_lines * 100 / base_count))
    echo "$similarity"
}

# メイン処理
main() {
    echo "# Markdown Similarity Check Report"
    echo "Generated: $(date)"
    echo

    local markdown_files=($(find software infra -name "*.md" -type f 2>/dev/null | sort))
    local warnings=0
    local errors=0

    local file_count=${#markdown_files[@]}
    for i in $(seq 0 $((file_count - 2))); do
        local file1="${markdown_files[$i]}"
        for j in $(seq $((i + 1)) $((file_count - 1))); do
            local file2="${markdown_files[$j]}"

            # 除外チェック
            if [ -f "$IGNORE_FILE" ]; then
                local base1=$(basename "$file1")
                local base2=$(basename "$file2")
                if grep -qF "$base1" "$IGNORE_FILE" && grep -qF "$base2" "$IGNORE_FILE"; then
                    continue
                fi
            fi

            local similarity=$(calculate_similarity "$file1" "$file2")

            if [ "$similarity" -ge "$SIMILARITY_THRESHOLD" ]; then
                echo "⚠️  HIGH SIMILARITY: $similarity%"
                echo "   File1: $file1"
                echo "   File2: $file2"
                echo
                ((warnings++))
            fi
        done
    done

    echo "# Summary"
    echo "Warnings: $warnings"
    echo "Errors: $errors"

    if [ "$errors" -gt 0 ]; then
        echo "❌ Check FAILED"
        exit 1
    elif [ "$warnings" -gt 0 ]; then
        echo "⚠️  Check PASSED with warnings"
        exit 0
    else
        echo "✅ Check PASSED"
        exit 0
    fi
}

main "$@"
