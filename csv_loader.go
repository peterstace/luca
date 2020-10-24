package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/peterstace/date"
)

type CSVLoader struct {
	coa    map[string]struct{}
	txns   []transactionFile
	amorts []amortisationFile
}

type transactionFile struct {
	filename string
	txns     []transactionRecord
}

type transactionRecord struct {
	id          string
	date        string
	drAccount   string
	crAccount   string
	amount      string
	description string
}

type amortisationFile struct {
	filename string
	amorts   []amortisationRecord
}

type amortisationRecord struct {
	txnDate         string
	startDate       string
	endDate         string
	singleDRAccount string
	singleCRAccount string
	repeatDRAccount string
	repeatCRAccount string
	amount          string
	description     string
}

func (m *CSVLoader) LoadCSV(r io.Reader, filename string) error {
	cr := csv.NewReader(r)
	recs, err := cr.ReadAll()
	if err != nil {
		return err
	}
	if len(recs) == 0 {
		return errors.New("missing header")
	}
	for i := range recs {
		for j := range recs[i] {
			recs[i][j] = strings.TrimSpace(recs[i][j])
		}
	}
	switch {
	case reflect.DeepEqual(recs[0], coaHeader):
		if m.coa == nil {
			m.coa = make(map[string]struct{})
		}
		for _, rec := range recs[1:] {
			m.coa[rec[0]] = struct{}{}
		}
		return nil
	case reflect.DeepEqual(recs[0], txnHeader):
		var txns []transactionRecord
		for _, rec := range recs[1:] {
			txns = append(txns, transactionRecord{"", rec[0], rec[1], rec[2], rec[3], rec[4]})
		}
		m.txns = append(m.txns, transactionFile{filename, txns})
		return nil
	case reflect.DeepEqual(recs[0], txnWithIDHeader):
		var txns []transactionRecord
		for _, rec := range recs[1:] {
			txns = append(txns, transactionRecord{rec[0], rec[1], rec[2], rec[3], rec[4], rec[5]})
		}
		m.txns = append(m.txns, transactionFile{filename, txns})
		return nil
	case reflect.DeepEqual(recs[0], amortHeader):
		var amorts []amortisationRecord
		for _, rec := range recs[1:] {
			amorts = append(amorts, amortisationRecord{
				rec[0], rec[1], rec[2], rec[3], rec[4], rec[5], rec[6], rec[7], rec[8]},
			)
		}
		m.amorts = append(m.amorts, amortisationFile{filename, amorts})
		return nil
	default:
		return fmt.Errorf("unknown header: %v", recs[0])
	}
}

var (
	coaHeader = []string{
		"Account",
	}
	txnHeader = []string{
		"Date", "DR", "CR", "Amount", "Description",
	}
	txnWithIDHeader = []string{
		"ID", "Date", "DR", "CR", "Amount", "Description",
	}
	amortHeader = []string{
		"TxnDate", "StartDate", "EndDate",
		"SingleDR", "SingleCR", "RepeatDR", "RepeatCR",
		"Amount", "Description",
	}
)

func (m *CSVLoader) Book() (Book, error) {
	var book Book

	for account := range m.coa {
		book.AddAccount(account)
	}

	for _, txnFile := range m.txns {
		var prevDate date.Date
		for _, txn := range txnFile.txns {
			date, err := date.FromString(txn.date)
			if err != nil {
				return Book{}, fmt.Errorf("invalid date: %v", err)
			}
			if date < prevDate {
				return Book{}, fmt.Errorf("decreasing dates: %v and %v in file %v", prevDate, date, txnFile.filename)
			}
			prevDate = date

			amount, err := AmountFromString(txn.amount)
			if err != nil {
				return Book{}, fmt.Errorf("invalid amount: %v", err)
			}

			if err := book.AddSingleTransaction(Transaction{
				ID:   txn.id,
				Date: date,
				Account: Accounts{
					DR: txn.drAccount,
					CR: txn.crAccount,
				},
				Amount:      amount,
				Description: txn.description,
			}); err != nil {
				return Book{}, err
			}
		}
	}

	for _, amortFile := range m.amorts {
		var prevDate date.Date
		for _, amort := range amortFile.amorts {
			txnDate, err := date.FromString(amort.txnDate)
			if err != nil {
				return Book{}, fmt.Errorf("invalid date: %v", err)
			} else if txnDate < prevDate {
				return Book{}, fmt.Errorf(
					"decreasing dates: %v and %v in file %v",
					prevDate, txnDate, amortFile.filename,
				)
			}
			prevDate = txnDate

			startDate, err := date.FromString(amort.startDate)
			if err != nil {
				return Book{}, fmt.Errorf("invalid date: %v", err)
			}
			endDate, err := date.FromString(amort.endDate)
			if err != nil {
				return Book{}, fmt.Errorf("invalid date: %v", err)
			}

			amount, err := AmountFromString(amort.amount)
			if err != nil {
				return Book{}, fmt.Errorf("invalid amount: %v", err)
			}

			amortTxn := AmortizedTransaction{
				SingleDate: txnDate,
				DateRange: DateRange{
					StartInclusive: startDate,
					EndExclusive:   endDate,
				},
				Single: Accounts{
					DR: amort.singleDRAccount,
					CR: amort.singleCRAccount,
				},
				Repeat: Accounts{
					DR: amort.repeatDRAccount,
					CR: amort.repeatCRAccount,
				},
				Amount:      amount,
				Description: amort.description,
			}
			if err := book.AddAmortizedTransaction(amortTxn); err != nil {
				return Book{}, fmt.Errorf("could not add amort transaction: %v", err)
			}
		}
	}
	return book, nil
}
