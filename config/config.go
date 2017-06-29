package config

var Conf *Config

type Config struct {
	MetainfoFetcher *MetainfoFetcher
	Stats           *Stats
	Scraper         *Scraper
	DB              *DB
	Models          *Models
	I2P             *I2P
}

type I2P struct {
	Name    string
	Addr    string
	Keyfile string
}

type Stats struct {
	Addr            string `default:"9999"`
	Workers         int    `default"4"`
	IntervalSeconds int64  `default:"3600"`
	Trackers        []ScrapeConfig
}
type ScrapeConfig struct {
	URL             string `default:"udp://tracker.coppersurfer.tk:6969/"`
	Name            string `default:"coppersurfer.tk"`
	IntervalSeconds int64
}
type MetainfoFetcher struct {
	QueueSize            int  `default:"10"`
	Timeout              int  `default:"120"`
	MaxDays              int  `default:"90"`
	BaseFailCooldown     int  `default:"1800"`
	MaxFailCooldown      int  `default:"172800"`
	WakeUpInterval       int  `default:"300"`
	UploadRateLimitKiB   int  `default:"1024"`
	DownloadRateLimitKiB int  `default:"1024"`
	FetchNewTorrentsOnly bool `default:"true"`
}

type Models struct {
	LastOldTorrentID       uint   `default:"923000"`
	TorrentsTableName      string `default:"torrents"`
	ReportsTableName       string `default:"torrent_reports"`
	CommentsTableName      string `default:"comments"`
	UploadsOldTableName    string `default:"user_uploads_old"`
	FilesTableName         string `default:"files"`
	NotificationsTableName string `default:"notifications"`
	ActivityTableName      string `defaukt:"activities"`
	ScrapeTableName        string `default:"scrape"`
}

type Scraper struct {
	User          string `required:"true"`
	Pass          string `required: "true"`
	Workers       int    `default:"4"`
	Anidex_Offset int    `default:"0"`
	Nyaasi_Offset int    `default:"0"`
	NumWorkers    int    `default:"4"`
	MaxPages      int    `default:"1000"`
	Nyaasi        bool   `default:"true"`
	Anidex        bool   `default:"true"`
}
type DB struct {
	Name     string `default:"nyaapantsu"`
	User     string `default:"nyaapantsu"`
	Password string `default:"nyaapantsu"`
	Port     uint   `default:"9998"`
}
