package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

func main() {
	csvDir := flag.String("csv-dir", os.ExpandEnv("$LUCA_CSV_DIR"), "directory containing csvs")
	operation := flag.String("operation", "", "operation to perform")
	account := flag.String("account", "", "account to operate on")
	flag.Parse()

	if *csvDir == "" {
		fmt.Fprintf(os.Stderr, "CSV dir not set (--csv-dir flag or LUCA_CSV_DIR env)\n")
		flag.Usage()
		os.Exit(1)
	}
	if *operation == "" {
		fmt.Fprintf(os.Stderr, "Operation not set (--operation flag)\n")
		flag.Usage()
		os.Exit(1)
	}
	if *account == "" && *operation != "coa" {
		fmt.Fprintf(os.Stderr, "Account not set (--account flag)\n")
		flag.Usage()
		os.Exit(1)
	}

	var op func(b Book, account string) (interface{}, error)
	switch *operation {
	case "ledger":
		op = ledgerOperation
	case "summary":
		op = summaryOperation
	case "coa":
		op = coaOperation
	case "series":
		op = seriesOperation
	default:
		fmt.Fprintf(os.Stderr, "unknown operation %q\n", *operation)
		flag.Usage()
		os.Exit(1)
	}

	book, err := buildBook(*csvDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not build book: %v\n", err)
		flag.Usage()
		os.Exit(1)
	}
	if len(book.Accounts()) == 0 {
		fmt.Fprintf(os.Stderr, "loaded 0 accounts\n")
		flag.Usage()
		os.Exit(1)
	}

	result, err := op(book, *account)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not perform operation %v: %v\n", *operation, err)
		flag.Usage()
		os.Exit(1)
	}

	buf, err := json.Marshal(result)
	if err != nil {
		panic(fmt.Sprintf("could not marshal result: %v", err))
	}
	os.Stdout.Write(buf)
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

func ledgerOperation(b Book, account string) (interface{}, error) {
	ledger := b.AccountLedger(account)
	return ledger, nil
}

func summaryOperation(b Book, account string) (interface{}, error) {
	accountsRegexp, err := regexp.Compile(account)
	if err != nil {
		return nil, err
	}
	return b.SummariseAccounts(accountsRegexp), nil
}

func coaOperation(b Book, _ string) (interface{}, error) {
	return b.Accounts(), nil
}

func seriesOperation(b Book, account string) (interface{}, error) {
	accountsRegexp, err := regexp.Compile(account)
	if err != nil {
		return nil, err
	}
	return b.Series(accountsRegexp), nil
}
