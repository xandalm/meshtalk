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
	ErrPostNotFoundMessage          = "there is no such post here"
	ErrUnsupportedPostMessage       = "unsupported data to parse as post"
	ErrMissingPostFieldsMessage     = "missing post fields (title, content and author are required)"
	ErrUnsupportedCommentMessage    = "unsupported data to parse as comment"
	ErrMissingCommentFieldsMessage  = "missing comment fields (content and author are required)"
	ErrCommentNotFoundMessage       = "there is no such comment here"
	ErrUnsupportedCustomerMessage   = "unsupported data to parse as customer"
	ErrMissingCustomerFieldsMessage = "missing customer fields (name is required)"
	ErrCustomerNotFoundMessage      = "there is no such customer here"
)

var (
	ErrPostNotFound          = NewError("ERR_POST_NOT_FOUND", ErrPostNotFoundMessage)
	ErrUnsupportedPost       = NewError("ERR_UNSUPPORTED_POST", ErrUnsupportedPostMessage)
	ErrMissingPostFields     = NewError("ERR_MISSING_POST_FIELDS", ErrMissingPostFieldsMessage)
	ErrCommentNotFound       = NewError("ERR_COMMENT_NOT_FOUND", ErrCommentNotFoundMessage)
	ErrUnsupportedComment    = NewError("ERR_UNSUPPORTED_COMMENT", ErrUnsupportedCommentMessage)
	ErrMissingCommentFields  = NewError("ERR_MISSING_COMMENT_FIELDS", ErrMissingCommentFieldsMessage)
	ErrUnsupportedCustomer   = NewError("ERR_UNSUPPORTED_CUSTOMER", ErrUnsupportedCustomerMessage)
	ErrMissingCustomerFields = NewError("ERR_MISSING_CUSTOMER_FIELDS", ErrMissingCustomerFieldsMessage)
	ErrCustomerNotFound      = NewError("ERR_CUSTOMER_NOT_FOUND", ErrCustomerNotFoundMessage)

	mErrors = map[error]*Error{
		storage.ErrPostNotFound:          ErrPostNotFound,
		storage.ErrMissingPostFields:     ErrMissingPostFields,
		storage.ErrCommentNotFound:       ErrCommentNotFound,
		storage.ErrMissingCommentFields:  ErrMissingCommentFields,
		storage.ErrCustomerNotFound:      ErrCustomerNotFound,
		storage.ErrMissingCustomerFields: ErrMissingCustomerFields,
	}
)

type Server struct {
	storage storage.Storage
	router  *router.Router
	to      time.Duration
}

