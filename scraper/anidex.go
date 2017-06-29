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
func getAnidexMax(b []byte) int {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
	if err != nil {
		fmt.Println("Could not get offset max")
		return 0
	}
	str := doc.Find("p.text-center:nth-child(2)").Text()
	fmt.Println(str)
	str = str[(len(str)-15):(len(str)-13)] + str[(len(str)-12):(len(str)-9)]
	max, err := strconv.Atoi(str)
	if err != nil {
		fmt.Println("Could not parse offset max")
		return 0
	}
	return max
}

//anidexChild crawls anidex torrent pages for relevant info
func anidexChild(chTorrent chan<- Torrent, chPageID chan string) {
	adultCategories := map[string][]int{
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
	categories := map[string][]int{
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
		"Other":        []int{7, 1},
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
		//defer resp.Body.Close()
		b, _ := ioutil.ReadAll(resp.Body)
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
		if err != nil {
			fmt.Println("Could not read file")
			resp.Body.Close()
			continue
		}

		info.Source = "https://anidex.info/?page=torrent&id=" + pageID
		//info.Title = doc.Find("div.panel:nth-child(1) > div:nth-child(1) > h3:nth-child(1) > span:nth-child(1)").Text()
		info.Title = doc.Find("div.panel:nth-child(1) > div:nth-child(1) > h3:nth-child(1)").Text()
		if len(info.Title) > 3 {
			info.Title = strings.TrimSpace(info.Title[:len(info.Title)-3])
		}
		info.Hash = strings.ToUpper(doc.Find("#edit_torrent_form > div:nth-child(1) > div:nth-child(2) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(7) > td:nth-child(2)").Text())
		info.Hash = strings.TrimSpace(info.Hash)
		info.Magnet, _ = doc.Find("a.btn-default:nth-child(2)").Attr("href")
		info.Magnet = strings.TrimSpace(info.Magnet)
		//userID, _ := doc.Find("tr.edit:nth-child(1) > td:nth-child(2)").Attr("id")
		//info.UploaderID, _ = strconv.Atoi(strings.TrimSpace(userID))
		info.Uploader = strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(1) > td:nth-child(2)").Text())
		//fmt.Println(info.Uploader)
		info.Language = strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(2) > td:nth-child(2)").Text())
		info.Label = strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(4) > td:nth-child(2)").Text())
		//fmt.Println(info.Language)
		//checkAdult := strings.TrimSpace(doc.Find("div.row:nth-child(1) > div:nth-child(1) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(4) > td:nth-child(2) > span:nth-child(1) > span:nth-child(1)").Text())
		checkAdult := strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(4) > td:nth-child(2) > span:nth-child(1)").Text())
		if checkAdult == "Hentai" {
			info.Adult = true
		} else {
			info.Adult = false
		}

		//Anidex has a much smaller category catalog so we can just switch every option
		//category := strings.TrimSpace(doc.Find("div.row:nth-child(1) > div:nth-child(1) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(3) > td:nth-child(2) > span:nth-child(1) > div:nth-child(1)").Text())
		category := strings.TrimSpace(doc.Find("table.edit:nth-child(1) > tbody:nth-child(2) > tr:nth-child(3) > td:nth-child(2) > div:nth-child(1)").Text())

		if info.Adult {
			copy(info.Category[:2], adultCategories[category][:2])
		} else {
			copy(info.Category[:2], categories[category][:2])
		}

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

		fmt.Println(category, info.Category, info.Language, pageID)

		fileSize := strings.Split(strings.TrimSpace(doc.Find("#edit_torrent_form > div:nth-child(1) > div:nth-child(2) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(2) > td:nth-child(2)").Text()), " ")
		fileFloat, err := strconv.ParseFloat(fileSize[0], 32)
		if err != nil {
			fmt.Println(err)
		}
		for i := 0; i < fileSizes[fileSize[1]]; i++ {
			fileFloat *= 1024
		}
		info.FileSize = int(fileFloat)
		info.Date = strings.TrimSpace(doc.Find("#edit_torrent_form > div:nth-child(1) > div:nth-child(2) > table:nth-child(1) > tbody:nth-child(2) > tr:nth-child(1) > td:nth-child(2)").Text())
		info.Seeders, _ = strconv.Atoi(strings.TrimSpace(doc.Find("td.text-success:nth-child(2)").Text()))
		info.Leechers, _ = strconv.Atoi(strings.TrimSpace(doc.Find("td.text-danger:nth-child(2)").Text()))
		info.Completed, _ = strconv.Atoi(strings.TrimSpace(doc.Find(".text-info").Text()))
		//info.Description = strings.TrimSpace(doc.Find(".panel-body > span:nth-child(1)").Text())
		descriptionHTML, err := doc.Find(".panel-body > span:nth-child(1)").Html()
		if err != nil {
			fmt.Println("Error getting description HTML", err)
		}

		info.Description = bluemonday.UGCPolicy().Sanitize(descriptionHTML)
		if len(info.Description) < 1 {
			info.Description = "No description found"
		}

		if len(info.Title) < 1 || len(info.Hash) == 0 || len(info.Magnet) == 0 {
			fmt.Println("Failed to read", pageID, "sending back into queue")
			chPageID <- pageID
			resp.Body.Close()
			continue
		}
		for len(chTorrent) == cap(chTorrent) {
			fmt.Println("Torrent channel full, sleeping 3 seconds")
			time.Sleep(time.Millisecond * 3000)
		}

		chTorrent <- info
		//fmt.Println("Success on", pageID, "in", time.Since(start), "Size of anidex channel:", len(chPageID))
		//fmt.Println(info.Adult, info.Category, info.Subcategory, category, "\n")
		resp.Body.Close()
	}
	fmt.Println("Exited loop")
}
