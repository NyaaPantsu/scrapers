package scraper

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	//"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

/*
	### Anidex categories ###
	Anime - Sub
	Anime - Raw
	Anime - Dub
	LA - Sub
	LA - Raw
	Light Novel
	Manga - TLed
	Manga - Raw
	♫ - Lossy
	♫ - Lossless
	♫ - Video
	Games
	Applications
	Pictures
	Adult Video
	Other
*/

//getAnidexMax returns the current number of torrents available so we know when to stop increasing the offset
//Deprecated for now
func getAnidexMax(b []byte) int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
	if err != nil {
		fmt.Println("Could not get offset max")
		return 0
	}
	str := doc.Find("p.text-center:nth-child(2)").Text()
	fmt.Println(str)
	//Wew lad, should probably make this a regex find
	str = str[(len(str)-15):(len(str)-13)] + str[(len(str)-12):(len(str)-9)]
	max, err := strconv.Atoi(str)
	if err != nil {
		fmt.Println("Could not parse offset max")
		return 0
	}
	return max
}

func anidexParent(startOffset, maxPages int, chHTML chan<- HTMLBlob) {
	anidexOffset := startOffset
	//Do it for as many page as specified
	var Blob HTMLBlob
	Blob.URL = "https://anidex.info"
	for i := 0; i < maxPages; i++ {

		//Fetch the page at the specified offset
		fmt.Println("Fetching anidex page offset", anidexOffset)
		req, err := http.NewRequest("GET",
			"https://anidex.info/ajax/page.ajax.php?page=torrents&category=0&filename=&lang_id=&r=0&b=0&order_by=upload_timestamp&order=desc&limit=50&offset="+strconv.Itoa(anidexOffset), nil)
		if err != nil {
			fmt.Println(err)
		}

		//Set it as an XML request
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		//Read dat shit yo
		b, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close() //We can just close the body right away
		if err != nil {
			log.Fatal(err)
		}

		//Increment our offset by 50
		anidexOffset += 50
		Blob.Raw = b
		chHTML <- Blob
	}
}

