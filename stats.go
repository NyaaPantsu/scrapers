package main

import (
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"github.com/Stephen304/goscrape"
	"github.com/anacrolix/torrent"

	//	"github.com/anacrolix/torrent/metainfo"
	"regexp"
)

type Stats struct {
	Btih      string
	Seeders   int
	Leechers  int
	Completed int
}

type TStruct struct {
	Peers    Stats
	Trackers []string
	//Files    []metainfo.FileInfo
	Files  []*torrent.File
	Magnet string
}

var validChar = regexp.MustCompile(`[a-zA-Z0-9_\-\~]`)

//byteToStr : Function to encode any non-tolerated characters in a tracker request to hex
func byteToStr(arr []byte) (str string) {
	for _, b := range arr {
		c := string(b)
		if !validChar.MatchString(c) {
			dst := make([]byte, hex.EncodedLen(len(c)))
			hex.Encode(dst, []byte(c))
			c = string(dst)
		}
		str += c
	}
	return
}

func udpScrape(trackers []string, hash string, chFin chan<- bool, torr *TStruct) {
	udpscrape := goscrape.NewBulk(trackers)
	results := udpscrape.ScrapeBulk([]string{hash})
	if results[0].Btih != "0" {
		torr.Peers = Stats(results[0])
	} else {
		fmt.Println("Bad results: ", results[0])
		udpScrape(trackers, hash, chFin, torr)
	}
	fmt.Println("fin")
	chFin <- true
}

func fileScrape(client *torrent.Client, torr *TStruct, chFin chan<- bool) {
	t, _ := client.AddMagnet(torr.Magnet)
	<-t.GotInfo()
	infoHash := t.InfoHash()
	dst := make([]byte, hex.EncodedLen(len(t.InfoHash())))
	hex.Encode(dst, infoHash[:])
	var UDP []string
	var HTTP []string
	torr.Trackers = t.Metainfo().AnnounceList[0]
	for _, tracker := range torr.Trackers {
		if strings.HasPrefix(tracker, "http") {
			HTTP = append(HTTP, tracker)
		} else if strings.HasPrefix(tracker, "udp") {
			UDP = append(UDP, tracker)
		}
	}
	// UDP = append(UDP, "udp://tracker.coppersurfer.tk:6969/announce")
	if len(UDP) != 0 {
		go udpScrape(UDP, string(dst), chFin, torr)
	}
	//	metaInfo := t.Info()
	//	torr.Files = t.UpvertedFiles()
	torr.Files = t.Files()
	t.Drop()
	fmt.Println("fin2")
	chFin <- true
}

//I'm sure there's a less sloppy way to do this, but let's call this an "alpha" version
func injectStats(t *Torrent, torr *TStruct) {
	fmt.Println("Injecting stats!")
	t.Seeders = torr.Peers.Seeders
	t.Leechers = torr.Peers.Leechers
	t.Completed = torr.Peers.Completed
	fmt.Println("Printing files for", t.Magnet)
	fmt.Println(torr.Files)
	//Current filelist struct uses a string array, not sure how to convert that
	//t.FileList = torr.Files
}

func grabEverything(client *torrent.Client, torr TStruct, t Torrent, chOut chan<- Torrent) {
	chFin := make(chan bool)
	go fileScrape(client, &torr, chFin)
	// for i := 0; i < 2; {
	// 	select {
	// 	case <-chFin:
	// 		i++
	// 	}
	// }
	// injectStats(&t, &torr)
	chOut <- t
}

func statWorker(chIn <-chan Torrent, chOut chan<- Torrent) {
	client, _ := torrent.NewClient(nil)
	for t := range chIn {
		fmt.Println("stats")
		torr := TStruct{}
		torr.Trackers = []string{
			"udp://tracker.uw0.xyz:6969/announce",
			"udp://tracker.coppersurfer.tk:6969",
			"udp://tracker.zer0day.to:1337/announce",
			"udp://tracker.leechers-paradise.org:6969",
			"udp://explodie.org:6969",
			"udp://tracker.opentrackr.org:1337",
			"udp://tracker.internetwarriors.net:1337/announce",
			"http://mgtracker.org:6969/announce",
			"udp://ipv6.leechers-paradise.org:6969/announce",
			"http://nyaa.tracker.wf:7777/announce",
			"http://sukebei.tracker.wf:7777/announce",
			"http://tracker.anirena.com:80/announce",
			"http://anidex.moe:6969/announce",
		}
		torr.Magnet = "magnet:?xt=urn:btih:" + t.Hash + "&dn=" + url.PathEscape(t.Title)
		for _, k := range torr.Trackers {
			torr.Magnet = torr.Magnet + "&tr=" + k
		}
		// fmt.Println(torr.Magnet)
		go grabEverything(client, torr, t, chOut)
	}
}
