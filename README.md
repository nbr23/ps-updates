# ps-updates

Simple tool to monitor PS4/PS5 updates release, and generate an RSS feed of update releases

## Requirements

This can be built using golang, and leverages sqlite3 to store versions history

## Usage

```
Usage of psupdates:
  -db string
        Path to the sqlite3 database to store the versions into
  -format string
        Output formatter. Can be "text" for plaintext, "rss" for an RSS XML (default "text")
  -hardware string
        Hardware to get the information for. Can be "ps4" or "ps5" (default "ps5")
```

To generate the rss feed formatted file for PS4 updates, add the following command in your scheduler (crontab, etc):

`psupdates --format rss --hardware ps4 --db ~/.config/psupdates.db > $WWW_DIR/PS4_updates.xml`