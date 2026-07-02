package store

import (
	"path/filepath"
	"testing"

	"simpleword/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestDeckLifecycle(t *testing.T) {
	s := newTestStore(t)

	id, err := s.CreateDeck("CET4")
	if err != nil {
		t.Fatal(err)
	}

	words := []model.Word{
		{Term: "apple", Phonetic: "ˈæpl", Definition: "苹果"},
		{Term: "banana", Definition: "香蕉"},
	}
	if err := s.AddWords(id, words); err != nil {
		t.Fatal(err)
	}

	var storedDefinition string
	if err := s.db.QueryRow(`SELECT definition FROM words WHERE term = ?`, "apple").Scan(&storedDefinition); err != nil {
		t.Fatal(err)
	}
	if storedDefinition != "\x1f苹果\x1f" {
		t.Errorf("数据库 Definition = %q, 期望用 \\x1f 包围", storedDefinition)
	}

	decks, err := s.ListDecks()
	if err != nil {
		t.Fatal(err)
	}
	if len(decks) != 1 || decks[0].Name != "CET4" {
		t.Fatalf("ListDecks = %+v", decks)
	}

	st, err := s.Stats(id)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 2 || st.Mastered != 0 {
		t.Errorf("Stats = %+v, 期望 Total=2 Mastered=0", st)
	}

	// 取一轮并标记掌握
	round, err := s.PickRound(id, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(round) != 2 {
		t.Fatalf("PickRound = %d, 期望 2", len(round))
	}
	if round[0].Definition != "\x1f苹果\x1f" {
		t.Errorf("读取 Definition = %q, 期望保留 \\x1f 包围", round[0].Definition)
	}
	if err := s.MarkMastered(round[0].ID); err != nil {
		t.Fatal(err)
	}
	st, _ = s.Stats(id)
	if st.Mastered != 1 {
		t.Errorf("掌握后 Mastered = %d, 期望 1", st.Mastered)
	}

	// 删除单词本及其单词
	if err := s.DeleteDeck(id); err != nil {
		t.Fatal(err)
	}
	decks, _ = s.ListDecks()
	if len(decks) != 0 {
		t.Errorf("删除后仍有 %d 个单词本", len(decks))
	}
	st, _ = s.Stats(id)
	if st.Total != 0 {
		t.Errorf("删除后仍有 %d 个单词", st.Total)
	}
}

func TestPickRoundOrder(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.CreateDeck("d")
	s.AddWords(id, []model.Word{{Term: "a"}, {Term: "b"}, {Term: "c"}})

	round, _ := s.PickRound(id, 1)
	if len(round) != 1 {
		t.Fatalf("PickRound(1) = %d", len(round))
	}
	// 掌握第一个后，下一轮应取未掌握的
	s.MarkMastered(round[0].ID)
	next, _ := s.PickRound(id, 1)
	if next[0].Mastered {
		t.Error("PickRound 应优先返回未掌握单词")
	}
}
