package main

import (
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const dbTimeFmt = "2006-01-02 15:04:05.999999999Z07:00"

const schema = `-- page table
CREATE TABLE IF NOT EXISTS pages(
	id INTEGER PRIMARY KEY,
	url TEXT,
	title TEXT,
	description TEXT,
	mime_type TEXT,
	added_at TEXT,

	UNIQUE(url)
);

CREATE VIRTUAL TABLE IF NOT EXISTS pages_fts USING fts5(
	url UNINDEXED, title, description, content=pages, content_rowid=id
);

CREATE TRIGGER IF NOT EXISTS pages_ai AFTER INSERT ON pages BEGIN
	INSERT INTO pages_fts(url, title, description) VALUES (new.url, new.title, new.description);
END;

CREATE TRIGGER IF NOT EXISTS pages_ad AFTER DELETE ON pages BEGIN
	INSERT INTO pages_fts(pages_fts, rowid, url, title, description) VALUES('delete', old.id, old.url, old.title, old.description);
END;
`

var columnNames = strings.Fields("id url title description mime_type added_at")
var columnNamesComma = strings.Join(columnNames, ",")
var columnPlaceholders = strings.TrimSuffix(strings.Repeat("?,", len(columnNames)), ",")
var columnNamesWithTable = prepend("pages.", columnNames)
var columnNamesWithTableComma = strings.Join(columnNamesWithTable, ",")

var insertPageSQL = `INSERT INTO pages(` + columnNamesComma + `) VALUES(` + columnPlaceholders + `)`

var searchSQL = `WITH results AS (
	SELECT rowid AS rid FROM pages_fts WHERE pages_fts MATCH ?
) SELECT ` + columnNamesWithTableComma + ` FROM pages, results WHERE pages.id = results.rid ORDER BY pages.added_at DESC
`

var searchSQLCount = `SELECT count(*) FROM pages_fts WHERE pages_fts MATCH ?`

var likeSQL = `SELECT ` + columnNamesComma + ` FROM pages WHERE title LIKE ? ORDER BY added_at DESC`

var likeSQLCount = `SELECT count(*) FROM pages WHERE title LIKE ?`

var (
	dbFile string
	db     *sql.DB
)

func openDatabase(dataSourceName string) {
	if d, err := sql.Open("sqlite3", "file:"+dataSourceName); err == nil {
		db = d
		dbFile = dataSourceName
	} else {
		log.Fatalf("can't open database %s: %s", dataSourceName, err)
	}

	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("can't create schema: %s", err)
	}
}

func closeDatabase() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("can't close database %s: %s", dbFile, err)
		}
	}
}

func search(q string) ([]*page, error) {
	return rowsToPagesWithQuery(db.Query(searchSQL, q))
}

func searchCount(q string) (count int, err error) {
	err = db.QueryRow(searchSQLCount, q).Scan(&count)
	return
}

func like(q string) ([]*page, error) {
	return rowsToPagesWithQuery(db.Query(likeSQL, q))
}

func likeCount(q string) (count int, err error) {
	err = db.QueryRow(likeSQLCount, q).Scan(&count)
	return
}

func rowsToPagesWithQuery(rows *sql.Rows, err error) ([]*page, error) {
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rowsToPages(rows)
}

func rowsToPages(rows *sql.Rows) (pages []*page, err error) {
	attrs := []interface{}{
		0: &sql.NullInt64{},  // id
		1: &sql.NullString{}, // url
		2: &sql.NullString{}, // title
		3: &sql.NullString{}, // description
		4: &sql.NullString{}, // mime_type
		5: &sql.NullString{}, // added_at
	}

	for rows.Next() {
		if err = rows.Scan(attrs...); err != nil {
			break
		}

		addedAt, err := time.Parse(dbTimeFmt, attrs[5].(*sql.NullString).String)
		if err != nil {
			break
		}

		pages = append(pages, &page{
			URL:         attrs[1].(*sql.NullString).String,
			Title:       attrs[2].(*sql.NullString).String,
			Description: attrs[3].(*sql.NullString).String,
			MimeType:    attrs[4].(*sql.NullString).String,
			AddedAt:     addedAt,
		})
	}

	err = rows.Err()
	return
}

func insertPage(pg *page) error {
	_, err := db.Exec(insertPageSQL,
		nil,
		pg.URL,
		nilIfEmpty(pg.Title),
		nilIfEmpty(pg.Description),
		nilIfEmpty(pg.MimeType),
		pg.AddedAt,
	)

	return err
}

func nilIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func prepend(hdr string, strs []string) []string {
	t := make([]string, len(strs))
	for i, s := range strs {
		t[i] = hdr + s
	}
	return t
}
