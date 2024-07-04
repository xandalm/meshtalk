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
	"time"

	router "github.com/xandalm/go-router"
	"github.com/xandalm/go-session"
	"github.com/xandalm/go-session/memory"
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
	ErrUnsupportedPostMessage       = "unsupported data to parse as post"
	ErrMissingPostFieldsMessage     = "missing post fields (title, content and author are required)"
	ErrUnsupportedCommentMessage    = "unsupported data to parse as comment"
	ErrMissingCommentFieldsMessage  = "missing comment fields (content and author are required)"
	ErrUnsupportedCustomerMessage   = "unsupported data to parse as customer"
	ErrMissingCustomerFieldsMessage = "missing customer fields (name is required)"
	ErrNothingToUpdateMessage       = "no changes to be updated"
	ErrNonexistentCustomerMessage   = "the given author doesn't exist"
)

var (
	ErrUnsupportedPost       = NewError("ERR_UNSUPPORTED_POST", ErrUnsupportedPostMessage)
	ErrMissingPostFields     = NewError("ERR_MISSING_POST_FIELDS", ErrMissingPostFieldsMessage)
	ErrUnsupportedComment    = NewError("ERR_UNSUPPORTED_COMMENT", ErrUnsupportedCommentMessage)
	ErrMissingCommentFields  = NewError("ERR_MISSING_COMMENT_FIELDS", ErrMissingCommentFieldsMessage)
	ErrUnsupportedCustomer   = NewError("ERR_UNSUPPORTED_CUSTOMER", ErrUnsupportedCustomerMessage)
	ErrMissingCustomerFields = NewError("ERR_MISSING_CUSTOMER_FIELDS", ErrMissingCustomerFieldsMessage)
	ErrNothingToUpdate       = NewError("ERR_NOTHING_TO_UPDATE", ErrNothingToUpdateMessage)
	ErrNonexistentAuthor     = NewError("ERR_NONEXISTENT_AUTHOR", ErrNonexistentCustomerMessage)

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
	storage        storage.Storage
	router         *router.Router
	sessionManager *session.Manager
	to             time.Duration
}

func NewServer(storage storage.Storage) *Server {
	sm := session.NewManager(
		session.NewProvider(
			memory.Storage(),
			session.SecondsAgeCheckerAdapter,
		),
		sessionCookieName,
		int64(24*time.Hour/time.Second),
	)
	sm.GC()
	s := &Server{
		storage:        storage,
		router:         &router.Router{},
		sessionManager: sm,
		to:             time.Minute,
	}

	s.router.PostFunc("/login", s.login)

	customersNS := s.router.Namespace("customers")
	customersNS.GetFunc("/{customer}", s.getCustomerHandler)
	customersNS.PutFunc("/{customer}", s.editCustomerHandler)
	customersNS.DeleteFunc("/{customer}", s.deleteCustomerHandler)
	customersNS.PostFunc(s.createCustomerHandler)

	postsNS := s.router.Namespace("posts")
	postsNS.GetFunc("/{post}", s.getPostHandler)
	postsNS.PutFunc("/{post}", s.editPostHandler)
	postsNS.DeleteFunc("/{post}", s.deletePostHandler)
	postsNS.GetFunc(s.getPostsHandler)
	postsNS.PostFunc(s.createPostHandler)

	postsCommentsNS := postsNS.Namespace("{post}/comments")
	postsCommentsNS.GetFunc("/{comment}", s.getPostCommentHandler)
	postsCommentsNS.PutFunc("/{comment}", s.editPostCommentHandler)
	postsCommentsNS.DeleteFunc("/{comment}", s.deleteCommentHandler)
	postsCommentsNS.GetFunc(s.getPostCommentsHandler)
	postsCommentsNS.PostFunc(s.createPostCommentHandler)

	s.router.GetFunc("/comments", s.getCommentsHandler)

	return s
}

func (s *Server) SetTimeout(duration time.Duration) error {
	if duration < time.Second {
		return errors.New("timeout duration must be greater than 1s")
	}
	s.to = duration
	return nil
}

