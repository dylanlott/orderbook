package accounts

// AccountManager defines a simple CRUD interface for managing accounts.
type AccountManager interface {
	Get(id string) Account
	Create() (Account, error)
	Update() (Account, error)
	Delete() error
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
