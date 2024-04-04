package httpserver

import (
	"errors"
	"fmt"
	"meshtalk/domain/entities"
	"meshtalk/domain/services/storage"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type stubStorage struct {
	customers        map[string]entities.Customer
	posts            map[string]entities.Post
	comments         map[string]map[string]entities.Comment
	postEditCalls    []string
	commentEditCalls []string
}

func NewStubStorage() *stubStorage {
	return &stubStorage{
		map[string]entities.Customer{},
		map[string]entities.Post{},
		map[string]map[string]entities.Comment{},
		[]string{},
		[]string{},
	}
}

func (s *stubStorage) GetPost(id string) (*entities.Post, error) {
	found, ok := s.posts[id]
	if !ok {
		return nil, nil
	}
	return &entities.Post{
		Id:        found.Id,
		Title:     found.Title,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}, nil
}

func (s *stubStorage) GetPosts() ([]entities.Post, error) {
	posts := make([]entities.Post, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}
	return posts, nil
}

func (s *stubStorage) CreatePost(post *entities.Post) error {
	if post.Title == "" || post.Content == "" || post.Author == "" {
		return storage.ErrMissingPostFields
	}
	post.Id = strconv.Itoa(len(s.posts) + 1)
	post.CreatedAt = timeToString(time.Now())
	s.posts[post.Id] = *post
	return nil
}

func timeToString(t time.Time) string {
	b, _ := t.UTC().MarshalText()
	return string(b)
}

func (s *stubStorage) EditPost(post *entities.Post) error {
	found, ok := s.posts[post.Id]
	if !ok {
		return storage.ErrPostNotFound
	}
	if post.Title == "" {
		post.Title = found.Title
	}
	if post.Content == "" {
		post.Content = found.Content
	}
	if post.Author == "" {
		post.Author = found.Author
	}
	s.postEditCalls = append(s.postEditCalls, post.Id)
	return nil
}

func (s *stubStorage) DeletePost(id string) error {
	delete(s.posts, id)
	return nil
}

func (s *stubStorage) GetComments(post string) ([]entities.Comment, error) {
	var res []entities.Comment

	if post != "" {
		found, ok := s.comments[post]
		if !ok {
			return nil, storage.ErrPostNotFound
		}
		for _, comment := range found {
			res = append(res, comment)
		}
	} else {
		for _, comments := range s.comments {
			for _, comment := range comments {
				res = append(res, comment)
			}
		}
	}

	return res, nil
}

func (s *stubStorage) GetComment(post, id string) (*entities.Comment, error) {
	var found entities.Comment
	found, ok := s.comments[post][id]
	if !ok {
		return nil, nil
	}
	return &entities.Comment{
		Id:        found.Id,
		Post:      found.Post,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}, nil
}

func (s *stubStorage) CreateComment(comment *entities.Comment) error {
	if comment.Author == "" || comment.Content == "" {
		return storage.ErrMissingCommentFields
	}
	if _, hasPost := s.posts[comment.Post]; !hasPost {
		return storage.ErrPostNotFound
	}
	_, hasComments := s.comments[comment.Post]
	if !hasComments {
		s.comments[comment.Post] = make(map[string]entities.Comment)
	}
	comment.Id = strconv.Itoa(len(s.comments[comment.Post]) + 1)
	comment.CreatedAt = timeToString(time.Now())
	s.comments[comment.Post][comment.Id] = *comment
	return nil
}

func (s *stubStorage) EditComment(comment *entities.Comment) error {
	comments, ok := s.comments[comment.Post]
	if !ok {
		return storage.ErrPostNotFound
	}
	found, ok := comments[comment.Id]
	if !ok {
		return storage.ErrCommentNotFound
	}
	if comment.Author == "" {
		comment.Author = found.Author
	}
	if comment.Content == "" {
		comment.Content = found.Content
	}
	s.commentEditCalls = append(s.commentEditCalls, fmt.Sprintf("%+v", comment))
	return nil
}

func (s *stubStorage) DeleteComment(post, id string) error {
	comments, ok := s.comments[post]
	if !ok {
		return ErrPostNotFound
	}
	delete(comments, id)
	return nil
}

func (s *stubStorage) CreateCustomer(customer *entities.Customer) error {
	if customer.Name == "" {
		return ErrMissingCustomerFields
	}
	customer.Id = strconv.Itoa(len(s.customers) + 1)
	customer.CreatedAt = timeToString(time.Now())
	s.customers[customer.Id] = *customer
	return nil
}

var errFoo = errors.New("some error")

type stubFailingStorage struct {
	posts map[string]entities.Post
}

func (s *stubFailingStorage) GetPost(id string) (*entities.Post, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) GetPosts() ([]entities.Post, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) CreatePost(post *entities.Post) error {
	return errFoo
}

func (s *stubFailingStorage) EditPost(post *entities.Post) error {
	return errFoo
}

func (s *stubFailingStorage) DeletePost(id string) error {
	return errFoo
}

func (s *stubFailingStorage) GetComments(post string) ([]entities.Comment, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) GetComment(post, id string) (*entities.Comment, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) CreateComment(comment *entities.Comment) error {
	return errFoo
}

func (s *stubFailingStorage) EditComment(comment *entities.Comment) error {
	return errFoo
}

func (s *stubFailingStorage) DeleteComment(post, id string) error {
	return errFoo
}

func (s *stubFailingStorage) CreateCustomer(customer *entities.Customer) error {
	return errFoo
}

type mockStorage struct {
	GetPostFunc        func(id string) (*entities.Post, error)
	GetPostsFunc       func() ([]entities.Post, error)
	CreatePostFunc     func(post *entities.Post) error
	EditPostFunc       func(post *entities.Post) error
	DeletePostFunc     func(id string) error
	GetCommentsFunc    func(post string) ([]entities.Comment, error)
	GetCommentFunc     func(post, id string) (*entities.Comment, error)
	CreateCommentFunc  func(comment *entities.Comment) error
	EditCommentFunc    func(comment *entities.Comment) error
	DeleteCommentFunc  func(post, id string) error
	CreateCustomerFunc func(customer *entities.Customer) error
}

func (s *mockStorage) GetPost(id string) (*entities.Post, error) {
	return s.GetPostFunc(id)
}

func (s *mockStorage) GetPosts() ([]entities.Post, error) {
	return s.GetPostsFunc()
}

func (s *mockStorage) CreatePost(post *entities.Post) error {
	return s.CreatePostFunc(post)
}

func (s *mockStorage) EditPost(post *entities.Post) error {
	return s.EditPostFunc(post)
}

func (s *mockStorage) DeletePost(id string) error {
	return s.DeletePostFunc(id)
}

func (s *mockStorage) GetComments(post string) ([]entities.Comment, error) {
	return s.GetCommentsFunc(post)
}

func (s *mockStorage) GetComment(post, id string) (*entities.Comment, error) {
	return s.GetCommentFunc(post, id)
}

func (s *mockStorage) CreateComment(comment *entities.Comment) error {
	return s.CreateCommentFunc(comment)
}

func (s *mockStorage) EditComment(comment *entities.Comment) error {
	return s.EditCommentFunc(comment)
}

func (s *mockStorage) DeleteComment(post, id string) error {
	return s.DeleteCommentFunc(post, id)
}

func (s *mockStorage) CreateCustomer(customer *entities.Customer) error {
	return s.CreateCustomerFunc(customer)
}

func assertStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("did not get correct status, got %d but want %d", response.Code, want)
	}
}

func assertGotPost(t testing.TB, got, want entities.Post) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong post received, got %v but want %v", got, want)
	}
}

func assertGotError(t testing.TB, got Error, want *Error) {
	t.Helper()

	if !reflect.DeepEqual(got, *want) {
		t.Errorf("got error %q, but want %q", got, *want)
	}
}

func assertContains[T any](t testing.TB, list []T, needle T) {
	t.Helper()
	contains := false
	for _, n := range list {
		if reflect.DeepEqual(n, needle) {
			contains = true
			break
		}
	}
	if !contains {
		t.Errorf("expected %v to contain %v but it didn't", list, needle)
	}
}
