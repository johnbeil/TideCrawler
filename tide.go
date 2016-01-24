// Copyright (c) 2016 John Beil.
// Use of this source code is governed by the MIT License.
// The MIT license can be found in the LICENSE file.

// TideCrawler 0.1
// Obtains annual tide forecasts for NOAA Station 9414275
// Parses each tide prediction
// Saves observation to database - TO DO

package main

import (
	"bytes"
	"database/sql"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/net/html/charset"
)

// Config stores database credentials
type Config struct {
	DatabaseURL      string
	DatabaseUser     string
	DatabasePassword string
	DatabaseName     string
}

// TideData stores a series of tide predictions
type TideData struct {
	Tides []Tide `xml:"data>item"`
}

// Tide stores a single tide prediction
type Tide struct {
	// XMLName xml.Name `xml"data`
	Date         string  `xml:"date"`
	Day          string  `xml:"day"`
	Time         string  `xml:"time"`
	PredictionFt float64 `xml:"predictions_in_ft"`
	PredictionCm float64 `xml:"predictions_in_cm"`
	HighLow      string  `xml:"highlow"`
	DateTime     time.Time
}

// NOAA URL for Annual Tide XML
const noaaURL = "http://tidesandcurrents.noaa.gov/noaatidepredictions/NOAATidesFacade.jsp?datatype=Annual+XML&Stationid=9414275&text=datafiles"

// Timezone to use for all time formatting
var timezone = "PST"

// Global variable for database
var db *sql.DB

// Fetches Annual tide data and processes XML data
func main() {
	// Start tide crawler
	fmt.Println("Starting TideCrawler...")

	// Load configuration
	config := Config{}
	loadConfig(&config)

	// Initialize tides to hold annual tide predictions
	var tides TideData

	// Load database
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		config.DatabaseUser, config.DatabasePassword, config.DatabaseName)
	var err error
	db, err = sql.Open("postgres", dbinfo)
	if err != nil {
		log.Fatal("Error opening database connection:", err)
	}
	defer db.Close()

	// Check database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error: Could not establish connection with the database.", err)
	}

	// Fetch annual data and store as byte b
	b := getDataFromURL(noaaURL)
	// fmt.Println("b is:", reflect.TypeOf(b))

	// Convert b from []uint8 to *bytes.Buffer
	c := bytes.NewBuffer(b)
	// fmt.Println("c is:", reflect.TypeOf(c))

	// Use decoder to unmarshal the XML since NOAA data is in ISO-8859-1 and
	// Unmarshal only reads UTF-8
	decoder := xml.NewDecoder(c)
	decoder.CharsetReader = charset.NewReaderLabel
	if err := decoder.Decode(&tides); err != nil {
		log.Fatal("decoder error:", err)
	}

	// Drop the existing tidedata table
	dropTable()

	// Create a new empty tidedata table
	createTable()

	// Iterate over each Tide in Tides and save in database
	for _, d := range tides.Tides {
		d.DateTime = formatTime(d)
		saveTide(d)
		// fmt.Printf("\t%s\n", d.DateTime)
		// fmt.Println(d)
	}
	fmt.Println("Success. Number of items saved to tidedata table is:", len(tides.Tides))
	// fmt.Println(tides.TideData)

	fmt.Println("Shutting down TideCrawler...")
}

// Returns formatted tide data
func (t Tide) String() string {
	// stime := t.DateTime.UTC().Format(time.UnixDate)
	return t.Date + " " + t.Day + " " + t.Time + " " + t.HighLow + " " + t.DateTime.UTC().Format(time.UnixDate)
}

// Given Tide struct, returns formatted date time
func formatTime(d Tide) time.Time {
	// Concatenate tide prediction data into string
	rawtime := d.Date + " " + d.Time + " " + timezone

	// Parse time given concatenated rawtime
	t, err := time.Parse("2006/01/02 3:04 PM PST", rawtime)
	if err != nil {
		log.Fatal("error processing rawtime:", err)
	}
	// set timezone for datetime and update time variable t
	// loc, err := time.LoadLocation("America/Los_Angeles")
	// if err != nil {
	// 	log.Fatal("error processing location", err)
	// }
	// t = t.In(loc)
	return t
}

// Given URL, returns raw data
func getDataFromURL(url string) (body []byte) {
	fmt.Println("Fetching data...")
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("Error fetching data:", err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("ioutil error reading resp.Body:", err)
	}
	if resp.StatusCode == 200 {
		fmt.Println("Fetch successful. Processing data...")
	} else {
		fmt.Println("Fetch returned unanticipated HTTP code:", resp.Status)
	}
	return
}

// Loads database credentials from environment variables
func loadConfig(config *Config) {
	config.DatabaseUser = os.Getenv("DATABASEUSER")
	config.DatabasePassword = os.Getenv("DATABASEPASSWORD")
	config.DatabaseURL = os.Getenv("DATABASEURL")
	config.DatabaseName = os.Getenv("DATABASENAME")
	fmt.Println("Config is:", config)
}

// savePrediction inserts a tide struct into the database
func saveTide(t Tide) {
	_, err := db.Exec("INSERT INTO tidedata(datetime, date, day, time, predictionft, predictioncm, highlow) VALUES($1, $2, $3, $4, $5, $6, $7)", t.DateTime, t.Date, t.Day, t.Time, t.PredictionFt, t.PredictionCm, t.HighLow)
	if err != nil {
		log.Fatal("Error saving tide:", err)
	}
}

// dropTable drops an existing table from the database
func dropTable() {
	_, err := db.Exec("DROP TABLE tidedata")
	if err != nil {
		log.Fatal("Error dropping table tidedata:", err)
	} else {
		fmt.Println("Dropped existing table tidedata...")
	}

}

// createTable creates a new tidedata table in the database
func createTable() {
	_, err := db.Exec("CREATE TABLE tidedata(uid serial NOT NULL, datetime timestamp, date varchar(16), day varchar (16), time varchar(16), predictionft real, predictioncm integer, highlow varchar (16));")
	if err != nil {
		log.Fatal("Error creating table tidedata:", err)
	} else {
		fmt.Println("Created new table tidedata...")
	}
}
