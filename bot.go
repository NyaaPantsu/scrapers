package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
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

func (t Torrent) String() string {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("Source:\t%#v\n", t.Source))
	buffer.WriteString(fmt.Sprintf("Title:\t%#v\n", t.Title))
	buffer.WriteString(fmt.Sprintf("Uploader:\t%#v\n", t.Uploader))
	buffer.WriteString(fmt.Sprintf("UserID:\t%#v\n", t.UploaderID))
	buffer.WriteString(fmt.Sprintf("Category:\t%#v\n", t.Category))
	buffer.WriteString(fmt.Sprintf("Language:\t%#v\n", t.Language))
	buffer.WriteString(fmt.Sprintf("Description Length:\t%#v\n", len(t.Description)))
	buffer.WriteString(fmt.Sprintf("Magnet:\t%#v\n", t.Magnet))
	buffer.WriteString(fmt.Sprintf("Hash:\t%#v\n", t.Hash))
	buffer.WriteString(fmt.Sprintf("FileSize:\t%#v\n", t.FileSize))
	buffer.WriteString(fmt.Sprintf("Date:\t%#v\n", t.Date))
	buffer.WriteString(fmt.Sprintf("Seeders:\t%#v\n", t.Seeders))
	buffer.WriteString(fmt.Sprintf("Leechers:\t%#v\n", t.Leechers))
	buffer.WriteString(fmt.Sprintf("Completed:\t%#v\n", t.Completed))
	buffer.WriteString(fmt.Sprintf("Label:\t%#v\n", t.Label))
	buffer.WriteString(fmt.Sprintf("Adult?:\t%#v\n", t.Adult))
	return buffer.String()
}

/*
//Leftovers from the original MainScrape method.  Unsure if these are still in use.
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
*/

//Timer times stuff
func timer(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

func main() {

	//Channels
	chFinished := make(chan bool)            //So we know when to exit
	chTorrent := make(chan Torrent, 500)     //Where our compiled torrent info goes
	chTorrAndStat := make(chan Torrent, 500) //"overload" channel to compile stats into torrents before insertion
	chURLCount := make(chan int)             //To make sure we actually scraped every URL we found
	chNyaaURL := make(chan string, 1000)     //Channel to send nyaa.si urls to, consumed in nyaa.go:nyaaChild
	chAnidexURL := make(chan string, 1000)   //Channel to send anidex urls to, consumed in anidex.go:anidexChild
	chHTML := make(chan HTMLBlob, 2000)      //Channel to send HTML binary blobs, consumed in htmlparse.go:parsePageMain
	chInsertCount := make(chan int)          //Channel to track how many new torrents were inserted
	chFoundCount := make(chan int)           //Channel to track how many hashes were already in the DB

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

	go parsePageMain(chHTML, chNyaaURL, chAnidexURL, chFinished, chURLCount)
	//Start crawling
	for _, url := range seedUrls {
		if strings.Contains(url, "anidex") {
			go anidexParent(numAnidexOffset, numMaxPages, chHTML)
		} else if strings.Contains(url, "sukebei.nyaa.si") {
			go nyaaParent(numNyaaOffset, numMaxPages, "https://sukebei.nyaa.si", chHTML)

		} else if strings.Contains(url, "nyaa.si") {
			go nyaaParent(numNyaaOffset, numMaxPages, "https://nyaa.si", chHTML)

		}
	}

	//Start workers
	for i := 0; i < numWorkers; i++ {
		go anidexChild(chTorrent, chAnidexURL)
		go nyaaChild(chTorrent, chNyaaURL)
	}

	//SQl/Stat scrapers
	go statWorker(chTorrent, chTorrAndStat)
	go sqlWorker(chTorrAndStat, chFinished, chInsertCount, chFoundCount)
	go sqlWorker(chTorrAndStat, chFinished, chInsertCount, chFoundCount)
	go sqlWorker(chTorrAndStat, chFinished, chInsertCount, chFoundCount)

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
