package entities

// Represents a customer.
type Customer struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

// Creates a new customer representation.
func NewCustomer(id, name string) *Customer {
	return &Customer{
		Id:   id,
		Name: name,
	}
}

// It's like a helper to expose editable fields of the Customer.
// The editable fields are pointers because this way is possible
// to keep them optional to be edited.
type CustomerInEditting struct {
	Id   string
	Name *string
}