//anidexChild crawls anidex torrent pages for relevant info
func anidexChild(chTorrent chan<- Torrent, chPageID chan string) {
	//I tried to move these maps out, but golang doesnt allow maps to be constants
	//So here they are, in their enormous, bulky glory
	anidexAdultCategories := map[string][]int{
		"Anime - Sub":  []int{1, 1},
		"Anime - Raw":  []int{1, 1},
		"Anime - Dub":  []int{1, 1},
		"LA - Sub":     []int{2, 2},
		"LA - Raw":     []int{2, 2},
		"Light Novel":  []int{1, 2},
		"Manga - TLed": []int{1, 4},
		"Manga - Raw":  []int{1, 4},
		"♫ - Video":    []int{1, 1},
		"Games":        []int{1, 3},
		"Applications": []int{1, 3},
		"Pictures":     []int{2, 1},
		"Adult Video":  []int{2, 2},
		"Other":        []int{7, 1},
	}
	anidexCategories := map[string][]int{
		"Anime - Sub":  []int{3, 5},
		"Anime - Raw":  []int{3, 6},
		"Anime - Dub":  []int{3, 5},
		"LA - Sub":     []int{5, 9},
		"LA - Raw":     []int{5, 11},
		"Light Novel":  []int{4, 8},
		"Manga - TLed": []int{4, 7},
		"Manga - Raw":  []int{4, 8},
		"♫ - Lossy":    []int{2, 4},
		"♫ - Lossless": []int{2, 3},
		"♫ - Video":    []int{3, 12},
		"Games":        []int{6, 15},
		"Applications": []int{1, 1},
		"Pictures":     []int{6, 16},
		"Adult Video":  []int{2, 2},
	}
	fileSizes := map[string]int{
		"GB": 3,
		"MB": 2,
		"KB": 1,
	}
	for pageID := range chPageID {
		var info Torrent

		//Try and get the page
		req, err := http.NewRequest("GET",
			"https://anidex.info/ajax/page.ajax.php?page=torrent&id="+pageID, nil)
		if err != nil {
			fmt.Println(err)
		}

		//Set it so we get the page generated from AJAX because dynamic cancer
		req.Header.Set("X-Requested-With", "XMLHttpRequest")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(err)
		}

		//Read what we got
		b, _ := ioutil.ReadAll(resp.Body)
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
		if err != nil {
			fmt.Println("Could not read file")
			resp.Body.Close()
			continue
		}

		//Begin parsing that shit in this big hideous block
		info.Source = "https://anidex.info/?page=torrent&id=" + pageID
		info.Title = doc.Find("div.panel:nth-child(1) > div:nth-child(1) > h3:nth-child(1)").Text()
		if len(info.Title) > 3 {
			info.Title = strings.TrimSpace(info.Title[:len(info.Title)-3])
		}
		info.Hash = strings.ToUpper(doc.Find("#edit_torrent_form > div:nth-child(1) > div:nth-child(2) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(7) > td:nth-child(2)").Text())
		info.Magnet, _ = doc.Find("a.btn-default:nth-child(2)").Attr("href")
		info.Uploader = strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(1) > td:nth-child(2)").Text())
		info.Language = strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(2) > td:nth-child(2)").Text())
		info.Label = strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(4) > td:nth-child(2)").Text())
		info.Date = strings.TrimSpace(doc.Find("#edit_torrent_form > div:nth-child(1) > div:nth-child(2) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(1) > td:nth-child(2)").Text())
		info.Seeders, _ = strconv.Atoi(strings.TrimSpace(doc.Find("td.text-success:nth-child(2)").Text()))
		info.Leechers, _ = strconv.Atoi(strings.TrimSpace(doc.Find("td.text-danger:nth-child(2)").Text()))
		info.Completed, _ = strconv.Atoi(strings.TrimSpace(doc.Find(".text-info").Text()))

		checkAdult := strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(4) > td:nth-child(2) > span:nth-child(1)").Text())
		if checkAdult == "Hentai" {
			info.Adult = true
		} else {
			info.Adult = false
		}

		category := strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(3) > td:nth-child(2) > div:nth-child(1)").Text())
		if info.Adult {
			copy(info.Category[:2], anidexAdultCategories[category][:2])
		} else {
			copy(info.Category[:2], anidexCategories[category][:2])
		}

		//Deal with special case bullshit
		if info.Language != "English" && !info.Adult && info.Category[1] != 6 {
			switch info.Category[0] {
			case 3:
				info.Category[1] = 13
			case 4:
				info.Category[1] = 14
			case 5:
				info.Category[1] = 18
			default:
			}
		}

		//Filesize comes in as an array of a float (the size) and a string (the format), e.g. [12.34, GiB] or [1.23, MiB]
		fileSize := strings.Split(strings.TrimSpace(doc.Find("#edit_torrent_form > div:nth-child(1) > div:nth-child(2) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(2) > td:nth-child(2)").Text()), " ")
		fileFloat, err := strconv.ParseFloat(fileSize[0], 32)
		if err != nil {
			fmt.Println(err)
		}
		for i := 0; i < fileSizes[fileSize[1]]; i++ {
			fileFloat *= 1024
		}
		info.FileSize = int(fileFloat)

		//Get and sanitize the description block for the torrent
		descriptionHTML, err := doc.Find(".panel-body > span:nth-child(1)").Html()
		if err != nil {
			fmt.Println("Error getting description HTML", err)
		}

		info.Description = bluemonday.UGCPolicy().Sanitize(descriptionHTML)
		if len(info.Description) < 1 {
			info.Description = "No description found"
		}

		//If we're missing any of these the scrape likely went bad, send the page back into the queue
		if len(info.Title) < 1 || len(info.Hash) == 0 || len(info.Magnet) == 0 {
			fmt.Println("Failed to read", pageID, "sending back into queue")
			chPageID <- pageID
			resp.Body.Close()
			continue
		}

		//If the channel is full, be a sleepy baby
		for len(chTorrent) == cap(chTorrent) {
			fmt.Println("Torrent channel full, sleeping 3 seconds")
			time.Sleep(time.Millisecond * 3000)
		}

		chTorrent <- info
		resp.Body.Close() //Close this on every loop since we can't defer it
	}
	fmt.Println("Exited loop")
}
