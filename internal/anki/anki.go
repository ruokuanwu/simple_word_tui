// Package anki 解析 Anki 的 .apkg 文件，将其转换为应用自身的数据格式。
//
// .apkg 是一个 zip 压缩包，内部包含：
//   - collection.anki2 : 一个 SQLite 数据库，notes 表存储笔记，字段以 \x1f 分隔
//   - media            : 一个 JSON，映射 {"0":"原始文件名", ...}
//   - 0, 1, 2 ...       : 以数字命名的媒体文件
package anki

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jaytaylor/html2text"

	"simpleword/internal/model"

	_ "modernc.org/sqlite"
)

// Result 是一次解析的结果。
type Result struct {
	Name  string       // 单词本名（取自文件名）
	Words []model.Word // 解析出的单词
}

var (
	htmlTag    = regexp.MustCompile(`<[^>]+>`)
	soundTag   = regexp.MustCompile(`\[sound:([^\]]+)\]`)
	phoneticRe = regexp.MustCompile(`/([^/\s][^/]*)/|\[([^\]\s][^\]]*)\]`)
	spaceRe    = regexp.MustCompile(`\s+`)
)

// Parse 解析指定的 apkg 文件，返回单词本数据。
// mediaDir 是媒体文件根目录（音频会被释放到以单词本名命名的子目录中）。
func Parse(apkgPath, mediaDir string) (*Result, error) {
	zr, err := zip.OpenReader(apkgPath)
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	name := strings.TrimSuffix(filepath.Base(apkgPath), filepath.Ext(apkgPath))

	// 1. 读取 media 映射，建立 {数字文件名 -> 原始文件名}
	mediaMap, err := readMediaMap(zr)
	if err != nil {
		return nil, err
	}

	// 2. 找到并解压 collection.anki2 到临时文件
	dbPath, cleanup, err := extractAnki2(zr)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// 3. 解压音频媒体文件到当前单词本目录，建立 {原始文件名 -> 本地绝对路径}
	audioPaths, err := extractMedia(zr, mediaMap, filepath.Join(mediaDir, name))
	if err != nil {
		return nil, err
	}

	// 4. 读取 anki2 数据库的 notes
	words, err := readNotes(dbPath, audioPaths)
	if err != nil {
		return nil, err
	}

	return &Result{Name: name, Words: words}, nil
}

// readMediaMap 读取 zip 中的 media JSON 文件。
func readMediaMap(zr *zip.ReadCloser) (map[string]string, error) {
	for _, f := range zr.File {
		if f.Name == "media" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return nil, err
			}
			m := map[string]string{}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &m); err != nil {
					return nil, err
				}
			}
			return m, nil
		}
	}
	return map[string]string{}, nil
}

// extractAnki2 将集合数据库解压到临时文件，返回路径与清理函数。
//
// 现代 .apkg（schema 11+）通常同时包含 collection.anki2 与 collection.anki21，
// 其中 .anki2 只是为兼容旧版保留的空占位库，.anki21 才是真正的数据库，
// 因此必须优先选用 .anki21，否则会导入到空库（0 个单词）。
func extractAnki2(zr *zip.ReadCloser) (string, func(), error) {
	var chosen *zip.File
	for _, f := range zr.File {
		switch f.Name {
		case "collection.anki21":
			chosen = f // 最高优先级，直接选中
		case "collection.anki2":
			if chosen == nil {
				chosen = f // 仅在没有 .anki21 时作为回退
			}
		}
	}
	if chosen == nil {
		return "", nil, os.ErrNotExist
	}

	rc, err := chosen.Open()
	if err != nil {
		return "", nil, err
	}
	defer rc.Close()

	tmp, err := os.CreateTemp("", "anki-*.db")
	if err != nil {
		return "", nil, err
	}
	if _, err := io.Copy(tmp, rc); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", nil, err
	}
	tmp.Close()
	path := tmp.Name()
	return path, func() { os.Remove(path) }, nil
}

