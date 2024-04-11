package httpserver

import (
	"errors"
	"fmt"
	"meshtalk/domain/entities"
	"meshtalk/domain/services/storage"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type stubStorage struct {
	customers         map[string]entities.Customer
	posts             map[string]entities.Post
	comments          map[string]map[string]entities.Comment
	customerEditCalls []string
	postEditCalls     []string
	commentEditCalls  []string
}

func NewStubStorage() *stubStorage {
	return &stubStorage{
		map[string]entities.Customer{},
		map[string]entities.Post{},
		map[string]map[string]entities.Comment{},
		[]string{},
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
	if _, ok := s.customers[post.Author]; !ok {
		return storage.ErrUnrecognizedAuthor
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

func (s *stubStorage) EditPost(edit entities.PostInEditting) (*entities.Post, error) {
	_, ok := s.posts[edit.Id]
	if !ok {
		return nil, storage.ErrPostNotFound
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Id=%q", edit.Id))
	if edit.Title != nil {
		builder.WriteString(", ")
		builder.WriteString(fmt.Sprintf("Title=%q", *edit.Title))
	}
	if edit.Content != nil {
		builder.WriteString(", ")
		builder.WriteString(fmt.Sprintf("Content=%q", *edit.Content))
	}
	s.postEditCalls = append(s.postEditCalls, builder.String())
	return nil, nil
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
	if _, ok := s.customers[comment.Author]; !ok {
		return storage.ErrUnrecognizedAuthor
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

func (s *stubStorage) EditComment(edit entities.CommentInEditting) (*entities.Comment, error) {
	comments, ok := s.comments[edit.Post]
	if !ok {
		return nil, storage.ErrPostNotFound
	}
	if _, ok := comments[edit.Id]; !ok {
		return nil, storage.ErrCommentNotFound
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Post=%q", edit.Post))
	builder.WriteString(", ")
	builder.WriteString(fmt.Sprintf("Id=%q", edit.Id))
	if edit.Content != nil {
		builder.WriteString(", ")
		builder.WriteString(fmt.Sprintf("Content=%q", *edit.Content))
	}
	s.commentEditCalls = append(s.commentEditCalls, builder.String())
	return nil, nil
}

func (s *stubStorage) DeleteComment(post, id string) error {
	comments, ok := s.comments[post]
	if !ok {
		return storage.ErrPostNotFound
	}
	delete(comments, id)
	return nil
}

func (s *stubStorage) CreateCustomer(customer *entities.Customer) error {
	if customer.Tag == "" || customer.Name == "" || customer.Password == "" {
		return storage.ErrMissingCustomerFields
	}
	customer.Id = strconv.Itoa(len(s.customers) + 1)
	customer.CreatedAt = timeToString(time.Now())
	s.customers[customer.Id] = *customer
	return nil
}

func (s *stubStorage) GetCustomer(id string) (*entities.Customer, error) {
	found, ok := s.customers[id]
	if !ok {
		return nil, nil
	}
	return &entities.Customer{
		Id:        found.Id,
		Name:      found.Name,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
		DeletedAt: found.DeletedAt,
	}, nil
}

func (s *stubStorage) EditCustomer(edit entities.CustomerInEditting) (*entities.Customer, error) {
	if _, ok := s.customers[edit.Id]; !ok {
		return nil, storage.ErrCustomerNotFound
	}
	if (edit.Name == nil || *edit.Name == "") && (edit.Password == nil || *edit.Password == "") {
		return nil, storage.ErrMissingCustomerFields
	}
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Id=%q", edit.Id))
	if edit.Name != nil {
		builder.WriteString(", ")
		builder.WriteString(fmt.Sprintf("Name=%q", *edit.Name))
	}
	if edit.Password != nil {
		builder.WriteString(", ")
		builder.WriteString(fmt.Sprintf("Password=%q", *edit.Password))
	}
	s.customerEditCalls = append(s.customerEditCalls, builder.String())
	return nil, nil
}

func (s *stubStorage) DeleteCustomer(id string) error {
	delete(s.customers, id)
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

func (s *stubFailingStorage) EditPost(edit entities.PostInEditting) (*entities.Post, error) {
	return nil, errFoo
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

func (s *stubFailingStorage) EditComment(edit entities.CommentInEditting) (*entities.Comment, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) DeleteComment(post, id string) error {
	return errFoo
}

func (s *stubFailingStorage) CreateCustomer(customer *entities.Customer) error {
	return errFoo
}

func (s *stubFailingStorage) GetCustomer(id string) (*entities.Customer, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) EditCustomer(edit entities.CustomerInEditting) (*entities.Customer, error) {
	return nil, errFoo
}

func (s *stubFailingStorage) DeleteCustomer(id string) error {
	return errFoo
}

type mockStorage struct {
	GetPostFunc        func(id string) (*entities.Post, error)
	GetPostsFunc       func() ([]entities.Post, error)
	CreatePostFunc     func(post *entities.Post) error
	EditPostFunc       func(edit entities.PostInEditting) (*entities.Post, error)
	DeletePostFunc     func(id string) error
	GetCommentsFunc    func(post string) ([]entities.Comment, error)
	GetCommentFunc     func(post, id string) (*entities.Comment, error)
	CreateCommentFunc  func(comment *entities.Comment) error
	EditCommentFunc    func(edit entities.CommentInEditting) (*entities.Comment, error)
	DeleteCommentFunc  func(post, id string) error
	CreateCustomerFunc func(customer *entities.Customer) error
	GetCustomerFunc    func(id string) (*entities.Customer, error)
	EditCustomerFunc   func(edit entities.CustomerInEditting) (*entities.Customer, error)
	DeleteCustomerFunc func(id string) error
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

func (s *mockStorage) EditPost(edit entities.PostInEditting) (*entities.Post, error) {
	return s.EditPostFunc(edit)
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

func (s *mockStorage) EditComment(edit entities.CommentInEditting) (*entities.Comment, error) {
	return s.EditCommentFunc(edit)
}

func (s *mockStorage) DeleteComment(post, id string) error {
	return s.DeleteCommentFunc(post, id)
}

func (s *mockStorage) CreateCustomer(customer *entities.Customer) error {
	return s.CreateCustomerFunc(customer)
}

func (s *mockStorage) GetCustomer(id string) (*entities.Customer, error) {
	return s.GetCustomerFunc(id)
}

func (s *mockStorage) EditCustomer(edit entities.CustomerInEditting) (*entities.Customer, error) {
	return s.EditCustomerFunc(edit)
}

func (s *mockStorage) DeleteCustomer(id string) error {
	return s.DeleteCustomerFunc(id)
}

func assertStatus(t testing.TB, response *httptest.ResponseRecorder, want int) {
	t.Helper()

	if response.Code != want {
		t.Fatalf("did not get correct status, got %d but want %d", response.Code, want)
	}
}

func isEqual[T comparable](a, b T) bool {
	return reflect.DeepEqual(a, b)
}

func assertGotPost(t testing.TB, got, want entities.Post) {
	t.Helper()

	if !isEqual(got, want) {
		t.Errorf("wrong post received, got %v but want %v", got, want)
	}
}

func assertGotComment(t testing.TB, got, want entities.Comment) {
	t.Helper()

	if !isEqual(got, want) {
		t.Errorf("wrong post received, got %v but want %v", got, want)
	}
}

func assertGotCustomer(t testing.TB, got, want entities.Customer) {
	t.Helper()

	if !isEqual(got, want) {
		t.Errorf("wrong post received, got %v but want %v", got, want)
	}
}

func assertGotError(t testing.TB, got Error, want *Error) {
	t.Helper()

	if !isEqual(got, *want) {
		t.Fatalf("got error %q, but want %q", got, *want)
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
