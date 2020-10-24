package main

import (
	"errors"
	"fmt"
)

type UnknownAccountError struct {
	Account string
}

func (e UnknownAccountError) Error() string {
	return fmt.Sprintf("unknown account %s", e.Account)
}

var (
	ErrNegativeAmount = errors.New("negative amount")
)
