package main

import (
	"encoding/json"
	"fmt"
)

// Amount is a monetary amount in cents.
type Amount int

// String gives a string representation of the dollar amount using using
// comma separators (thousands, millions etc). The dollar amount is always
// shown to 2 decimal places.
func (a Amount) String() string {
	var buf []byte
	var neg bool
	if a < 0 {
		neg = true
		a *= -1
	}
	var pos int
	for a != 0 || pos < 3 {
		if pos == 2 {
			buf = append(buf, '.')
		}
		if place := (pos - 2); place > 0 && place%3 == 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, '0'+byte(a%10))
		a /= 10
		pos++
	}
	if neg {
		buf = append(buf, '-')
	}
	for i := 0; i < len(buf)/2; i++ {
		j := len(buf) - i - 1
		buf[i], buf[j] = buf[j], buf[i]
	}
	return string(buf)
}

func AmountFromString(s string) (Amount, error) {
	var dollars float64
	_, err := fmt.Sscanf(s, "%f", &dollars)
	if dollars >= 0 {
		return Amount(dollars*100 + 0.5), err
	}
	return Amount(dollars*100 - 0.5), err
}

func (a Amount) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}
