package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"time"
)

const (
	host     = "localhost"
	port     = 9998
	user     = "nyaapantsu"
	password = ""
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

func sqlTorrentInsert(db *sql.DB, torrent Torrent, table string) {
	sqlTorrentInsert := `INSERT INTO ` + table + ` (torrent_name, torrent_hash, category, sub_category,` +
		`status, date, uploader, downloads, stardom, filesize, description) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := db.Exec(sqlTorrentInsert, torrent.Title, torrent.Hash, torrent.Category[0], torrent.Category[1],
		0, torrent.Date, torrent.UploaderID, 0, 0, torrent.FileSize, torrent.Description)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Inserted", torrent.Hash, "into DB!")
}

func sqlUserExists(db *sql.DB, username string) (userID int) {
	sqlUserQuery := `SELECT user_id FROM public.users WHERE username=$1;`
	row := db.QueryRow(sqlUserQuery, username)
	switch err := row.Scan(&userID); err {
	case sql.ErrNoRows:
		fmt.Println("User not found, attempting insert")
		return 0
	case nil:
		fmt.Println("User found, checking hash")
		return userID
	default:
		fmt.Println(err)
		return userID
	}
}

func sqlUserInsert(db *sql.DB, username string) (userID int) {
	sqlUserInsert := `INSERT INTO public.users (username, password, status, created_at, api_token_expiry)
		VALUES ($1, $2, $3, $4, $5)`
	//Status is hardcoded to 3, as that means it was a scraped user
	_, err := db.Exec(sqlUserInsert, username, RandPassword(), 3, time.Now(), time.Now())
	if err != nil {
		fmt.Println(err)
	}
	return sqlUserExists(db, username)
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
	for torrent := range chTorrent {

		//Check if the user exists first
		//Special condition for nyaa.si anonymous uploads
		if torrent.Uploader != "Anonymous" {
			if userID := sqlUserExists(db, torrent.Uploader); userID == 0 {
				torrent.UploaderID = sqlUserInsert(db, torrent.Uploader)
			} else {
				torrent.UploaderID = userID
			}
		} else {
			torrent.UploaderID = 0
		}

		//Determine the table we want
		if torrent.Adult {
			table = "public.sukebei_torrents"
		} else {
			table = "public.torrents"
		}

		if !sqlHashExists(db, torrent.Hash, table) {
			//If the hash doesnt exist, run an insert
			sqlTorrentInsert(db, torrent, table)
			chInsertCount <- 1
		} else {
			//Otherwise send an int to count it towards our total scrape
			chFoundCount <- 1
		}
	}
	fmt.Println("Exiting SQL worker")
}
