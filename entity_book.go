package main

import (
	"regexp"
	"sort"

	"github.com/peterstace/date"
)

type Book struct {
	accounts              map[string]struct{}
	singleTxns            []Transaction
	AmortizedTransactions []AmortizedTransaction
}

type Transaction struct {
	ID          string    `json:"id"`
	Date        date.Date `json:"date"`
	Account     Accounts  `json:"account"`
	Amount      Amount    `json:"amount"`
	Description string    `json:"description"`
}

type AmortizedTransaction struct {
	SingleDate  date.Date
	DateRange   DateRange
	Single      Accounts
	Repeat      Accounts
	Amount      Amount
	Description string
}

type Accounts struct {
	DR string `json:"dr"`
	CR string `json:"cr"`
}

type DateRange struct {
	StartInclusive date.Date
	EndExclusive   date.Date
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
	b.singleTxns = append(b.singleTxns, txn)
	return nil
}

func (b *Book) AddAmortizedTransaction(txn AmortizedTransaction) error {
	for _, acc := range []string{
		txn.Single.DR,
		txn.Single.CR,
		txn.Repeat.DR,
		txn.Repeat.CR,
	} {
		if _, ok := b.accounts[acc]; !ok {
			return UnknownAccountError{acc}
		}
	}
	if txn.Amount < 0 {
		return ErrNegativeAmount
	}
	b.AmortizedTransactions = append(b.AmortizedTransactions, txn)
	return nil
}

func (m *Book) AllTransactions() []Transaction {
	all := m.singleTxns
	for _, amort := range m.AmortizedTransactions {
		all = append(all, Transaction{
			Date:        amort.SingleDate,
			Account:     amort.Single,
			Amount:      amort.Amount,
			Description: amort.Description,
		})
		for d := amort.DateRange.StartInclusive; d < amort.DateRange.EndExclusive; d++ {
			amount := amort.Amount / Amount(amort.DateRange.EndExclusive-d)
			amort.Amount -= amount
			all = append(all, Transaction{
				Date:        d,
				Account:     amort.Repeat,
				Amount:      amount,
				Description: amort.Description,
			})
		}
	}
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

	start := txns[0].Date
	end := txns[len(txns)-1].Date
	var i int
	var bal Amount
	var entries []SeriesEntry
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
