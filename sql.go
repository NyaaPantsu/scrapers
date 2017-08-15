package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 9998
	user     = "nyaapantsu"
	password = "nyaapantsu"
	dbname   = "nyaapantsu"
	pwChars  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

//sqlHashExists returns a boolean on whether or not the hash is already in the table
func sqlHashExists(db *sql.DB, hash, table string) bool {
	sqlTorrentQuery := `SELECT torrent_hash FROM ` + table + ` WHERE torrent_hash=$1;`
	var torrentHash string
	row := db.QueryRow(sqlTorrentQuery, hash)
	switch err := row.Scan(&torrentHash); err {
	case sql.ErrNoRows:
		fmt.Println("No rows returned, attempting insert")
		return false
	case nil:
		fmt.Println("Found", torrentHash, "skipping")
		return true
	default:
		fmt.Println(err)
		return false
	}
}

//sqlTorrentInsert does what it says on the tin
func sqlTorrentInsert(db *sql.DB, torrent Torrent, table string) {
	sqlTorrentInsert := `INSERT INTO ` + table + ` (torrent_name, torrent_hash,
		category, sub_category, status, date, uploader, downloads, stardom,
		filesize, description, hidden)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := db.Exec(sqlTorrentInsert, torrent.Title, torrent.Hash,
		torrent.Category[0], torrent.Category[1], 0, torrent.Date,
		torrent.UploaderID, 0, 0, torrent.FileSize, torrent.Description, 0)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Inserted", torrent.Hash, "into DB!")
}

func sqlStatsInsert(db *sql.DB, torrent Torrent, table string) {
	sqlUserQuery := `SELECT torrent_id FROM public` + table + ` WHERE torrent_hash=$1;`
	row := db.QueryRow(sqlUserQuery, torrent.Hash)
	var torrentID string
	switch err := row.Scan(&torrentID); err {
	case sql.ErrNoRows:
		fmt.Println("torrent not found skipping")
		return
	case nil:
		fmt.Println("Torrent Found inserting stats")
	default:
		fmt.Println(err)
		return

	}
	sqlStatsInsert := `INSERT INTO` + table + `(torrent_id, seeders, leechers, completed, last_scrape) values($1, $2, $3, $4, $5)`
	_, err := db.Exec(sqlStatsInsert, torrentID, torrent.Seeders, torrent.Leechers, torrent.Completed, time.Now())
	if err != nil {
		fmt.Println(err)
	}
}

//sqlUserExists does what it says on the tin
//If the user doesnt exist, attempts an insert
//Returns the userID and userStatus
func sqlUserExists(db *sql.DB, username string) (userID, userStatus int) {
	sqlUserQuery := `SELECT user_id, status FROM public.users WHERE username=$1;`
	row := db.QueryRow(sqlUserQuery, username)
	switch err := row.Scan(&userID, &userStatus); err {
	case sql.ErrNoRows:
		fmt.Println("User not found, attempting insert")
		userID = sqlUserInsert(db, username)
		userStatus = 3
		return
	case nil:
		fmt.Println("User found, checking hash")
		return
	default:
		fmt.Println(err)
		return
	}
}

func sqlUserInsert(db *sql.DB, username string) (userID int) {
	sqlUserInsert := `INSERT INTO public.users (username, password, status,
			created_at, api_token_expiry) VALUES ($1, $2, $3, $4, $5)`

	//Status is hardcoded to 3, as that means it was a scraped user
	_, err := db.Exec(sqlUserInsert, username, RandPassword(), 3, time.Now(), time.Now())
	if err != nil {
		fmt.Println(err)
	}
	//TODO: Rewrite this in a non-stupid fashion
	userID, _ = sqlUserExists(db, username)
	return
}

//RandPassword generates a random password for scraped users not in the DB
//Unabashedly stolen from https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-golang
func RandPassword() string {
	b := make([]byte, 12)
	for i := range b {
		b[i] = pwChars[rand.Intn(len(pwChars))]
	}
	return string(b)
}

func sqlWorker(chTorrent <-chan Torrent, chFinished chan<- bool, chInsertCount chan<- int, chFoundCount chan<- int) {
	defer func() {
		chFinished <- true
	}()

	//Connect to the DB
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected!")

	var table string
	var scrape string
	var userStatus int
	for torrent := range chTorrent {

		//Check if the user exists first
		//Special condition for nyaa.si anonymous uploads
		if torrent.Uploader != "Anonymous" {
			torrent.UploaderID, userStatus = sqlUserExists(db, torrent.Uploader)
		} else {
			torrent.UploaderID = 0
			userStatus = 3 //Anonymous username means we scraped it
		}

		//Determine the table we want
		if torrent.Adult {
			table = "public.sukebei_torrents"
			scrape = "public.sukebei_scrape"
		} else {
			table = "public.torrents"
			scrape = "public.scrape"
		}

		//If our user was scraped and the hash doesnt exist, insert
		if userStatus == 3 && !sqlHashExists(db, torrent.Hash, table) {
			sqlTorrentInsert(db, torrent, table)
		}
		sqlStatsInsert(db, torrent, scrape)
		chInsertCount <- 1 //Tracker to ensure we've attempted every hash we find
	}
	fmt.Println("Exiting SQL worker")
}
