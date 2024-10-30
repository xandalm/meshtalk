package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"meshtalk/domain/entities"
	"meshtalk/domain/services/storage"
	"net/http"
	"os"
	"path/filepath"
	"time"

	router "github.com/xandalm/go-router"
	"github.com/xandalm/go-session"
	"github.com/xandalm/go-session/filesystem"
)

type Error struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewError(name, message string) *Error {
	return &Error{
		Name:    name,
		Message: message,
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf(`[%s "%s"]`, e.Name, e.Message)
}

type ResponseModel struct {
	Data  any `json:"data,omitempty"`
	Error any `json:"error,omitempty"`
}

type CustomerInput struct {
	Tag      *string `json:"tag"`
	Name     *string `json:"name"`
	Password *string `json:"password"`
}

type PostInput struct {
	Title   *string `json:"title"`
	Content *string `json:"content"`
}

type CommentInput struct {
	Post    *string `json:"post"`
	Content *string `json:"content"`
}

const (
	ErrUnsupportedContentTypeMessage = "content-type header property is required and must be application/json"
	ErrUnsupportedPostMessage        = "unsupported data to parse as post"
	ErrMissingPostFieldsMessage      = "missing post fields (title, content and author are required)"
	ErrUnsupportedCommentMessage     = "unsupported data to parse as comment"
	ErrMissingCommentFieldsMessage   = "missing comment fields (content and author are required)"
	ErrUnsupportedCustomerMessage    = "unsupported data to parse as customer"
	ErrMissingCustomerFieldsMessage  = "missing customer fields (name is required)"
	ErrNothingToUpdateMessage        = "no changes to be updated"
	ErrNonexistentCustomerMessage    = "the given author doesn't exist"
)

var (
	ErrUnsupportedContentType = NewError("ERR_UNSUPPORTED_CONTENT_TYPE", ErrUnsupportedContentTypeMessage)
	ErrUnsupportedPost        = NewError("ERR_UNSUPPORTED_POST", ErrUnsupportedPostMessage)
	ErrMissingPostFields      = NewError("ERR_MISSING_POST_FIELDS", ErrMissingPostFieldsMessage)
	ErrUnsupportedComment     = NewError("ERR_UNSUPPORTED_COMMENT", ErrUnsupportedCommentMessage)
	ErrMissingCommentFields   = NewError("ERR_MISSING_COMMENT_FIELDS", ErrMissingCommentFieldsMessage)
	ErrUnsupportedCustomer    = NewError("ERR_UNSUPPORTED_CUSTOMER", ErrUnsupportedCustomerMessage)
	ErrMissingCustomerFields  = NewError("ERR_MISSING_CUSTOMER_FIELDS", ErrMissingCustomerFieldsMessage)
	ErrNothingToUpdate        = NewError("ERR_NOTHING_TO_UPDATE", ErrNothingToUpdateMessage)
	ErrNonexistentAuthor      = NewError("ERR_NONEXISTENT_AUTHOR", ErrNonexistentCustomerMessage)

	overwrittenErrors = map[error]*Error{
		storage.ErrPostNotFound:          nil,
		storage.ErrMissingPostFields:     ErrMissingPostFields,
		storage.ErrCommentNotFound:       nil,
		storage.ErrMissingCommentFields:  ErrMissingCommentFields,
		storage.ErrCustomerNotFound:      nil,
		storage.ErrMissingCustomerFields: ErrMissingCustomerFields,
		storage.ErrUnrecognizedAuthor:    ErrNonexistentAuthor,
	}
)

const sessionCookieName = "SESSION_ID"

func SessionCookieName() string {
	return sessionCookieName
}

type Server struct {
	storage storage.Storage
	router  *router.Router
	to      time.Duration
}

func NewServer(storage storage.Storage) *Server {
	session.Config(sessionCookieName, int64(24*time.Hour/time.Second), session.DefaultSessionFactory, filesystem.NewStorage(filepath.Join(os.TempDir(), "MeshtalkSessions"), ""))
	s := &Server{
		storage: storage,
		router:  &router.Router{},
		to:      time.Minute,
	}

	s.router.UseFunc(func(w router.ResponseWriter, r *router.Request, next router.NextMiddlewareCaller) {
		if r.Body == nil {
			next()
			return
		}
		_, err := r.Body.Read(make([]byte, 0))
		if err != io.EOF && r.Header.Get("Content-Type") != "application/json" {
			next(ErrUnsupportedContentType)
			return
		}
		next()
	})

	s.router.PostFunc("/login", s.login)

	nsCustomers := s.router.Namespace("customers")
	nsCustomers.GetFunc("/{customer}", s.getCustomerHandler)
	nsCustomers.PutFunc("/{customer}", s.editCustomerHandler)
	nsCustomers.DeleteFunc("/{customer}", s.deleteCustomerHandler)
	nsCustomers.PostFunc(s.createCustomerHandler)

	nsPosts := s.router.Namespace("posts")
	nsPosts.GetFunc("/{post}", s.getPostHandler)
	nsPosts.PutFunc("/{post}", s.editPostHandler)
	nsPosts.DeleteFunc("/{post}", s.deletePostHandler)
	nsPosts.GetFunc(s.getPostsHandler)
	nsPosts.PostFunc(s.createPostHandler)

	nsPostsComments := nsPosts.Namespace("{post}/comments")
	nsPostsComments.GetFunc("/{comment}", s.getPostCommentHandler)
	nsPostsComments.PutFunc("/{comment}", s.editPostCommentHandler)
	nsPostsComments.DeleteFunc("/{comment}", s.deleteCommentHandler)
	nsPostsComments.GetFunc(s.getPostCommentsHandler)
	nsPostsComments.PostFunc(s.createPostCommentHandler)

	s.router.GetFunc("/comments", s.getCommentsHandler)

	s.router.UseFunc(func(w router.ResponseWriter, r *router.Request, err error) {
		s.writeResponseModelWithError(w, err)
	})

	return s
}

func (s *Server) SetTimeout(duration time.Duration) error {
	if duration < time.Second {
		return errors.New("timeout duration must be greater than 1s")
	}
	s.to = duration
	return nil
}

func (s *Server) writeResponseModelWithoutError(w router.ResponseWriter, data any, status ...int) {
	if data == nil {
		return
	}
	if len(status) > 0 {
		w.SetStatus(status[0])
	}
	w.SendJSON(ResponseModel{
		Data: data,
	})
}

func (s *Server) writeResponseModelWithError(w router.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch err {
	case storage.ErrPostNotFound,
		storage.ErrCommentNotFound,
		storage.ErrCustomerNotFound:
		w.SetStatus(http.StatusNotFound)
	case storage.ErrMissingPostFields,
		storage.ErrMissingCommentFields,
		storage.ErrMissingCustomerFields,
		storage.ErrUnrecognizedAuthor,
		ErrUnsupportedPost,
		ErrUnsupportedComment,
		ErrUnsupportedCustomer,
		ErrMissingCustomerFields,
		ErrMissingPostFields,
		ErrMissingCommentFields,
		ErrNothingToUpdate,
		ErrUnsupportedContentType:
		w.SetStatus(http.StatusBadRequest)
	default:
		w.SetStatus(http.StatusInternalServerError)
	}
	// Maybe overwrite the error
	if e, ok := overwrittenErrors[err]; ok {
		err = e
	}
	if err != nil {
		w.SendJSON(ResponseModel{
			Error: err,
		})
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())

	time.AfterFunc(s.to, cancel)
	r = r.WithContext(ctx)

	c := make(chan struct{})
	go func() {
		s.router.ServeHTTP(w, r)
		close(c)
	}()

	select {
	case <-ctx.Done():
		w.WriteHeader(http.StatusRequestTimeout)
	case <-c:
		return
	}
}

func (s *Server) isLogged(sess session.Session) bool {
	customer := sess.Get("customer")
	return customer != nil
}

func (s *Server) login(w router.ResponseWriter, r *router.Request) {

	var credentials struct {
		Tag      string `json:"tag"`
		Password string `json:"password"`
	}

	r.ParseBodyInto(&credentials)

	customer, err := s.storage.GetCustomerByTag(credentials.Tag)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	if customer == nil && os.Getenv("GO_ENV") == "DEVELOPMENT" {
		// Backdoor allowing test routines on dev environment
		customer = entities.NewCustomer(credentials.Tag, credentials.Tag, credentials.Password)
		if err := s.storage.CreateCustomer(customer); err != nil {
			s.writeResponseModelWithError(w, err)
			return
		}
	}

	if customer == nil || customer.Password != credentials.Password {
		w.SetStatus(http.StatusForbidden)
		return
	}

	sess := session.Start(w.(http.ResponseWriter), r.Request)

	if err := sess.Set("customer", *customer); err != nil {
		session.Destroy(w.(http.ResponseWriter), r.Request)
		w.SetStatus(http.StatusInternalServerError)
	}
}

func (s *Server) createCustomerHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	var input CustomerInput
	if err := r.ParseBodyInto(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedCustomer)
		return
	}
	if input.Tag == nil || input.Name == nil || input.Password == nil {
		s.writeResponseModelWithError(w, ErrMissingCustomerFields)
		return
	}

	customer := entities.NewCustomer(*input.Tag, *input.Name, *input.Password)

	if err := s.storage.CreateCustomer(customer); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	s.writeResponseModelWithoutError(
		w,
		customer,
		http.StatusCreated,
	)
}

