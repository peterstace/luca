package main

import (
	"sort"

	"github.com/peterstace/date"
)

type Book struct {
	accounts              map[string]struct{}
	singleTxns            []Transaction
	AmortizedTransactions []AmortizedTransaction
}

type Transaction struct {
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

func (m *Book) Series(account string) (date.Date, []Amount) {
	ledger := m.AccountLedger(account)
	if len(ledger) == 0 {
		return 0, nil
	}

	var series []Amount
	var i int
	for d := ledger[0].Date; d <= ledger[len(ledger)-1].Date; d++ {
		for i+1 < len(ledger) && ledger[i+1].Date <= d {
			i++
		}
		series = append(series, ledger[i].Balance)
	}
	return ledger[0].Date, series
}
