// Package store 负责所有数据的 SQLite 持久化。
package store

import (
	"database/sql"
	"strings"
	"time"

	"simpleword/internal/model"

	_ "modernc.org/sqlite"
)

// Store 封装数据库连接。
type Store struct {
	db *sql.DB
}

const definitionSep = "\x1f"

// Open 打开（并初始化）指定路径的 SQLite 数据库。
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.db.Close()
}

// init 创建所需的数据表。
func (s *Store) init() error {
	const schema = `
CREATE TABLE IF NOT EXISTS decks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	created_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS words (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	deck_id      INTEGER NOT NULL,
	term         TEXT NOT NULL,
	phonetic     TEXT NOT NULL DEFAULT '',
	definition   TEXT NOT NULL DEFAULT '',
	audio        TEXT NOT NULL DEFAULT '',
	mastered     INTEGER NOT NULL DEFAULT 0,
	review_count INTEGER NOT NULL DEFAULT 0,
	last_review  INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY(deck_id) REFERENCES decks(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_words_deck ON words(deck_id);
`
	_, err := s.db.Exec(schema)
	return err
}

// CreateDeck 创建一个单词本并返回其 ID。
func (s *Store) CreateDeck(name string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO decks(name, created_at) VALUES(?, ?)`,
		name, time.Now().Unix(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListDecks 返回所有单词本。
func (s *Store) ListDecks() ([]model.Deck, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM decks ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var decks []model.Deck
	for rows.Next() {
		var d model.Deck
		var ts int64
		if err := rows.Scan(&d.ID, &d.Name, &ts); err != nil {
			return nil, err
		}
		d.CreatedAt = time.Unix(ts, 0)
		decks = append(decks, d)
	}
	return decks, rows.Err()
}

// DeleteDeck 删除单词本及其全部单词。
func (s *Store) DeleteDeck(id int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM words WHERE deck_id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM decks WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// AddWords 批量插入单词到指定单词本（在一个事务中）。
func (s *Store) AddWords(deckID int64, words []model.Word) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(
		`INSERT INTO words(deck_id, term, phonetic, definition, audio) VALUES(?, ?, ?, ?, ?)`,
	)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, w := range words {
		if _, err := stmt.Exec(deckID, w.Term, w.Phonetic, encodeDefinition(w.Definition), w.Audio); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// Stats 返回某单词本的学习统计。
func (s *Store) Stats(deckID int64) (model.DeckStats, error) {
	var st model.DeckStats
	err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(mastered), 0) FROM words WHERE deck_id = ?`,
		deckID,
	).Scan(&st.Total, &st.Mastered)
	return st, err
}

// PickRound 从单词本中取出一轮（n 个）单词，优先选择未掌握且复习次数少的单词。
func (s *Store) PickRound(deckID int64, n int) ([]model.Word, error) {
	rows, err := s.db.Query(
		`SELECT id, deck_id, term, phonetic, definition, audio, mastered, review_count, last_review
		 FROM words
		 WHERE deck_id = ?
		 ORDER BY mastered ASC, review_count ASC, last_review ASC
		 LIMIT ?`,
		deckID, n,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanWords(rows)
}

// MarkMastered 将单词标记为已掌握，并更新复习信息。
func (s *Store) MarkMastered(wordID int64) error {
	_, err := s.db.Exec(
		`UPDATE words SET mastered = 1, review_count = review_count + 1, last_review = ? WHERE id = ?`,
		time.Now().Unix(), wordID,
	)
	return err
}

// scanWords 将查询结果扫描为 Word 切片。
func scanWords(rows *sql.Rows) ([]model.Word, error) {
	var words []model.Word
	for rows.Next() {
		var w model.Word
		var mastered int
		var lastReview int64
		if err := rows.Scan(
			&w.ID, &w.DeckID, &w.Term, &w.Phonetic, &w.Definition, &w.Audio,
			&mastered, &w.ReviewCount, &lastReview,
		); err != nil {
			return nil, err
		}
		w.Mastered = mastered == 1
		w.LastReview = time.Unix(lastReview, 0)
		words = append(words, w)
	}
	return words, rows.Err()
}

func encodeDefinition(def string) string {
	if def == "" || strings.HasPrefix(def, definitionSep) && strings.HasSuffix(def, definitionSep) {
		return def
	}
	return definitionSep + def + definitionSep
}
