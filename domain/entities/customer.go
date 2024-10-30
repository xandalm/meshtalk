package entities

// Represents a customer.
type Customer struct {
	Id        string `json:"id"`
	Tag       string `json:"tag"`
	Name      string `json:"name"`
	Password  string `json:"password"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

// Creates a new customer representation.
func NewCustomer(tag, name, password string) *Customer {
	return &Customer{
		Tag:      tag,
		Name:     name,
		Password: password,
	}
}
