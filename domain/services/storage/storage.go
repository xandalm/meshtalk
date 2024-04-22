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
	ErrMissingCustomerFields = errors.New("storage: tag, name and password is required for the customer")
	ErrUnrecognizedAuthor    = errors.New("storage: invalid author, must be a customer id")
)

// Storage manages the entities data persistence.
type Storage interface {
	// Create customer data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrMissingCustomerFields.
	// Any other error is a uncategorized for storage interface.
	CreateCustomer(customer *entities.Customer) error

	// Get customer data.
	//
	// In case of error, this error is uncategorized for storage interface.
	GetCustomer(id string) (*entities.Customer, error)

	// Get customer data by tag.
	//
	// In case of error, this error is uncategorized for storage interface.
	GetCustomerByTag(tag string) (*entities.Customer, error)

	// Update customer data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrCustomerNotFound,
	// ErrMissingCustomerFields.
	// Any other error is a uncategorized for storage interface.
	EditCustomer(edit entities.CustomerInEditting) (*entities.Customer, error)

	// Delete customer data.
	//
	// In case of error, this error is uncategorized for storage interface.
	DeleteCustomer(id string) error

	// Create post data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrUnrecognizedAuthor,
	// ErrMissingPostFields.
	// Any other error is a uncategorized for storage interface.
	CreatePost(post *entities.Post) error

	// Get post data.
	//
	// In case of error, this error is uncategorized for storage interface.
	GetPost(id string) (*entities.Post, error)

	// Get post dataset.
	//
	// In case of error, this error is uncategorized for storage interface.
	GetPosts() ([]entities.Post, error)

	// Update post data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrPostNotFound,
	// ErrMissingPostFields.
	// Any other error is a uncategorized for storage interface.
	EditPost(edit entities.PostInEditting) (*entities.Post, error)

	// Delete post data.
	//
	// In case of error, this error is uncategorized for storage interface.
	DeletePost(id string) error

	// Create comment data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrPostNotFound,
	// ErrCustomerNotFound,
	// ErrMissingCommentFields.
	// Any other error is a uncategorized for storage interface.
	CreateComment(comment *entities.Comment) error

	// Get comment data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrPostNotFound.
	// Any other error is a uncategorized for storage interface.
	GetComment(post, id string) (*entities.Comment, error)

	// Get comment dataset.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrPostNotFound.
	// Any other error is a uncategorized for storage interface.
	GetComments(post string) ([]entities.Comment, error)

	// Update comment data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrPostNotFound,
	// ErrMissingCommentFields.
	// Any other error is a uncategorized for storage interface.
	EditComment(edit entities.CommentInEditting) (*entities.Comment, error)

	// Delete comment data.
	//
	// In case of error, it's expected to return one of the errors:
	// ErrPostNotFound.
	// Any other error is a uncategorized for storage interface.
	DeleteComment(post, id string) error
}
