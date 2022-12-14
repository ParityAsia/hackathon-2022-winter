package polka

import (
	"errors"

	"githup.com/youthonline/ndk/base"
)

type Token struct {
	chain *Chain
}

func NewToken(chain *Chain) *Token {
	return &Token{chain}
}

// MARK - Implement the protocol Token

func (t *Token) Chain() base.Chain {
	return t.chain
}

// Warning: polka chain is not currently supported
func (t *Token) TokenInfo() (*base.TokenInfo, error) {
	return nil, errors.New("Polka chain is not currently supported")
}

func (t *Token) BalanceOfAddress(address string) (*base.Balance, error) {
	return t.chain.BalanceOfAddress(address)
}
func (t *Token) BalanceOfPublicKey(publicKey string) (*base.Balance, error) {
	return t.chain.BalanceOfPublicKey(publicKey)
}
func (t *Token) BalanceOfAccount(account base.Account) (*base.Balance, error) {
	return t.chain.BalanceOfAccount(account)
}
