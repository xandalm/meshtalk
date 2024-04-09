package storage

import (
	"errors"
	"meshtalk/domain/entities"
)

var (
	ErrPostNotFound          = errors.New("storage: post not found")
	ErrMissingPostFields     = errors.New("storage: title, content and author are required for the post")
	ErrCommentNotFound       = errors.New("storage: comment not found")
	ErrMissingCommentFields  = errors.New("storage: content and author are required for the comment")
	ErrCustomerNotFound      = errors.New("storage: customer not found")
	ErrMissingCustomerFields = errors.New("storage: name is required for the customer")
)

type Storage interface {
	CreateCustomer(customer *entities.Customer) error
	GetCustomer(id string) (*entities.Customer, error)
	EditCustomer(edit entities.CustomerInEditting) (*entities.Customer, error)
	DeleteCustomer(id string) error
	GetPost(id string) (*entities.Post, error)
	GetPosts() ([]entities.Post, error)
	CreatePost(post *entities.Post) error
	EditPost(edit entities.PostInEditting) (*entities.Post, error)
	DeletePost(id string) error
	GetComments(post string) ([]entities.Comment, error)
	GetComment(post, id string) (*entities.Comment, error)
	CreateComment(comment *entities.Comment) error
	EditComment(edit entities.CommentInEditting) (*entities.Comment, error)
	DeleteComment(post, id string) error
}
