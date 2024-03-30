package services

import "meshtalk/domain/entities"

type Storage interface {
	GetPost(id string) (*entities.Post, error)
	GetPosts() ([]entities.Post, error)
	StorePost(post *entities.Post) error
	EditPost(post *entities.Post) error
	DeletePost(id string) error
	GetComments(post string) ([]entities.Comment, error)
	GetComment(post, id string) (*entities.Comment, error)
	StoreComment(comment *entities.Comment) error
	EditComment(comment *entities.Comment) error
}
