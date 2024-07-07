package main

import (
	"regexp"
	"sort"
	"strconv"

	"github.com/peterstace/date"
)

type Book struct {
	accounts map[string]struct{}
	txns     []Transaction
}

type Transaction struct {
	ID          string    `json:"id"`
	Date        date.Date `json:"date"`
	Account     Accounts  `json:"account"`
	Amount      Amount    `json:"amount"`
	Description string    `json:"description"`
}

type Accounts struct {
	DR string `json:"dr"`
	CR string `json:"cr"`
}

func (m *Book) AddAccount(accountName string) {
	if m.accounts == nil {
		m.accounts = make(map[string]struct{})
	}
	m.accounts[accountName] = struct{}{}
}

func (m *Book) Accounts() []string {
	var accounts []string
	for acc := range m.accounts {
		accounts = append(accounts, acc)
	}
	sort.Strings(accounts)
	return accounts
}

func (b *Book) AddSingleTransaction(txn Transaction) error {
	if _, ok := b.accounts[txn.Account.DR]; !ok {
		return UnknownAccountError{Account: txn.Account.DR}
	}
	if _, ok := b.accounts[txn.Account.CR]; !ok {
		return UnknownAccountError{Account: txn.Account.CR}
	}
	if txn.Amount < 0 {
		return ErrNegativeAmount
	}
	b.txns = append(b.txns, txn)
	return nil
}

func (m *Book) AllTransactions() []Transaction {
	all := m.txns
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Date < all[j].Date
	})
	return all
}

type AccountLedgerEntry struct {
	Transaction `json:"transaction"`
	Balance     Amount `json:"balance"`
}

func (m *Book) AccountLedger(account string) []AccountLedgerEntry {
	var entries []AccountLedgerEntry
	var bal Amount
	for _, txn := range m.AllTransactions() {
		if account != txn.Account.CR && account != txn.Account.DR {
			continue
		}
		if account == txn.Account.DR {
			bal += txn.Amount
			entries = append(entries, AccountLedgerEntry{
				Transaction: txn,
				Balance:     bal,
			})
		}
		if account == txn.Account.CR {
			bal -= txn.Amount
			entries = append(entries, AccountLedgerEntry{
				Transaction: txn,
				Balance:     bal,
			})
		}
	}
	return entries
}

type AccountSummary struct {
	Account          string    `json:"account"`
	Balance          Amount    `json:"balance"`
	LastTransaction  date.Date `json:"lastTransactionDate"`
	TransactionCount int       `json:"transactionCount"`
}

func (m *Book) SummariseAccounts(accounts *regexp.Regexp) []AccountSummary {
	summariesMap := make(map[string]AccountSummary)
	for _, txn := range m.AllTransactions() {
		update := func(account string, neg bool) {
			if !accounts.MatchString(account) {
				return
			}
			summary := summariesMap[account]
			summary.Account = account
			if neg {
				summary.Balance -= txn.Amount
			} else {
				summary.Balance += txn.Amount
			}
			summary.LastTransaction = txn.Date
			summary.TransactionCount++
			summariesMap[account] = summary
		}
		update(txn.Account.DR, false)
		update(txn.Account.CR, true)
	}
	summariesList := make([]AccountSummary, 0, len(summariesMap))
	for _, summary := range summariesMap {
		summariesList = append(summariesList, summary)
	}
	sort.Slice(summariesList, func(i, j int) bool {
		return summariesList[i].Account < summariesList[j].Account
	})
	return summariesList
}

type Series struct {
	Entries []SeriesEntry `json:"entries"`
}

type SeriesEntry struct {
	Date    date.Date `json:"date"`
	Balance Amount    `json:"balance"`
}

func (m *Book) Series(accounts *regexp.Regexp) Series {
	txns := m.AllTransactions()
	if len(txns) == 0 {
		return Series{}
	}

	end := txns[len(txns)-1].Date
	start := end + 1
	for _, txn := range txns {
		if accounts.MatchString(txn.Account.DR) || accounts.MatchString(txn.Account.CR) {
			start = txn.Date
			break
		}
	}

	var i int
	var bal Amount
	entries := make([]SeriesEntry, 0, end-start+1)
	for d := start; d <= end; d++ {
		for i+1 < len(txns) && txns[i+1].Date <= d {
			i++
			txn := txns[i]
			if accounts.MatchString(txn.Account.DR) {
				bal += txn.Amount
			}
			if accounts.MatchString(txn.Account.CR) {
				bal -= txn.Amount
			}
		}
		entries = append(entries, SeriesEntry{d, bal})
	}
	return Series{entries}
}

func (m *Book) Reconstruct(account string) [][]string {
	header := []string{
		"date",
		"amount",
		"balance",
		"other_account",
		"description",
	}
	rows := [][]string{header}

	var balance Amount
	for _, txn := range m.AllTransactions() {
		if txn.Account.CR != account && txn.Account.DR != account {
			continue
		}
		other := txn.Account.DR
		if other == account {
			other = txn.Account.CR
		}

		amount := txn.Amount
		if txn.Account.DR == account {
			amount *= -1
		}
		balance += amount

		row := []string{
			txn.Date.String(),
			strconv.FormatFloat(float64(amount)/100, 'f', 2, 64),
			strconv.FormatFloat(float64(balance)/100, 'f', 2, 64),
			other,
			txn.Description,
		}
		rows = append(rows, row)
	}
	return rows
}
