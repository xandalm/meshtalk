package services

import "meshtalk/domain/entities"

type Storage interface {
	GetPost(id string) *entities.Post
	GetPosts() []entities.Post
	StorePost(post *entities.Post) error
	EditPost(post *entities.Post) error
	DeletePost(id string) error
	GetComments(post string) []entities.Comment
	GetComment(post, id string) *entities.Comment
	StoreComment(comment *entities.Comment) error
	EditComment(comment *entities.Comment) error
}