func (s *Server) getCustomerHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	customer := r.Params()["customer"]

	found, err := s.storage.GetCustomer(customer)
	if err != nil {
		s.writeResponseModelWithError(w, err)
	}
	if found == nil {
		w.SetStatus(http.StatusNotFound)
		return
	}
	s.writeResponseModelWithoutError(w, found)
}

func (s *Server) editCustomerHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	customerId := r.Params()["customer"]

	if c, ok := sess.Get("customer").(entities.Customer); !ok || c.Id != customerId {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	customer, err := s.storage.GetCustomer(customerId)
	if err != nil {
		s.writeResponseModelWithError(w, err)
	}
	if customer == nil {
		w.SetStatus(http.StatusNotFound)
		return
	}

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedCustomer)
		return
	}

	unchanged := true
	if name, ok := input["Name"]; ok {
		customer.Name = name.(string)
		unchanged = false
	}
	if password, ok := input["Password"]; ok {
		customer.Password = password.(string)
		unchanged = false
	}

	if unchanged {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}

	if err := s.storage.EditCustomer(customer); err != nil {
		s.writeResponseModelWithError(w, err)
	}

	w.SetStatus(http.StatusNoContent)
}

func (s *Server) deleteCustomerHandler(w router.ResponseWriter, r *router.Request) {
	session := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(session) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	customer := r.Params()["customer"]

	if c, ok := session.Get("customer").(entities.Customer); !ok || c.Id != customer {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	if err := s.storage.DeleteCustomer(customer); err != nil {
		s.writeResponseModelWithError(w, err)
	}
	w.SetStatus(http.StatusNoContent)
}

func (s *Server) createPostHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	author := sess.Get("customer").(entities.Customer)

	var input PostInput
	err := r.ParseBodyInto(&input)
	if err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedPost)
		return
	}
	if input.Title == nil || input.Content == nil {
		s.writeResponseModelWithError(w, ErrMissingPostFields)
		return
	}

	post := entities.NewPost(*input.Title, *input.Content, &author)

	if err := s.storage.CreatePost(post); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	s.writeResponseModelWithoutError(
		w,
		post,
		http.StatusCreated,
	)
}

