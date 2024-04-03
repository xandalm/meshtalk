package storage

import (
	"errors"
	"meshtalk/domain/entities"
)

var (
	ErrPostNotFound         = errors.New("storage: post not found")
	ErrMissingPostFields    = errors.New("storage: title, content and author are required for the post")
	ErrCommentNotFound      = errors.New("storage: comment not found")
	ErrMissingCommentFields = errors.New("storage: content and author are required for the comment")
)

type Storage interface {
	CreateCustomer(customer *entities.Customer) error
	GetPost(id string) (*entities.Post, error)
	GetPosts() ([]entities.Post, error)
	CreatePost(post *entities.Post) error
	EditPost(post *entities.Post) error
	DeletePost(id string) error
	GetComments(post string) ([]entities.Comment, error)
	GetComment(post, id string) (*entities.Comment, error)
	CreateComment(comment *entities.Comment) error
	EditComment(comment *entities.Comment) error
	DeleteComment(post, id string) error
}
