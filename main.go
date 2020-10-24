package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	csvDir := flag.String("csv-dir", os.ExpandEnv("$LUCA_CSV_DIR"), "directory containing csvs")
	operation := flag.String("operation", "ledger", "operation to perform")
	account := flag.String("account", "", "account to operate on")
	flag.Parse()

	if *csvDir == "" {
		fmt.Fprintf(os.Stderr, "CSV dir not set (--csv-dir flag or LUCA_CSV_DIR env)\n")
		flag.Usage()
	}
	if *operation == "" {
		fmt.Fprintf(os.Stderr, "Operation not set (--operation flag)")
		flag.Usage()
	}
	if *account == "" {
		fmt.Fprintf(os.Stderr, "Account not set (--account flag)")
		flag.Usage()
	}

	switch *operation {
	case "ledger":
		book, err := buildBook(*csvDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "could not build book: %v\n", err)
			flag.Usage()
		}
		if len(book.Accounts()) == 0 {
			fmt.Fprintf(os.Stderr, "loaded 0 accounts\n")
			flag.Usage()
		}
		ledger := book.AccountLedger(*account)
		buf, err := json.Marshal(ledger)
		if err != nil {
			panic(fmt.Sprintf("could not marshal account ledger: %v", err))
		}
		os.Stdout.Write(buf)
	default:
		fmt.Fprintf(os.Stderr, "unknown operation %q", *operation)
		flag.Usage()
	}
}

func buildBook(csvDir string) (Book, error) {
	dirListing, err := ioutil.ReadDir(csvDir)
	if err != nil {
		return Book{}, fmt.Errorf("could not read %q: %v", csvDir, err)
	}
	var loader CSVLoader
	for _, entry := range dirListing {
		fname := entry.Name()
		if entry.IsDir() || filepath.Ext(fname) != ".csv" {
			continue
		}
		f, err := os.Open(filepath.Join(csvDir, fname))
		if err != nil {
			return Book{}, fmt.Errorf("could not open %q: %v", err, fname)
		}
		defer f.Close()
		if err := loader.LoadCSV(f, fname); err != nil {
			f.Close()
			return Book{}, fmt.Errorf("could not load csv %q: %v", fname, err)
		}
	}
	return loader.Book()
}
