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
	atom_url TEXT,
	tag TEXT,
	referrer TEXT,
	host TEXT,
	is_root_page INTEGER,
	added_at TEXT,

	UNIQUE(url)
);

CREATE INDEX IF NOT EXISTS pages_tag ON pages(tag);
CREATE INDEX IF NOT EXISTS pages_referrer ON pages(referrer);
CREATE INDEX IF NOT EXISTS pages_host ON pages(host);

CREATE TABLE IF NOT EXISTS pages_tags(
	host TEXT,
	tag TEXT
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

var columnNames = strings.Fields("id url title description mime_type atom_url tag referrer host is_root_page added_at")
var columnNamesComma = strings.Join(columnNames, ",")
var columnPlaceholders = strings.TrimSuffix(strings.Repeat("?,", len(columnNames)), ",")
var columnNamesWithTable = prepend("pages.", columnNames)
var columnNamesWithTableComma = strings.Join(columnNamesWithTable, ",")

var insertPageSQL = `INSERT INTO pages(` + columnNamesComma + `) VALUES(` + columnPlaceholders + `)`

var listPagesByTagSQL = `SELECT ` + columnNamesComma + ` FROM pages WHERE tag = ? ORDER BY added_at DESC`

var listPagesByReferrerSQL = `SELECT ` + columnNamesComma + ` FROM pages WHERE referrer = ? ORDER BY added_at DESC`

var searchSQL = `WITH results AS (
	SELECT rowid AS rid FROM pages_fts WHERE pages_fts MATCH ?
) SELECT ` + columnNamesWithTableComma + ` FROM pages, results WHERE pages.id = results.rid ORDER BY pages.added_at DESC
`

var searchSQLCount = `SELECT count(*) FROM pages_fts WHERE pages_fts MATCH ?`

var likeSQL = `SELECT ` + columnNamesComma + ` FROM pages WHERE title LIKE ? ORDER BY added_at DESC`

var likeSQLCount = `SELECT count(*) FROM pages WHERE title LIKE ?`

var tagCountSQL = `SELECT tag, count(*) FROM pages GROUP BY tag ORDER BY 2 DESC`

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

	if t, err := readPredefinedTags(); err == nil {
		predefinedTags = t
	} else {
		log.Fatal(err)
	}
}

func closeDatabase() {
	if db != nil {
		if err := db.Close(); err != nil {
			log.Printf("can't close database %s: %s", dbFile, err)
		}
	}
}

func listByTag(tag string) ([]*page, error) {
	return rowsToPagesWithQuery(db.Query(listPagesByTagSQL, tag))
}

func listByReferrer(referrer string) ([]*page, error) {
	return rowsToPagesWithQuery(db.Query(listPagesByReferrerSQL, referrer))
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
		0:  &sql.NullInt64{},  // id
		1:  &sql.NullString{}, // url
		2:  &sql.NullString{}, // title
		3:  &sql.NullString{}, // description
		4:  &sql.NullString{}, // mime_type
		5:  &sql.NullString{}, // atom_url
		6:  &sql.NullString{}, // tag
		7:  &sql.NullString{}, // referrer
		8:  &sql.NullString{}, // host
		9:  &sql.NullBool{},   // is_root_page
		10: &sql.NullString{}, // added_at
	}

	for rows.Next() {
		if err = rows.Scan(attrs...); err != nil {
			break
		}

		addedAt, err := time.Parse(dbTimeFmt, attrs[10].(*sql.NullString).String)
		if err != nil {
			break
		}

		pages = append(pages, &page{
			URL:         attrs[1].(*sql.NullString).String,
			Title:       attrs[2].(*sql.NullString).String,
			Description: attrs[3].(*sql.NullString).String,
			MimeType:    attrs[4].(*sql.NullString).String,
			AtomURL:     attrs[5].(*sql.NullString).String,
			Tag:         attrs[6].(*sql.NullString).String,
			Referrer:    attrs[7].(*sql.NullString).String,
			Host:        attrs[8].(*sql.NullString).String,
			IsRootPage:  attrs[9].(*sql.NullBool).Bool,
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
		nilIfEmpty(pg.AtomURL),
		nilIfEmpty(pg.Tag),
		nilIfEmpty(pg.Referrer),
		pg.Host,
		boolToInt(pg.IsRootPage),
		pg.AddedAt,
	)

	return err
}

func readPredefinedTags() ([]tag, error) {
	rows, err := db.Query(`SELECT host, tag FROM pages_tags`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]tag, 0, 64)

	var host, vtag sql.NullString
	for rows.Next() {
		if err := rows.Scan(&host, &vtag); err != nil {
			return nil, err
		}
		tags = append(tags, tag{host.String, vtag.String})
	}

	return tags, nil
}

func tagCounts() ([]tagCount, error) {
	rows, err := db.Query(`SELECT tag, count(*) FROM pages GROUP BY tag ORDER BY 2 DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make([]tagCount, 0)
	noTag := 0

	var tag sql.NullString
	var count sql.NullInt32
	for rows.Next() {
		if err := rows.Scan(&tag, &count); err != nil {
			return nil, err
		}
		if tag.String == "" {
			noTag++
		} else {
			counts = append(counts, tagCount{tag.String, int(count.Int32)})
		}
	}
	if noTag > 0 {
		counts = append(counts, tagCount{"", noTag})
	}

	return counts, nil
}

func nilIfEmpty(s string) interface{} {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func prepend(hdr string, strs []string) []string {
	t := make([]string, len(strs))
	for i, s := range strs {
		t[i] = hdr + s
	}
	return t
}
