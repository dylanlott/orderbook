package accounts

import (
	"fmt"
)

// Transaction specifies an interface for transactions between Accounts.
type Transaction interface {
	Tx(fromID string, toID string, amount float64) ([]Account, error)
}

// AccountManager defines a simple CRUD interface for managing accounts.
type AccountManager interface {
	Get(id string) (Account, error)
	Create(id string, acct Account) (Account, error)
	Update(id string, acct Account) (Account, error)
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
	Accounts map[string]*UserAccount
}

// Get returns an account
func (i *InMemoryManager) Get(id string) (Account, error) {
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
	i.Accounts[email] = a
	return a, nil
}

// Tx transacts across accounts in the InMemoryManager.
func (i *InMemoryManager) Tx(from string, to string, amount float64) error {
	panic("not implemented") // TODO: implement transactions across amounts.
}

// Update updates the account id with provided Account information.
func (i *InMemoryManager) Update(id string, account Account) (Account, error) {
	panic("not implemented") // TODO: Implement
}

// Delete removes the account at key id in the accounts map.
func (i *InMemoryManager) Delete(id string) error {
	panic("not implemented") // TODO: Implement
}
