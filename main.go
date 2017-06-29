package main

import (
	"net"

	"github.com/NyaaPantsu/nyaa/util/log"
	"github.com/NyaaPantsu/nyaa/util/signals"
	"github.com/NyaaPantsu/scrapers/config"
	"github.com/NyaaPantsu/scrapers/metainfoFetcher"
	"github.com/NyaaPantsu/scrapers/scraper"
	"github.com/NyaaPantsu/scrapers/stats"
	"github.com/jinzhu/configor"
)

// RunMetainfoFetcher runs the database filesize fetcher main loop
func RunMetainfoFetcher(conf *config.Config) {
	fetcher, err := metainfoFetcher.New(conf.MetainfoFetcher)
	if err != nil {
		log.Fatalf("failed to start fetcher, %s", err)
		return
	}

	signals.OnInterrupt(func() {
		fetcher.Close()
	})
	fetcher.RunAsync()
	fetcher.Wait()
}

// CreateScraperSocket creates a UDP Scraper socket
func CreateScraperSocket(conf *config.Config) (net.PacketConn, error) {
	if conf.I2P != nil {
		log.Fatal("i2p udp scraper not supported")
	}
	var laddr *net.UDPAddr
	laddr, err := net.ResolveUDPAddr("udp", conf.Stats.Addr)
	if err != nil {
		return nil, err
	}
	return net.ListenUDP("udp", laddr)
}

// RunScraper runs tracker scraper mainloop
func RunStats(conf *config.Config) {

	// bind to network
	pc, err := CreateScraperSocket(conf)
	if err != nil {
		log.Fatalf("failed to bind udp socket for scraper: %s", err)
	}
	// configure tracker scraperv
	var scraper *stats.Scraper
	scraper, err = stats.New(conf.Stats)
	if err != nil {
		pc.Close()
		log.Fatalf("failed to configure scraper: %s", err)
	}

	workers := conf.Stats.Workers
	if workers < 1 {
		workers = 1
	}

	signals.OnInterrupt(func() {
		pc.Close()
		scraper.Close()
	})
	// run udp scraper worker
	for workers > 0 {
		log.Infof("starting up worker %d", workers)
		go scraper.RunWorker(pc)
		workers--
	}
	// run scraper
	go scraper.Run()
	scraper.Wait()
}

func RunScraper(conf *config.Config) {
	go scraper.New(conf.Scraper)
}

func main() {
	configor.Load(&config.Conf, "config.yml")
	RunScraper(config.Conf)
	RunStats(config.Conf)
	RunMetainfoFetcher(config.Conf)

}