func (s *Server) getPostHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	post := r.Params()["post"]

	found, err := s.storage.GetPost(post)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	if found == nil {
		w.SetStatus(http.StatusNotFound)
		return
	}
	s.writeResponseModelWithoutError(w, found)
}

func (s *Server) getPostsHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	posts, err := s.storage.GetPosts()
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	s.writeResponseModelWithoutError(w, posts)
}

func (s *Server) editPostHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	postId := r.Params()["post"]

	post, err := s.storage.GetPost(postId)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	if post == nil {
		w.SetStatus(http.StatusNotFound)
		return
	}

	var input map[string]any

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedPost)
		return
	}

	unchanged := true
	if title, ok := input["Title"]; ok {
		post.Title = title.(string)
		unchanged = false
	}
	if content, ok := input["Content"]; ok {
		post.Content = content.(string)
		unchanged = false
	}

	if unchanged {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}

	if err := s.storage.EditPost(post); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.SetStatus(http.StatusNoContent)
}

func (s *Server) deletePostHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	post := r.Params()["post"]
	if err := s.storage.DeletePost(post); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	w.SetStatus(http.StatusNoContent)
}

func (s *Server) getCommentsHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	query := r.URL.Query()

	post, hasPost := query["post"]
	comment, hasComment := query["comment"]

	if !hasPost {
		comments, err := s.storage.GetComments("")
		if err != nil {
			s.writeResponseModelWithError(w, err)
			return
		}
		s.writeResponseModelWithoutError(w, comments)
		return
	}
	if !hasComment {
		comments, err := s.storage.GetComments(post[0])
		if err != nil {
			s.writeResponseModelWithError(w, err)
			return
		}
		s.writeResponseModelWithoutError(w, comments)
		return
	}
	comments, err := s.storage.GetComment(post[0], comment[0])
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	s.writeResponseModelWithoutError(w, []entities.Comment{*comments})
}

