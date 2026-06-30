package anki

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// makeApkg 构造一个最小可用的 .apkg：包含 collection.anki2、media 映射与一个音频文件。
func makeApkg(t *testing.T, dir string) string {
	t.Helper()

	// 1. 构造 anki2 数据库
	dbPath := filepath.Join(dir, "collection.anki2")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE notes (id INTEGER PRIMARY KEY, flds TEXT)`); err != nil {
		t.Fatal(err)
	}
	// 字段以 \x1f 分隔：单词 / 音标+语音 / 释义（带 HTML）
	flds := "apple\x1f/ˈæpl/ [sound:apple.mp3]\x1f<b>n.</b> 苹果"
	if _, err := db.Exec(`INSERT INTO notes(flds) VALUES(?)`, flds); err != nil {
		t.Fatal(err)
	}
	db.Close()

	// 2. 构造一个假音频文件（zip 内名为 "0"）
	audioContent := []byte("FAKEAUDIO")

	// 3. media 映射
	mediaJSON, _ := json.Marshal(map[string]string{"0": "apple.mp3"})

	// 4. 打包成 zip(.apkg)
	apkgPath := filepath.Join(dir, "测试词库.apkg")
	zf, err := os.Create(apkgPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zf.Close()
	zw := zip.NewWriter(zf)

	writeEntry := func(name string, data []byte) {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	dbData, _ := os.ReadFile(dbPath)
	writeEntry("collection.anki2", dbData)
	writeEntry("media", mediaJSON)
	writeEntry("0", audioContent)

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return apkgPath
}

func TestParse(t *testing.T) {
	dir := t.TempDir()
	apkg := makeApkg(t, dir)
	mediaDir := filepath.Join(dir, "media")

	res, err := Parse(apkg, mediaDir)
	if err != nil {
		t.Fatalf("Parse 失败: %v", err)
	}

	if res.Name != "测试词库" {
		t.Errorf("单词本名 = %q, 期望 %q", res.Name, "测试词库")
	}
	if len(res.Words) != 1 {
		t.Fatalf("单词数 = %d, 期望 1", len(res.Words))
	}

	w := res.Words[0]
	if w.Term != "apple" {
		t.Errorf("Term = %q, 期望 apple", w.Term)
	}
	if w.Phonetic != "ˈæpl" {
		t.Errorf("Phonetic = %q, 期望 ˈæpl", w.Phonetic)
	}
	if w.Definition != "n. 苹果" {
		t.Errorf("Definition = %q, 期望 'n. 苹果'", w.Definition)
	}
	// 音频应被解压到 mediaDir
	if w.Audio == "" {
		t.Error("Audio 为空，期望指向解压后的音频文件")
	} else if _, err := os.Stat(w.Audio); err != nil {
		t.Errorf("音频文件未解压: %v", err)
	}
}

func TestClean(t *testing.T) {
	cases := map[string]string{
		"<b>hello</b>":            "hello",
		"a&nbsp;b":                "a b",
		"[sound:x.mp3]word":       "word",
		"  多   空格 ":               "多 空格",
	}
	for in, want := range cases {
		if got := clean(in); got != want {
			t.Errorf("clean(%q) = %q, 期望 %q", in, got, want)
		}
	}
}
