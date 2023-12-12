package meshtalk

import (
	"strconv"
	"time"
)

type InMemoryStorage struct {
	posts_pk int
	posts    map[string]Post
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		1,
		map[string]Post{},
	}
}

func (s *InMemoryStorage) GetPost(id string) *Post {
	found, ok := s.posts[id]
	if !ok {
		return nil
	}
	return &Post{
		Id:        found.Id,
		Title:     found.Title,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}
}

func (s *InMemoryStorage) StorePost(post *Post) error {
	post.Id = strconv.Itoa(s.posts_pk)
	createdAt := time.Now()
	post.CreatedAt = &createdAt
	s.posts[post.Id] = *post
	s.posts_pk++
	return nil
}

func (s *InMemoryStorage) EditPost(post *Post) error {
	found, ok := s.posts[post.Id]
	if !ok {
		return ErrPostNotFound
	}
	found.Title = post.Title
	found.Content = post.Content
	found.Author = post.Author

	updatedAt := time.Now()
	found.UpdatedAt = &updatedAt
	post.UpdatedAt = &updatedAt

	s.posts[post.Id] = found
	return nil
}

func (s *InMemoryStorage) DeletePost(id string) error {
	delete(s.posts, id)
	return nil
}
