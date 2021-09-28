package main

import (
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"log"
	"net/http"

	"github.com/PuerkitoBio/goquery"
)

func parseLatestVersion(doc goquery.Document) (string, error) {
	var latestversion string
	var err error

	doc.Find("div .accordion div .parbase.textblock div p b").Each(func(i int, s *goquery.Selection) {
		// Assuming the first paragraph with version in the text is the latest version
		matched, err := regexp.MatchString("[Vv]ersion", s.Text())
		if err == nil && matched {
			latestversion = s.Text()
		}
	})
	if len(latestversion) == 0 {
		err = fmt.Errorf("unable to parse the latest version in the page")
	}
	return latestversion, err
}

func parsePublishDate(doc goquery.Document) (string, error) {
	var publishdate string
	var err error

	// Find the document publish date metadata
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		name, _ := s.Attr("name")
		if name == "publish_date_timestamp" {
			pubdate, _ := s.Attr("content")
			timestamp, err := strconv.ParseInt(pubdate, 10, 64)
			if err == nil {
				t := time.Unix(timestamp, 0)
				publishdate = t.Format(time.UnixDate)
			}
		}
	})
	if len(publishdate) == 0 {
		err = fmt.Errorf("unable to parse the latest publication date in the page")
	}
	return publishdate, err
}

func getLatestRelease(hardware string) (string, string, error) {
	url := fmt.Sprintf("https://www.playstation.com/en-gb/support/hardware/%s/system-software/", hardware)
	var latestversion string
	var publishdate string

	resp, err := http.Get(url)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return publishdate, latestversion, fmt.Errorf("unable to fetch the update page, status code: %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return publishdate, latestversion, err
	}

	publishdate, err = parsePublishDate(*doc)
	if err != nil {
		return publishdate, latestversion, err
	}

	latestversion, err = parseLatestVersion(*doc)

	return publishdate, latestversion, err
}

func main() {
	var hardware = flag.String("hardware", "ps5", "Hardware to get the information for. Can be ps4 or ps5")

	flag.Parse()

	publishdate, latestversion, err := getLatestRelease(*hardware)

	if err != nil {
		panic(err)
	}

	fmt.Println(publishdate, latestversion)
}
