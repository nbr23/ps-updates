package main

import (
	"crypto/sha256"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

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
		if err == nil && matched && latestversion == "" {
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

func getHardwareURL(hardware string, local string) string {
	return fmt.Sprintf("https://www.playstation.com/%s/support/hardware/%s/system-software/", strings.ToLower(local), strings.ToLower(hardware))
}

func getLatestRelease(hardware string, local string) (psupdate, error) {
	url := getHardwareURL(hardware, local)
	var update psupdate

	resp, err := http.Get(url)

	if err != nil {
		return update, err
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

func (update psupdate) Guid() string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprint(update.ReleaseDate, update.VersionName))))
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

func (updates psupdates) writeAsRSS(wr io.Writer, hardware string, local string) error {
	rss_tpl := `
{{ $hardware := .Hardware }}
{{ $link := .Link }}
<rss version="2.0">
	<channel>
		<title>{{ $hardware }} Updates</title>
		<description>Feed of the {{ $hardware }} Software update releases</description>
		{{ range .Updates }}
		<item>
			<title>{{ $hardware }} Update: {{ .VersionName }}</title>
			<guid>{{ .Guid }}</guid>
			<description>The {{ $hardware }} software update {{ .VersionName }} was released on {{ .ReleaseDate }}</description>
			<pubDate>{{ .ReleaseDate }}</pubDate>
			<link>{{ $link }}</link>
		</item>
		{{ end }}
	</channel>
</rss>
`
	atom, err := template.New("rss").Parse(rss_tpl)

	if err != nil {
		return err
	}

	return atom.Execute(wr, struct {
		Hardware string
		Link     string
		Updates  []psupdate
	}{
		Hardware: strings.ToUpper(hardware),
		Link:     getHardwareURL(hardware, local),
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
	var hardware = flag.String("hardware", "ps5", "Hardware to get the information for. Can be \"ps4\" or \"ps5\"")
	var dbfilepath = flag.String("db", "", "Path to the sqlite3 database to store the versions into")
	var outputformat = flag.String("format", "text", "Output formatter. Can be \"text\" for plaintext, \"rss\" for an RSS XML")
	var outputfile = flag.String("output", "", "Output file path")
	var local = flag.String("local", "en-us", "Localisation of the PlayStation website to use to retrieve the updates. For best results, use an English based local: \"en-XX\"")
	var updates psupdates
	var output_fd = os.Stdout
	var err error

	flag.Parse()

	if strings.Compare(strings.ToLower(*hardware), "ps4") != 0 && strings.Compare(strings.ToLower(*hardware), "ps5") != 0 {
		panic("Only \"ps4\" and \"ps5\" are supported hardwares at this time")
	}

	latest_update, release_err := getLatestRelease(*hardware, *local)
	if release_err != nil {
		log.Printf("Unable to get latest release: %s\n", release_err)
	}

	if release_err == nil {
		updates = psupdates{latest_update}
	}

	if len(*dbfilepath) > 0 {
		if release_err == nil {
			err := writeToDB(*dbfilepath, *hardware, latest_update)
			if err != nil {
				log.Printf("Unable to save latest release to db: %s\n", err)
			}
		}

		updates, err = readUpdatesFromDB(*dbfilepath, *hardware)

		if err != nil {
			log.Printf("Unable to read db: %s\n", err)
			panic(err)
		}
	}

	if *outputfile != "" {
		fd, err := os.OpenFile(*outputfile, os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(fmt.Errorf("couldn't open output file %s", *outputfile))
		}
		output_fd = fd
		defer fd.Close()
	}

	switch strings.ToLower(*outputformat) {
	case "text":
		updates.writeAsString(output_fd, *hardware)
	case "rss":
		updates.writeAsRSS(output_fd, *hardware, *local)
	default:
		panic(fmt.Errorf("unsupported output format %s", *outputformat))
	}
}
