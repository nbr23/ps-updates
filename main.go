package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/mattn/go-sqlite3"
)

type psupdate struct {
	ReleaseDate      string
	ReleaseTimeStamp int64
	VersionName      string
}
type psupdates []psupdate

func parseLatestVersion(doc goquery.Document) (string, error) {
	var latestversion string
	var err error

	doc.Find("div .accordion div .parbase.textblock div p b").Each(func(i int, s *goquery.Selection) {
		// Assuming the first paragraph with version in the text is the latest version
		matched, err := regexp.MatchString("[Vv]ersion", s.Text())
		if err == nil && matched {
			latestversion = strings.TrimSpace(s.Text())
		}
	})
	if len(latestversion) == 0 {
		err = fmt.Errorf("unable to parse the latest version in the page")
	}
	return latestversion, err
}

func parsePublishDate(doc goquery.Document) (int64, string, error) {
	var publishtimestamp int64
	var publishdate string
	var err error

	// Find the document publish date metadata
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		if name == "publish_date_timestamp" {
			pubdate, _ := s.Attr("content")
			publishtimestamp, err = strconv.ParseInt(pubdate, 10, 64)
			if err == nil {
				t := time.Unix(publishtimestamp, 0)
				publishdate = t.Format(time.UnixDate)
			}
		}
	})
	return publishtimestamp, publishdate, err
}

func getLatestRelease(hardware string) (psupdate, error) {
	url := fmt.Sprintf("https://www.playstation.com/en-gb/support/hardware/%s/system-software/", hardware)
	var update psupdate

	resp, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return update, fmt.Errorf("unable to fetch the update page, status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return update, err
	}

	update.ReleaseTimeStamp, update.ReleaseDate, err = parsePublishDate(*doc)
	if err != nil {
		return update, err
	}

	update.VersionName, err = parseLatestVersion(*doc)

	return update, err
}

func readUpdatesFromDB(dbpath string, hardware string) ([]psupdate, error) {
	var updates []psupdate
	db, err := sql.Open("sqlite3", dbpath)

	if err != nil {
		return updates, err
	}
	defer db.Close()

	stmt, err := db.Prepare(fmt.Sprintf("SELECT pubtimestamp, pubdate, version FROM %s ORDER BY pubtimestamp DESC", hardware))
	if err != nil {
		return updates, err
	}
	rows, err := stmt.Query()
	if err != nil {
		return updates, err
	}

	for rows.Next() {
		var update psupdate
		rows.Scan(&update.ReleaseTimeStamp, &update.ReleaseDate, &update.VersionName)
		updates = append(updates, update)
	}
	return updates, err
}

func writeToDB(dbpath string, hardware string, update psupdate) error {
	db, err := sql.Open("sqlite3", dbpath)

	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (pubtimestamp INTEGER, pubdate TEXT, version TEXT);", hardware))
	if err != nil {
		return err
	}

	// Check if in table already
	stmt, err := db.Prepare(fmt.Sprintf("SELECT * FROM %s WHERE version = ?", hardware))
	if err != nil {
		return err
	}
	rows, err := stmt.Query(update.VersionName)
	if err != nil {
		return err
	}

	defer rows.Close()

	if !rows.Next() {
		stmt, err := db.Prepare(fmt.Sprintf("INSERT INTO %s(pubtimestamp, pubdate, version) VALUES(?,?,?)", hardware))
		if err != nil {
			return err
		}
		_, err = stmt.Exec(update.ReleaseTimeStamp, update.ReleaseDate, update.VersionName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (updates psupdates) writeAsRSS(wr io.Writer, hardware string) error {
	atom, err := template.ParseFiles("templates/rss.goxml")

	if err != nil {
		return err
	}

	return atom.Execute(wr, struct {
		Hardware string
		Updates  []psupdate
	}{
		Hardware: strings.ToUpper(hardware),
		Updates:  updates,
	},
	)
}

func (updates psupdates) writeAsString(wr io.Writer, hardware string) error {
	fmt.Fprintf(wr, "%s Updates:\n", strings.ToUpper(hardware))
	for _, update := range updates {
		fmt.Fprintf(wr, "- %s: %s\n", update.ReleaseDate, update.VersionName)
	}
	return nil
}

func main() {
	var hardware = flag.String("hardware", "ps5", "Hardware to get the information for. Can be ps4 or ps5")

	flag.Parse()

	update, err := getLatestRelease(strings.ToLower(*hardware))

	if err != nil {
		panic(err)
	}

	fmt.Println(publishdate, latestversion)
}
