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

// It's like a helper to expose editable fields of the Customer.
// The editable fields are pointers because this way is possible
// to keep them optional to be edited.
type CustomerInEditting struct {
	Id   string
	Name *string
}
