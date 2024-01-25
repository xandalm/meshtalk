package meshtalk

import (
	"strconv"
	"time"
)

type InMemoryStorage struct {
	posts_pk int
	posts    map[string]Post
	comments map[string]map[string]Comment
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		1,
		map[string]Post{},
		map[string]map[string]Comment{},
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

func (s *InMemoryStorage) GetPosts() []Post {
	posts := make([]Post, 0, len(s.posts))

	for _, p := range s.posts {
		posts = append(posts, p)
	}

	return posts
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

func (s *InMemoryStorage) GetComments(post string) []Comment {
	var res []Comment
	for _, comments := range s.comments {
		for _, comment := range comments {
			res = append(res, comment)
		}
	}
	return res
}

func (s *InMemoryStorage) GetComment(post, id string) *Comment {
	var found Comment
	found, ok := s.comments[post][id]
	if !ok {
		return nil
	}
	return &Comment{
		Id:        found.Id,
		Post:      found.Post,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}
}
