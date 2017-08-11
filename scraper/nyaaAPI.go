package scraper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

const (
	nyaaUser = ""
	nyaaPW   = ""
)

type nyaaJSON struct {
	CreatedOn   string `json:"creation_date"`
	Description string `json:"description"`
	//Files          []string `json:"files"`
	FileSize       int    `json:"filesize"`
	HashB32        string `json:"hash_b32"`
	HashHex        string `json:"hash_hex"`
	ID             int    `json:"id"`
	Info           string `json:"information"`
	Complete       bool   `json:"is_complete"`
	Remake         bool   `json:"is_remake"`
	Trusted        bool   `json:"is_trusted"`
	Magnet         string `json:"magnet"`
	MainCategory   string `json:"main_category"`
	MainCategoryID int    `json:"main_category_id"`
	Name           string `json:"name"`
	Stats          Stats  `json:"stats"`
	SubCategory    string `json:"sub_category"`
	SubCategoryID  int    `json:"sub_category_id"`
	Uploader       string `json:"submitter"`
	URL            string `json:"url"`
}

//Stats is an internal struct in the NyaaJSON struct
type Stats struct {
	Downloads int `json:"downloads"`
	Leechers  int `json:"leechers"`
	Seeders   int `json:"seeders"`
}

func nyaaAPI(url string) (n nyaaJSON, err error) {
	apiInfo := `/api/info/`
	page := strings.Split(url, "/view/")
	req, err := http.NewRequest("GET", page[0]+apiInfo+page[1], nil)
	if err != nil {
		return
	}

	//Set auth credentials
	req.SetBasicAuth(nyaaUser, nyaaPW)
	resp, err := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return
	}

	//Read that garbage, then unwrap it into a usable struct
	b, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal(b, &n)
	return
}
