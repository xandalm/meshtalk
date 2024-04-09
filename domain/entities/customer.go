package entities

type Customer struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

func NewCustomer(id, name string) *Customer {
	return &Customer{
		Id:   id,
		Name: name,
	}
}

type CustomerInEditting struct {
	Id   string
	Name *string
}