func (s *Server) writeResponseModelWithoutError(w http.ResponseWriter, data any, status ...int) {
	if data == nil {
		return
	}
	if len(status) > 0 {
		w.WriteHeader(status[0])
	}
	writeJSON(w, ResponseModel{
		Data: data,
	})
}

func (s *Server) writeResponseModelWithError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch err {
	case storage.ErrPostNotFound,
		storage.ErrCommentNotFound,
		storage.ErrCustomerNotFound:
		w.WriteHeader(http.StatusNotFound)
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
		ErrNothingToUpdate:
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	// Maybe overwrite the error
	if e, ok := overwrittenErrors[err]; ok {
		err = e
	}
	if err != nil {
		writeJSON(w, ResponseModel{
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
		// Backdoor on dev environment allowing test routines
		customer = entities.NewCustomer(credentials.Tag, credentials.Tag, credentials.Password)
		if err := s.storage.CreateCustomer(customer); err != nil {
			s.writeResponseModelWithError(w, err)
			return
		}
	}

	if customer == nil || customer.Password != credentials.Password {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	session := s.sessionManager.StartSession(w, r.Request)

	if err := session.Set("customer", *customer); err != nil {
		s.sessionManager.DestroySession(w, r.Request)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) createCustomerHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
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
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	customer := r.Params()["customer"]

	found, err := s.storage.GetCustomer(customer)
	if err != nil {
		s.writeResponseModelWithError(w, err)
	}
	if found == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.writeResponseModelWithoutError(w, found)
}

func (s *Server) editCustomerHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	customer := r.Params()["customer"]

	if c, ok := session.Get("customer").(entities.Customer); !ok || c.Id != customer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var input CustomerInput
	if err := r.ParseBodyInto(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedCustomer)
		return
	}

	edit := entities.CustomerInEditting{
		Name:     input.Name,
		Password: input.Password,
	}

	if edit.Name == nil && edit.Password == nil {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}
	edit.Id = customer

	if _, err := s.storage.EditCustomer(edit); err != nil {
		s.writeResponseModelWithError(w, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCustomerHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	customer := r.Params()["customer"]

	if c, ok := session.Get("customer").(entities.Customer); !ok || c.Id != customer {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if err := s.storage.DeleteCustomer(customer); err != nil {
		s.writeResponseModelWithError(w, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createPostHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	author := session.Get("customer").(entities.Customer)

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
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	post := r.Params()["post"]

	found, err := s.storage.GetPost(post)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	if found == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	s.writeResponseModelWithoutError(w, found)
}

func (s *Server) getPostsHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
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
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	params := r.Params()

	var input PostInput
	err := r.ParseBodyInto(&input)
	if err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedPost)
		return
	}

	edit := entities.PostInEditting{
		Title:   input.Title,
		Content: input.Content,
	}
	edit.Id = params["post"]

	if edit.Title == nil && edit.Content == nil {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}

	if _, err := s.storage.EditPost(edit); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deletePostHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	post := r.Params()["post"]
	if err := s.storage.DeletePost(post); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getCommentsHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
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
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
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
		w.WriteHeader(http.StatusNotFound)
		return
	}

	s.writeResponseModelWithoutError(w, found)
}

func (s *Server) getPostCommentsHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
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
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	author := session.Get("customer").(entities.Customer)

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

	w.WriteHeader(http.StatusCreated)
	s.writeResponseModelWithoutError(w, comment)
}

func (s *Server) editPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	params := r.Params()

	var input CommentInput
	if err := r.ParseBodyInto(&input); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedComment)
		return
	}

	edit := entities.CommentInEditting{
		Post:    params["post"],
		Id:      params["comment"],
		Content: input.Content,
	}

	if edit.Content == nil {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}

	if _, err := s.storage.EditComment(edit); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCommentHandler(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if !s.isLogged(session) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	params := r.Params()

	if err := s.storage.DeleteComment(params["post"], params["comment"]); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
