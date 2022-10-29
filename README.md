# ps-updates

Simple tool to monitor PS4/PS5 updates release, and generate an RSS feed of update releases.

Add the following feeds to your RSS Reader to be informed of the latest PS4 and PS5 update releases:

- [PS5 Updates RSS](https://ps.wip.tf/PS5.xml)
- [PS4 Updates RSS](https://ps.wip.tf/PS4.xml)

![RSS Feed][1]

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
  -local string
        Localisation of the PlayStation website to use to retrieve the updates. For best results, use an English based local: "en-XX" (default "en-us")
  -output string
        Output file path
```

To generate the rss feed formatted file for PS4 updates, add the following command in your scheduler (crontab, etc):

`psupdates -format rss -hardware ps4 -db ~/.config/psupdates.db -output $WWW_DIR/PS4_updates.xml`

[1]:docs/rss_screen.png