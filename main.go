package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/namsral/flag"
)

var (
	db            *sql.DB
	server        *http.Server
	csvFile       string
	csvDelimiter  rune
	csvFieldCount int
	csvHasHeader  bool
	csvHeader     []string
	csvIndicies   []string
)

// trySplit attempts to split the given string by calling
// strings.Split for each separator in the order given.
// It returns the first successful split that occurs.
func trySplit(str string, seps ...string) []string {
	for _, sep := range seps {
		if s := strings.Split(str, sep); len(s) != 1 {
			return s
		}
	}
	return []string{str}
}

func parseEnvironment() {

	// Info message
	log.Println("*** Parsing configuration from environment ***")

	// Define configuration variables
	flagSet := flag.NewFlagSetWithEnvPrefix(os.Args[0], "CSV", 0)
	flagSet.StringVar(&csvFile, "file", "", "path to csv file")
	delimiter := flagSet.String("delimiter", ",", "data separator in the csv file")
	flagSet.IntVar(&csvFieldCount, "field-count", 0, "the number of fields/columns in the csv file")
	flagSet.BoolVar(&csvHasHeader, "has-header", true, "whether or not the csv file has a header")
	header := flagSet.String("header", "", "a custom header to use")
	indicies := flagSet.String("indicies", "", "headers to create indicies for")

	// Parse the CLI flags/environment variables
	flagSet.Parse(os.Args[1:])

	// Check that a CSV file was provided
	if csvFile == "" {
		log.Fatalln("no CSV file specified")
	}
	log.Printf("using %s as the input CSV file\n", csvFile)

	// Decode the delimiter into a Rune
	if csvDelimiter, _ = utf8.DecodeRuneInString(*delimiter); csvDelimiter == utf8.RuneError {
		log.Fatalf("'%s' is not a valid delimiter, expected a single character\n", *delimiter)
	}
	log.Printf("using %#U as the delimiter\n", csvDelimiter)

	// Check if a field count was provided
	if csvFieldCount == 0 {
		log.Println("field count was not provided, it will be detected automatically")
	} else {
		log.Printf("using %d as the field count\n", csvFieldCount)
	}

	// Check if the csv file has a header
	if csvHasHeader {
		log.Println("assuming the CSV file has a header")
	} else {
		log.Println("assuming the CSV file has no header")
	}

	// Check if a custom header was provided and attempt to parse the
	// header using either the CSV delimiter or comma delimiter.
	if *header != "" {
		csvHeader = trySplit(*header, string(csvDelimiter), ",")
		log.Printf("using a custom header: %+v", csvHeader)
	}

	// Check if a indicies were provided and attempt to parse the
	// indicies using either the CSV delimiter or comma delimiter.
	if *indicies != "" {
		csvIndicies = trySplit(*indicies, string(csvDelimiter), ",")
		log.Printf("using a provided indices: %+v", csvIndicies)
	}

}