func NewServer(storage storage.Storage) *Server {
	s := &Server{
		storage: storage,
		router:  &router.Router{},
		to:      time.Minute,
	}

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

func (s *Server) writeResponse(w http.ResponseWriter, data any, err error) {
	if err != nil {
		if e, ok := mErrors[err]; ok {
			err = e
		}
		switch err {
		case ErrPostNotFound,
			ErrCommentNotFound,
			ErrCustomerNotFound:
			w.WriteHeader(http.StatusNotFound)
		case ErrMissingPostFields,
			ErrMissingCommentFields,
			ErrMissingCustomerFields,
			ErrUnsupportedPost,
			ErrUnsupportedComment,
			ErrUnsupportedCustomer:
			w.WriteHeader(http.StatusBadRequest)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	writeJSON(
		w,
		ResponseModel{
			Data:  data,
			Error: err,
		},
	)
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

func (s *Server) createCustomerHandler(w router.ResponseWriter, r *router.Request) {
	var customer entities.Customer
	if err := r.ParseBodyInto(&customer); err != nil {
		s.writeResponse(w, nil, ErrUnsupportedCustomer)
		return
	}
	if err := s.storage.CreateCustomer(&customer); err != nil {
		s.writeResponse(w, nil, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	s.writeResponse(
		w,
		customer,
		nil,
	)
}

func (s *Server) getCustomerHandler(w router.ResponseWriter, r *router.Request) {
	customerId := r.Params()["id"]

	found, err := s.storage.GetCustomer(customerId)
	if err != nil {
		s.writeResponse(w, nil, err)
	}
	if found != nil {
		s.writeResponse(w, found, nil)
		return
	}
	s.writeResponse(w, nil, ErrCustomerNotFound)
}

func (s *Server) editCustomerHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	var edit entities.CustomerInEditting
	if err := r.ParseBodyInto(&edit); err != nil {
		s.writeResponse(w, nil, ErrUnsupportedCustomer)
		return
	}
	edit.Id = params["id"]

	if _, err := s.storage.EditCustomer(edit); err != nil {
		s.writeResponse(w, nil, err)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCustomerHandler(w router.ResponseWriter, r *router.Request) {
	customerId := r.Params()["id"]

	if err := s.storage.DeleteCustomer(customerId); err != nil {
		s.writeResponse(w, nil, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) createPostHandler(w router.ResponseWriter, r *router.Request) {
	var post entities.Post
	err := r.ParseBodyInto(&post)
	if err != nil {
		s.writeResponse(w, nil, ErrUnsupportedPost)
		return
	}

	if err := s.storage.CreatePost(&post); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponse(
		w,
		post,
		nil,
	)
}

func (s *Server) getPostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]

	found, err := s.storage.GetPost(postId)
	if err != nil {
		s.writeResponse(w, nil, err)
		return
	}
	if found != nil {
		s.writeResponse(w, *found, nil)
		return
	}
	s.writeResponse(w, nil, ErrPostNotFound)
}

func (s *Server) getPostsHandler(w router.ResponseWriter, _ *router.Request) {
	posts, err := s.storage.GetPosts()
	if err != nil {
		s.writeResponse(w, nil, err)
		return
	}
	s.writeResponse(w, posts, nil)
}

func (s *Server) editPostHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	var edit entities.PostInEditting
	err := r.ParseBodyInto(&edit)
	if err != nil {
		s.writeResponse(w, nil, ErrUnsupportedPost)
		return
	}
	edit.Id = params["id"]

	if _, err := s.storage.EditPost(edit); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deletePostHandler(w router.ResponseWriter, r *router.Request) {
	postId := r.Params()["id"]
	if err := s.storage.DeletePost(postId); err != nil {
		s.writeResponse(w, nil, err)
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
			s.writeResponse(w, nil, err)
			return
		}
		s.writeResponse(w, comments, nil)
		return
	}
	if !hasComment {
		comments, err := s.storage.GetComments(post[0])
		if err != nil {
			s.writeResponse(w, nil, err)
			return
		}
		s.writeResponse(w, comments, nil)
		return
	}
	comments, err := s.storage.GetComment(post[0], comment[0])
	if err != nil {
		s.writeResponse(w, nil, err)
		return
	}
	s.writeResponse(w, []entities.Comment{*comments}, nil)
}

func (s *Server) getPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	pid := params["pid"]
	cid := params["cid"]

	post, err := s.storage.GetPost(pid)
	if err != nil {
		s.writeResponse(w, nil, err)
		return
	}
	if post == nil {
		s.writeResponse(w, nil, ErrPostNotFound)
		return
	}

	comment, err := s.storage.GetComment(pid, cid)
	if err != nil {
		s.writeResponse(w, nil, err)
		return
	}
	if comment == nil {
		s.writeResponse(w, nil, ErrCommentNotFound)
		return
	}
	s.writeResponse(w, comment, nil)
}

func (s *Server) getPostCommentsHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	pid := params["pid"]

	comments, err := s.storage.GetComments(pid)
	if err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	s.writeResponse(w, comments, nil)
}

func (s *Server) createPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	pid := r.Params()["pid"]

	var comment entities.Comment
	err := r.ParseBodyInto(&comment)

	if err != nil {
		s.writeResponse(w, nil, ErrUnsupportedComment)
		return
	}

	comment.Post = pid

	if err := s.storage.CreateComment(&comment); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	s.writeResponse(w, comment, nil)
}

func (s *Server) editPostCommentHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	var edit entities.CommentInEditting
	if err := r.ParseBodyInto(&edit); err != nil {
		s.writeResponse(w, nil, ErrUnsupportedComment)
		return
	}

	edit.Post = params["pid"]
	edit.Id = params["cid"]

	if _, err := s.storage.EditComment(edit); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCommentHandler(w router.ResponseWriter, r *router.Request) {
	params := r.Params()

	if err := s.storage.DeleteComment(params["pid"], params["cid"]); err != nil {
		s.writeResponse(w, nil, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w io.Writer, s any) error {
	return json.NewEncoder(w).Encode(s)
}
