package accounts

import (
	"fmt"
	"log"
	"sync"
)

// Transaction specifies an interface for transactions between Accounts.
type Transaction interface {
	Tx(fromID string, toID string, amount float64) ([]Account, error)
}

// AccountManager defines a simple CRUD interface for managing accounts.
type AccountManager interface {
	Transaction

	Get(id string) (Account, error)
	Create(id string, acct Account) (Account, error)
	Delete(id string) error
}

// Account relates a user to a balance sheet in our system.
// Filling an Order will add or subtract from the account's Balance
type Account interface {
	// UserID returns a unique ID for the acocunt.
	UserID() string
	// Balance returns the balance of the account.
	Balance() float64
}

// UserAccount fulfills the Account interface with a typical user implementation
type UserAccount struct {
	Email          string
	CurrentBalance float64
}

// UserID returns the unique identifier for a UserAccount which is Email
func (u *UserAccount) UserID() string {
	return u.Email
}

// Balance returns the account balance in dollars as float64
func (u *UserAccount) Balance() float64 {
	return u.CurrentBalance
}

// InMemoryManager is an in memory account manager for testing purposes.
type InMemoryManager struct {
	sync.Mutex

	Accounts map[string]*UserAccount
}

// Get returns an account
func (i *InMemoryManager) Get(id string) (Account, error) {
	i.Lock()
	defer i.Unlock()
	if v, ok := i.Accounts[id]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("failed to find account %s", id)
}

// Create makes a new account
func (i *InMemoryManager) Create(email string, account Account) (Account, error) {
	a := &UserAccount{
		Email:          email,
		CurrentBalance: 0.0,
	}
	i.Lock()
	defer i.Unlock()
	i.Accounts[email] = a
	return a, nil
}

// Tx transacts across accounts in the InMemoryManager.
func (i *InMemoryManager) Tx(from string, to string, amount float64) ([]Account, error) {
	i.Lock()
	defer i.Unlock()
	fromAcct, ok := i.Accounts[from]
	if !ok {
		return nil, fmt.Errorf("account %s does not exist", from)
	}

	toAcct, ok := i.Accounts[to]
	if !ok {
		return nil, fmt.Errorf("account %s does not exist", to)
	}

	if fromAcct.Balance() < amount {
		return nil, fmt.Errorf("insufficient balance %v in %v", amount, fromAcct)
	}

	// everything checks out so let's do the math now
	fromAcct.CurrentBalance = fromAcct.CurrentBalance - amount
	toAcct.CurrentBalance = toAcct.CurrentBalance + amount

	// save the udpated account balances to our InMemoryManager.
	i.Accounts[from] = fromAcct
	i.Accounts[to] = toAcct

	log.Printf("transaction: moved %v from %s to account %s", amount, from, to)

	return []Account{fromAcct, toAcct}, nil
}

// Delete removes the account at key id in the accounts map.
func (i *InMemoryManager) Delete(id string) error {
	delete(i.Accounts, id)
	return nil
}
