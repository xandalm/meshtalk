package main

import (
	"log"
	"meshtalk"
	"net/http"
	"strconv"
)

type InMemoryStorage struct {
	posts_pk int
	posts    map[string]*meshtalk.Post
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		1,
		map[string]*meshtalk.Post{},
	}
}

func (s *InMemoryStorage) GetPost(id string) *meshtalk.Post {
	post := s.posts[id]
	return post
}

func (s *InMemoryStorage) StorePost(post *meshtalk.Post) string {
	post.Id = strconv.Itoa(s.posts_pk)
	s.posts_pk++
	s.posts[post.Id] = post
	return post.Id
}

func (s *InMemoryStorage) EditPost(post *meshtalk.Post) bool {
	found := s.GetPost(post.Id)
	if found == nil {
		return false
	}
	found.Title = post.Title
	found.Content = post.Content
	return true
}

func (s *InMemoryStorage) DeletePost(id string) bool {
	delete(s.posts, id)
	return true
}

func main() {
	storage := NewInMemoryStorage()
	server := meshtalk.NewServer(storage)
	log.Fatal(http.ListenAndServe(":5000", server))
}