func processCSV() {

	// Info message
	log.Println("*** Processing CSV file ***")

	// Open a connection to an in-memory sqlite database
	var err error
	db, err = sql.Open("sqlite3", "file::memory:?mode=memory&cache=shared")
	if err != nil {
		log.Fatalf("could not create SQLite database: %s\n", err)
	}

	// Open the CSV file
	file, err := os.Open(csvFile)
	if err != nil {
		log.Fatalf("could not open CSV file at: %s\n", csvFile)
	}
	defer file.Close()

	// Create the CSV reader
	reader := csv.NewReader(bufio.NewReader(file))
	reader.Comma = csvDelimiter
	reader.LazyQuotes = true

	// Read the header from the file if we've been told there's a header.
	// Don't do anything with it if a custom header was specified.
	if csvHasHeader {

		// Read the header
		line, err := reader.Read()
		if err != nil {
			log.Fatalf("could not read header: %s\n", err)
		}

		// Use the data header only if a custom header wasn't provided
		if len(csvHeader) == 0 {
			log.Println("using the first line in CSV as the header")
			csvHeader = line
		} else {
			log.Println("discarding header in favour of the provided custom header")
		}

	}

	// Determine the field count if it wasn't provided
	if csvFieldCount == 0 {
		csvFieldCount = len(csvHeader)
		log.Printf("using a field count of %d\n", csvFieldCount)
	}

	// Build the SQL command to create the table using the header information
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	query := "create table csv ("
	for i := 0; i < csvFieldCount; i++ {
		columnName := reg.ReplaceAllString(csvHeader[i], "")
		query += strings.ToLower(columnName)
		if i == csvFieldCount-1 {
			query += " text);"
		} else {
			query += " text, "
		}
	}

	// Execute the query
	log.Printf("creating SQLite table: %s\n", query)
	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("could not create SQLite table: %s\n", err)
	}

	// Create indicies
	for _, index := range csvIndicies {
		query := "create index " + index + "_idx on csv (" + index + ")"
		log.Printf("creating index: %s\n", query)
		_, err = db.Exec(query)
		if err != nil {
			log.Fatalf("could not create index on %s: %s\n", index, err)
		}
	}

	// Read the CSV data
	log.Println("importing CSV data into SQLite..")
	rows := 0
	for {

		// Read the next line, quit when we are done or if we encounter an error
		line, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatalln(err)
		}

		// Build the SQL command to insert the data
		query := "insert into csv values ("
		var args []interface{}
		for i := 0; i < csvFieldCount; i++ {
			if i == csvFieldCount-1 {
				query += "?);"
			} else {
				query += "?,"
			}
			args = append(args, line[i])
		}

		// Execute the query
		_, err = db.Exec(query, args...)
		if err != nil {
			log.Fatalf("could not import data into SQLite: %s; command: %s\n", err, query)
		}
		rows++

	}
	log.Printf("successfully imported %d rows into SQLite\n", rows)

}

func handleRequest(c *gin.Context) {

	// Build the query
	query := "SELECT * FROM csv WHERE "
	var args []interface{}
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			query += key + "=? AND "
			args = append(args, value)
		}
	}
	query = query[0 : len(query)-5]

	// Execute the query
	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("could not query for data: %s; command: %s\n", err, query)
		c.JSON(400, err.Error())
		return
	}
	defer rows.Close()

	// Convert the results into a JSON array
	data := []map[string]interface{}{}
	columns, _ := rows.Columns()
	for rows.Next() {

		// Create a slice to hold the resulting data and a second slice
		// of pointers to each element in the former slice.
		values := make([]string, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan the result into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("could not read row: %s\n", err)
			c.JSON(500, err.Error())
			return
		}

		// Store the results into a map, using the column name as the key. We also
		// attempt to convert the value into a float64 (number) when possible.
		m := make(map[string]interface{})
		for i, column := range columns {
			value := strings.Replace(values[i], ",", "", -1)
			if number, err := strconv.ParseFloat(value, 64); err == nil {
				m[column] = number
			} else {
				m[column] = values[i]
			}
		}

		// Add the converted row into the array of results
		data = append(data, m)

	}

	// Check if an error occured while iterating through the results
	if err := rows.Err(); err != nil {
		log.Printf("error iterating through the results: %s\n", err)
		c.JSON(500, err.Error())
		return
	}

	// Return the results as JSON
	c.IndentedJSON(200, data)

}

func serveJSON() {

	// Info message
	log.Println("*** Starting HTTP server ***")

	// Create a new GIN router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery(), cors.Default())

	// Request handler
	router.GET("/", handleRequest)

	// Create the HTTP server
	server = &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Start the HTTP server
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("could not start HTTP server: %s\n", err)
		}
	}()

}

func shutdown() {

	// Info message
	log.Println("*** Shutting Down ***")

	// Shutdown the HTTP server
	log.Println("stopping HTTP server..")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("could not stop HTTP server: %s\n", err)
	}

	// Close the database
	log.Println("closing SQLite database..")
	if err := db.Close(); err != nil {
		log.Printf("could not close database: %s\n", err)
	}

}

func main() {

	// Parse the environment for configuration
	parseEnvironment()

	// Convert the input CSV file into a SQLite database
	processCSV()

	// Serve the SQLite database over HTTP
	serveJSON()

	// Wait for interrupt signal to gracefully exit
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt

	// Gracefully shutdown when an interrupt is received
	shutdown()

}
