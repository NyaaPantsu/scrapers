package scraper

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

const ()

/*
	### Nyaa.si adult categories ###
	Art - Anime
	Art - Doujinshi
	Art - Games
	Art - Manga
	Art - Pictures
	Real Life - Photobooks and Pictures
	Real Life - Videos

	### Nyaa.si categories ###
	Anime - Anime Music Video
	Anime - English-translated
	Anime - Non-English-translated
	Anime - Raw
	Audio - Lossless
	Audio - Lossy
	Literature - English-translated
	Literature - Non-English-translated
	Literature - Raw
	Live Action - English-translated
	Live Action - Idol/Promotional Video
	Live Action - Non-English-translated
	Live Action - Raw
	Pictures - Graphics
	Pictures - Photos
	Software - Applications
	Software - Games
*/

//nyaaParent crawls nyaa.si main pages to get torrent IDs
//startOffset is the page to start scraping on
//pageMax is the maximum number of pages we want to scrape
func nyaaParent(startOffset, pageMax int, baseURL string, chHTML chan<- []byte) {
	nyaaPage := startOffset

	//I'm pretty sure there's a way to do this without an iterator
	for i := 0; i < pageMax; i++ {
		nyaaURL := baseURL + "/?p=" + strconv.Itoa(nyaaPage)
		nyaaPage++
		response, err := http.Get(nyaaURL)
		if err != nil {
			fmt.Println("ERROR: Failed to crawl\"" + nyaaURL + "\"")
			response.Body.Close()
			break
		}
		b, _ := ioutil.ReadAll(response.Body)
		response.Body.Close()
		//TODO: This really should be its own function
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
		if err != nil {
			fmt.Println("Errored checking for Nyaa.si 404", err)
			return
		}
		is404 := doc.Find("div.container:nth-child(2) > h1:nth-child(1)").Text()
		if is404 == "404 Not Found" {
			fmt.Println("Found 404, exiting crawler")
			return
		}
		chHTML <- b
	}
}

//nyaaChild leverages their API for torrent info
func nyaaChild(chTorrent chan<- Torrent, chNyaaURL chan string) {
	nyaaCategoryMap := map[string][]int{
		"Art - Anime":                          []int{1, 1},
		"Art - Doujinshi":                      []int{1, 2},
		"Art - Games":                          []int{1, 3},
		"Art - Manga":                          []int{1, 4},
		"Art - Pictures":                       []int{1, 5},
		"Real Life - Photobooks and Pictures":  []int{2, 1},
		"Real Life - Videos":                   []int{2, 2},
		"Anime - Anime Music Video":            []int{3, 12},
		"Anime - English-translated":           []int{3, 5},
		"Anime - Non-English-translated":       []int{3, 13},
		"Anime - Raw":                          []int{3, 6},
		"Audio - Lossless":                     []int{2, 3},
		"Audio - Lossy":                        []int{2, 4},
		"Literature - English-translated":      []int{4, 7},
		"Literature - Non-English-translated":  []int{4, 14},
		"Literature - Raw":                     []int{4, 8},
		"Live Action - English-translated":     []int{5, 9},
		"Live Action - Idol/Promotional Video": []int{5, 10},
		"Live Action - Non-English-translated": []int{5, 18},
		"Live Action - Raw":                    []int{5, 11},
		"Pictures - Graphics":                  []int{6, 15},
		"Pictures - Photos":                    []int{6, 16},
		"Software - Applications":              []int{1, 1},
		"Software - Games":                     []int{1, 2},
	}
	for url := range chNyaaURL {
		n, err := nyaaAPI(url)
		if err != nil {
			fmt.Println(err, "on page", url)
		}

		info := nyaaBuildStruct(n, url, nyaaCategoryMap)

		if len(info.Description) < 1 {
			info.Description = "No description found"
		}

		if len(info.Uploader) < 2 {
			info.Uploader = "Anonymous"
		}

		//If any key info was missed, send it back and rescrape it
		if len(info.Title) < 2 || len(info.Hash) == 0 || len(info.Magnet) == 0 {
			fmt.Println("Nyaa scrape failed, missing title, hash, or magnet link." +
				"Pushing to end of queue")
			chNyaaURL <- url
			continue
		}

		for len(chTorrent) == cap(chTorrent) {
			fmt.Println("Torrent channel full, sleeping 3 seconds")
			time.Sleep(time.Millisecond * 3000)
		}

		chTorrent <- info
	}
}

func nyaaBuildStruct(n nyaaJSON, url string, categories map[string][]int) (info Torrent) {
	info.Source = url
	info.Title = n.Name
	info.Uploader = n.Uploader
	info.UploaderID = 0
	//info.Language = //Doesn't exist on Nyaa.si
	info.Description = n.Description
	info.Magnet = n.Magnet
	info.Hash = strings.TrimSpace(n.HashHex)
	info.Hash = strings.ToUpper(info.Hash)
	info.FileSize = n.FileSize
	info.Date = n.CreatedOn
	info.Seeders = n.Stats.Seeders
	info.Leechers = n.Stats.Leechers
	info.Completed = n.Stats.Downloads

	if strings.Contains(info.Source, "subekei") {
		info.Adult = true
	} else {
		info.Adult = false
	}

	//Join the api (sub)category with - to map it easier
	category := n.MainCategory + " - " + n.SubCategory
	copy(info.Category[:2], categories[category][:2])

	//Convert the api markdown description to sanitized HTML
	unsafe := blackfriday.MarkdownCommon([]byte(n.Description))
	b := bluemonday.UGCPolicy().SanitizeBytes(unsafe)
	info.Description = string(b)
	return
}
