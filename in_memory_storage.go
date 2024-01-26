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
	if !ok || found.DeletedAt == "" {
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
		if p.DeletedAt == "" {
			posts = append(posts, p)
		}
	}

	return posts
}

func timeToString(t time.Time) string {
	b, _ := t.UTC().MarshalText()
	str := string(b)
	return str
}

func (s *InMemoryStorage) StorePost(post *Post) error {
	post.Id = strconv.Itoa(s.posts_pk)
	post.CreatedAt = timeToString(time.Now())
	s.posts[post.Id] = *post
	s.posts_pk++
	return nil
}

func (s *InMemoryStorage) EditPost(post *Post) error {
	found, ok := s.posts[post.Id]
	if !ok || found.DeletedAt != "" {
		return ErrPostNotFound
	}

	post.UpdatedAt = timeToString(time.Now())
	s.posts[post.Id] = *post
	return nil
}

func (s *InMemoryStorage) DeletePost(id string) error {
	post, ok := s.posts[id]
	if !ok {
		return ErrPostNotFound
	}
	post.DeletedAt = timeToString(time.Now())
	s.posts[id] = post
	return nil
}

func (s *InMemoryStorage) GetComments(post string) []Comment {
	var res []Comment
	for _, comments := range s.comments {
		for _, comment := range comments {
			if comment.DeletedAt != "" {
				res = append(res, comment)
			}
		}
	}
	return res
}

func (s *InMemoryStorage) GetComment(post, id string) *Comment {
	var found Comment
	found, ok := s.comments[post][id]
	if !ok || found.DeletedAt != "" {
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

func (s *InMemoryStorage) StoreComment(comment *Comment) error {
	_, hasComments := s.comments[comment.Post]
	if !hasComments {
		s.comments[comment.Post] = make(map[string]Comment)
	}
	comment.Id = strconv.Itoa(len(s.comments[comment.Post]) + 1)
	comment.CreatedAt = timeToString(time.Now())
	s.comments[comment.Post][comment.Id] = *comment
	return nil
}
