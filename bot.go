package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	_ "github.com/lib/pq"
	"golang.org/x/net/html"
)

/*
	1_1 - Art - Anime
	1_2 - Art - Doujinshi
	1_3 - Art - Games
	1_4 - Art - Manga
	1_5 - Art - Pictures
	2_1 - Real Life - Photobooks / Pictures
	2_2 - Real Life - V

	3_5 - Anime - English-translated
	3_13 - Anime - Non-English-translated
	3_6 - Anime - Raw
	2_3 - Audio - Lossless
	2_4 - Audio - Lossy
	4_7 - Literature - English-translated
	4_14 - Literature - Non-English-translated
	4_8 - Literature - Raw
	5_9 - Live Action - English-translated
	5_10 - Live Action - Idol/Promotional Video
	5_18 - Live Action - Non-English-translated
	5_11 - Live Action - Raw
	6_15 - Pictures - Graphics
	6_16 - Pictures - Photos
	1_1 - Software - Applications
	1_2 - Software - Games
*/

//Torrent is a struct to contain relevant info from scraping the sites
type Torrent struct {
	Source      string //Source URL
	Title       string //Torrent name
	Uploader    string
	UploaderID  int
	Category    [2]int //Mapped as best as it can be to the above list
	Language    string
	Description string
	Magnet      string
	Hash        string
	FileSize    int    //In bytes
	Date        string //In UTC
	Seeders     int
	Leechers    int
	Completed   int
	FileList    []string //Not used yet
	Label       string   //Not used yet
	Adult       bool     //Whether it belongs to sukebei or not
}

//getHrefMain scrapes the main index listing of the page for links to torrent descriptions
func getHrefMain(tok html.Token) (ok bool, href string) {
	for _, a := range tok.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}
	return //implicit signature return
}

//crawlMain gets the list of torrent page links from each site
func crawlMain(baseURL string, maxPages, startOffset int, chNyaaURL chan<- string, chAnidexURL chan<- string, chFinished chan<- bool, chURLCount chan<- int) {

	var tokenizer *html.Tokenizer
	var leave bool
	anidexOffset := startOffset
	maxoffset := anidexOffset + 1 //Set to one for the first loop, should be updated on first run
	maxOffsetSet := false
	nyaaPage := startOffset
	childPageCount := 0

	for childPageCount < maxPages {
		//TODO: D.R.Y. Error checking
		leave = false
		if strings.Contains(baseURL, "anidex") {
			//
			if anidexOffset < maxoffset {
				fmt.Println("Fetching anidex page offset", anidexOffset)
				req, err := http.NewRequest("GET",
					"https://anidex.info/ajax/page.ajax.php?page=torrents&category=0&filename=&lang_id=&r=0&b=0&order_by=upload_timestamp&order=desc&limit=50&offset="+strconv.Itoa(anidexOffset), nil)
				if err != nil {
					fmt.Println(err)
				}
				req.Header.Set("X-Requested-With", "XMLHttpRequest")

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					fmt.Println(err)
				}

				b, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close() //We can just close the body right away since we're not doing anything with the connection afterwards
				if err != nil {
					log.Fatal(err)
				}

				if !maxOffsetSet {
					maxoffset = getAnidexMax(b)
					fmt.Println("---Max offset is", maxoffset)
					maxOffsetSet = true
				}

				anidexOffset += 50
				//fmt.Println("---Offset increase to", anidexOffset)

				tokenizer = html.NewTokenizer(strings.NewReader(string(b)))
				if err != nil {
					fmt.Println("ERROR: Failed to crawl\"" + baseURL + "\"")
					resp.Body.Close()
					break
				}
			} else {
				fmt.Println("------Offset maximum reached, exiting crawler loop-------")
				return
			}
		} else {
			//nyaaURL := baseURL + "/?s=id&o=asc&p=" + strconv.Itoa(nyaaPage)
			nyaaURL := baseURL + "/?p=" + strconv.Itoa(nyaaPage)
			nyaaPage++
			response, err := http.Get(nyaaURL)
			if err != nil {
				fmt.Println("ERROR: Failed to crawl\"" + nyaaURL + "\"")
				response.Body.Close()
				break
			}

			b, _ := ioutil.ReadAll(response.Body)
			tokenizer = html.NewTokenizer(strings.NewReader(string(b)))
			response.Body.Close() //close body when func returns

			//TODO: This really should be its own function
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(b)))
			if err != nil {
				fmt.Println("Errored checking for Nyaa.si 404", err)
				return
			}

			is404 := doc.Find("div.container:nth-child(2) > h1:nth-child(1)").Text()
			if is404 == "404 Not Found" {
				fmt.Println("Found 404, exiting crawler")
				chFinished <- true
				return
			}

		}
		for {
			if leave {
				break
			}
			tokType := tokenizer.Next()
			switch {
			case tokType == html.ErrorToken:
				//EOF
				leave = true
				break
			case tokType == html.StartTagToken:
				tok := tokenizer.Token()
				isAnchor := tok.Data == "a"
				if !isAnchor {
					continue
				}
				ok, url := getHrefMain(tok)

				//If the URL ends in t, it's a nyaa.si torrent quicklink
				//We don't give a shit about those at this point, so skip them
				if !ok || url[len(url)-1:] == "t" {
					continue
				}

				//var names correspond to their respective sites
				nyaaSi := strings.Index(url, "/view") == 0
				anidex := strings.Index(url, "?page=torrent&id=") == 0
				if nyaaSi {
					for len(chNyaaURL) == cap(chAnidexURL) {
						fmt.Println("Nyaa channel full, sleeping 3 seconds")
						time.Sleep(time.Millisecond * 3000)
					}
					chNyaaURL <- baseURL + url
					chURLCount <- 1
					childPageCount++
					continue
				}
				if anidex {
					for len(chAnidexURL) == cap(chAnidexURL) {
						fmt.Println("Anidex channel full, sleeping 3 seconds")
						time.Sleep(time.Millisecond * 3000)
					}
					chAnidexURL <- url[17:]
					chURLCount <- 1
					childPageCount++
					continue
				}
			}
		}
		fmt.Println("--Number of torrent pages collected:", childPageCount)
		fmt.Println("--Size of anidex channel:", len(chAnidexURL))
		fmt.Println("--Size of nyaa channel:", len(chNyaaURL))
	}
	f, err := os.Create(baseURL[8:] + "_endOffset.txt")
	if err != nil {
		panic(err)
	}
	if strings.Contains(baseURL, "anidex") {
		_, err := f.WriteString(strconv.Itoa(anidexOffset))
		if err != nil {
			panic(err)
		}
	} else if strings.Contains(baseURL, "nyaa") {
		_, err := f.WriteString(strconv.Itoa(nyaaPage))
		if err != nil {
			panic(err)
		}
	}
	chFinished <- true
}

