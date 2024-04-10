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

type Server struct {
	storage        storage.Storage
	router         *router.Router
	sessionManager *session.Manager
	to             time.Duration
}

func NewServer(storage storage.Storage) *Server {
	sm := session.NewManager(
		session.NewProvider(
			filesystem.Storage(),
			session.SecondsAgeCheckerAdapter,
		),
		"SESSION_ID",
		60,
	)
	sm.GC()
	s := &Server{
		storage:        storage,
		router:         &router.Router{},
		sessionManager: sm,
		to:             time.Minute,
	}

	s.router.PostFunc("/login", s.login)

	s.router.GetFunc("/customers/{id}", s.getCustomerHandler)
	s.router.PutFunc("/customers/{id}", s.editCustomerHandler)
	s.router.DeleteFunc("/customers/{id}", s.deleteCustomerHandler)
	s.router.PostFunc("/customers", s.createCustomerHandler)

	s.router.GetFunc("/posts/{id}", s.getPostHandler)
	s.router.PutFunc("/posts/{id}", s.editPostHandler)
	s.router.DeleteFunc("/posts/{id}", s.deletePostHandler)
	s.router.GetFunc("/posts", s.getPostsHandler)
	s.router.PostFunc("/posts", s.createPostHandler)

	s.router.GetFunc("/posts/{pid}/comments/{cid}", s.getPostCommentHandler)
	s.router.PutFunc("/posts/{pid}/comments/{cid}", s.editPostCommentHandler)
	s.router.DeleteFunc("/posts/{pid}/comments/{cid}", s.deleteCommentHandler)
	s.router.GetFunc("/posts/{pid}/comments", s.getPostCommentsHandler)
	s.router.PostFunc("/posts/{pid}/comments", s.createPostCommentHandler)

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

func (s *Server) hasPermission(sess session.Session) bool {
	logged, ok := sess.Get("logged").(bool)
	return ok && logged
}

func (s *Server) login(w router.ResponseWriter, r *router.Request) {
	session := s.sessionManager.StartSession(w, r.Request)
	if err := session.Set("logged", true); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) createCustomerHandler(w router.ResponseWriter, r *router.Request) {
	var customer entities.Customer
	if err := r.ParseBodyInto(&customer); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedCustomer)
		return
	}
	if err := s.storage.CreateCustomer(&customer); err != nil {
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
	customerId := r.Params()["id"]

	found, err := s.storage.GetCustomer(customerId)
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
	params := r.Params()

	var edit entities.CustomerInEditting
	if err := r.ParseBodyInto(&edit); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedCustomer)
		return
	}
	edit.Id = params["id"]

	if edit.Name == nil {
		s.writeResponseModelWithError(w, ErrNothingToUpdate)
		return
	}

	if _, err := s.storage.EditCustomer(edit); err != nil {
		s.writeResponseModelWithError(w, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCustomerHandler(w router.ResponseWriter, r *router.Request) {
	customerId := r.Params()["id"]

	if err := s.storage.DeleteCustomer(customerId); err != nil {
		s.writeResponseModelWithError(w, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createPostHandler(w router.ResponseWriter, r *router.Request) {
	var post entities.Post
	err := r.ParseBodyInto(&post)
	if err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedPost)
		return
	}

	if err := s.storage.CreatePost(&post); err != nil {
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
	postId := r.Params()["id"]

	found, err := s.storage.GetPost(postId)
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
	if !s.hasPermission(session) {
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
	params := r.Params()

	var edit entities.PostInEditting
	err := r.ParseBodyInto(&edit)
	if err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedPost)
		return
	}
	edit.Id = params["id"]

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
	postId := r.Params()["id"]
	if err := s.storage.DeletePost(postId); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getCommentsHandler(w router.ResponseWriter, r *router.Request) {

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
	params := r.Params()

	pid := params["pid"]
	cid := params["cid"]

	found, err := s.storage.GetComment(pid, cid)
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
	params := r.Params()

	pid := params["pid"]

	comments, err := s.storage.GetComments(pid)
	if err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	s.writeResponseModelWithoutError(w, comments)
}

func (s *Server) createPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	pid := r.Params()["pid"]

	var comment entities.Comment
	err := r.ParseBodyInto(&comment)

	if err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedComment)
		return
	}

	comment.Post = pid

	if err := s.storage.CreateComment(&comment); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponseModelWithoutError(w, comment)
}

func (s *Server) editPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	var edit entities.CommentInEditting
	if err := r.ParseBodyInto(&edit); err != nil {
		s.writeResponseModelWithError(w, ErrUnsupportedComment)
		return
	}

	edit.Post = params["pid"]
	edit.Id = params["cid"]

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
	params := r.Params()

	if err := s.storage.DeleteComment(params["pid"], params["cid"]); err != nil {
		s.writeResponseModelWithError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
