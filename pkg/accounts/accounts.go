package accounts

type Account interface {
	UserID() string
	Email() string
	Balance() float64
}