//Timer times stuff
func timer(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func main() {

	//Channels
	chFinished := make(chan bool)          //So we know when to exit
	chTorrent := make(chan Torrent, 500)   //Where our compiled torrent info goes
	chURLCount := make(chan int)           //To make sure we actually scraped every URL we found
	chNyaaURL := make(chan string, 1000)   //Channel to send nyaa.si urls to, consumed in nyaaChild
	chAnidexURL := make(chan string, 1000) //Channel to send anidex urls to, consumed in anidexChild
	chInsertCount := make(chan int)        //Channel to track how many new torrents were inserted
	chFoundCount := make(chan int)         //Channel to track how many hashes were already in the DB

	/*
		//Debugging garbage
		//go anidexChild(chTorrent, chAnidexURL)
		//chAnidexURL <- "48882"
		go nyaaChild(chTorrent, chNyaaURL)
		chNyaaURL <- "https://nyaa.si/view/929072"
		fmt.Println("Press any key to continue")
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
	*/

	defer timer(time.Now(), "Execution")
	numWorkers, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	numMaxPages, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println(err)
		return
	}
	numAnidexOffset, err := strconv.Atoi(os.Args[3])
	if err != nil {
		fmt.Println(err)
		return
	}
	numNyaaOffset, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Println(err)
		return
	}
	seedUrls := os.Args[5:]

	//Start crawling
	for _, url := range seedUrls {
		if strings.Contains(url, "anidex") {
			go crawlMain(url, numMaxPages, numAnidexOffset, chNyaaURL, chAnidexURL, chFinished, chURLCount)
		} else if strings.Contains(url, "nyaa") {
			go crawlMain(url, numMaxPages, numNyaaOffset, chNyaaURL, chAnidexURL, chFinished, chURLCount)
		}
	}

	//Start child workers
	for i := 0; i < numWorkers; i++ {
		go anidexChild(chTorrent, chAnidexURL)
		go nyaaChild(chTorrent, chNyaaURL)
		//Buy one get two free!
		go sqlWorker(chTorrent, chFinished, chInsertCount, chFoundCount)
		go sqlWorker(chTorrent, chFinished, chInsertCount, chFoundCount)
		go sqlWorker(chTorrent, chFinished, chInsertCount, chFoundCount)
	}

	leave := false
	urlCount := 0
	insertCount := 0
	foundCount := 0
	for {
		//Only break out once we receive a finished flag and we've attempted every URL we found
		if leave && insertCount+foundCount == urlCount {
			break
		}

		select {
		case n := <-chFoundCount:
			foundCount += n
			fmt.Println("Total pages trawled:", foundCount+insertCount)
		case n := <-chInsertCount:
			insertCount += n
			fmt.Println("Total pages trawled:", foundCount+insertCount)
		case n := <-chURLCount:
			urlCount += n
		case <-chFinished:
			leave = true
		}

	}
	//fmt.Println("Left the loop, press any key to continue")

	//reader := bufio.NewReader(os.Stdin)
	//text, _ := reader.ReadString('\n')

	//fmt.Println(text)
	close(chAnidexURL)
	close(chNyaaURL)

	fmt.Println("Finished crawling")
	fmt.Println("Inserted", insertCount, "against", urlCount, "found URLS:\n")
	close(chTorrent)
}