// extractMedia 将音频类媒体文件解压到 mediaDir，返回 {原始文件名 -> 本地绝对路径}。
func extractMedia(zr *zip.ReadCloser, mediaMap map[string]string, mediaDir string) (map[string]string, error) {
	if err := os.MkdirAll(mediaDir, 0o755); err != nil {
		return nil, err
	}
	// 建立 zip 内文件名索引
	byName := map[string]*zip.File{}
	for _, f := range zr.File {
		byName[f.Name] = f
	}

	out := map[string]string{}
	for numName, origName := range mediaMap {
		if !isAudio(origName) {
			continue
		}
		zf, ok := byName[numName]
		if !ok {
			continue
		}
		dst := filepath.Join(mediaDir, origName)
		if err := copyZipFile(zf, dst); err != nil {
			return nil, err
		}
		out[origName] = dst
	}
	return out, nil
}

// copyZipFile 将 zip 中的文件写出到 dst。
func copyZipFile(zf *zip.File, dst string) error {
	rc, err := zf.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, rc)
	return err
}

// readNotes 从 anki2 数据库读取 notes 表并转换为 Word。
func readNotes(dbPath string, audioPaths map[string]string) ([]model.Word, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT flds FROM notes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var words []model.Word
	for rows.Next() {
		var flds string
		if err := rows.Scan(&flds); err != nil {
			return nil, err
		}
		// 字段以 \x1f 分隔
		fields := strings.Split(flds, "\x1f")
		w := fieldsToWord(fields, audioPaths)
		if w.Term != "" {
			words = append(words, w)
		}
	}
	return words, rows.Err()
}

// fieldsToWord 将一条 note 的多个字段映射为 Word。
// 约定：第 1 个非空字段为单词，其余字段拼接为释义；从中提取音标和语音。
func fieldsToWord(fields []string, audioPaths map[string]string) model.Word {
	var w model.Word
	var defParts []string

	for i, raw := range fields {
		// 提取语音：[sound:xxx.mp3]
		if m := soundTag.FindStringSubmatch(raw); m != nil {
			if p, ok := audioPaths[m[1]]; ok && w.Audio == "" {
				w.Audio = p
			}
		}
		text := clean(raw)
		if text == "" {
			continue
		}
		if w.Term == "" && i == 0 {
			w.Term = text
			continue
		}
		// 提取音标（首次出现 /.../ 或 [...]）
		if w.Phonetic == "" {
			if p := extractPhonetic(raw); p != "" {
				w.Phonetic = p
				continue
			}
		}
		defParts = append(defParts, text)
	}
	w.Definition = strings.Join(defParts, "\x1f")
	return w
}

// extractPhonetic 从字段中提取音标，跳过 [sound:...] 和普通比例/路径里的斜杠。
func extractPhonetic(s string) string {
	s = soundTag.ReplaceAllString(s, "")
	s = htmlTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")

	for _, m := range phoneticRe.FindAllStringSubmatch(s, -1) {
		p := strings.TrimSpace(m[1])
		if p == "" {
			p = strings.TrimSpace(m[2])
		}
		if isPhonetic(p) {
			return p
		}
	}
	return ""
}

// isPhonetic 判断候选文本是否像音标。
func isPhonetic(s string) bool {
	for _, r := range s {
		if strings.ContainsRune("ˈˌəɪʊʌæɑɒɔɜɚɝθðʃʒŋɡːˑ", r) {
			return true
		}
	}
	return false
}

// clean 去除 HTML 标签、sound 标签并规整空白。
func clean(s string) string {
	s = soundTag.ReplaceAllString(s, "")
	text, err := html2text.FromString(s, html2text.Options{OmitLinks: true})
	if err == nil {
		s = text
	}
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "\u00a0", " ")
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// isAudio 判断文件名是否为受支持的音频格式。
func isAudio(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp3", ".wav", ".ogg", ".m4a", ".flac":
		return true
	}
	return false
}