func (s *Server) getPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	params := r.Params()

	post := params["post"]
	comment := params["comment"]

	found, err := s.storage.GetComment(post, comment)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	if found == nil {
		w.SetStatus(http.StatusNotFound)
		return
	}

	s.writeResponseModelWithoutError(w, found)
}

func (s *Server) getPostCommentsHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	params := r.Params()

	post := params["post"]

	comments, err := s.storage.GetComments(post)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	s.writeResponseModelWithoutError(w, comments)
}

func (s *Server) createPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	author := sess.Get("customer").(entities.Customer)

	post := r.Params()["post"]

	var input CommentInput
	if err := r.ParseBodyInto(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedComment)
		return
	}
	input.Post = &post
	if input.Content == nil {
		s.writeResponseModelWithError(w, ErrMissingCommentFields)
		return
	}

	comment := entities.NewComment(*input.Post, *input.Content, &author)

	if err := s.storage.CreateComment(comment); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.SetStatus(http.StatusCreated)
	s.writeResponseModelWithoutError(w, comment)
}

func (s *Server) editPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	params := r.Params()
	postId := params["post"]
	commentId := params["comment"]

	comment, err := s.storage.GetComment(postId, commentId)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	if comment == nil {
		w.SetStatus(http.StatusNotFound)
		return
	}

	var input map[string]any
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedComment)
		return
	}

	if content, ok := input["Content"]; ok {
		comment.Content = content.(string)
	} else {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}

	if err := s.storage.EditComment(comment); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.SetStatus(http.StatusNoContent)
}

func (s *Server) deleteCommentHandler(w router.ResponseWriter, r *router.Request) {
	sess := session.Start(w.(http.ResponseWriter), r.Request)
	if !s.isLogged(sess) {
		w.SetStatus(http.StatusUnauthorized)
		return
	}

	params := r.Params()

	if err := s.storage.DeleteComment(params["post"], params["comment"]); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.SetStatus(http.StatusNoContent)
}
