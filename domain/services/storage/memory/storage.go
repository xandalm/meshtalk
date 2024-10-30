package memory

import (
	"meshtalk/domain/entities"
	"meshtalk/domain/services/storage"
	"strconv"
	"time"
)

type Storage struct {
	customers_pk  int
	customers     map[string]entities.Customer
	customers_tag map[string]string
	posts_pk      int
	posts         map[string]entities.Post
	comments      map[string]map[string]entities.Comment
}

func NewStorage() *Storage {
	return &Storage{
		1,
		map[string]entities.Customer{},
		map[string]string{},
		1,
		map[string]entities.Post{},
		map[string]map[string]entities.Comment{},
	}
}

func (s *Storage) GetPost(id string) (*entities.Post, error) {
	found, ok := s.posts[id]
	if !ok || found.DeletedAt != "" {
		return nil, nil
	}
	return &entities.Post{
		Id:        found.Id,
		Title:     found.Title,
		Content:   found.Content,
		Author:    found.Author,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
	}, nil
}

func (s *Storage) GetPosts() ([]entities.Post, error) {
	posts := make([]entities.Post, 0, len(s.posts))

	for _, p := range s.posts {
		if p.DeletedAt == "" {
			posts = append(posts, p)
		}
	}

	return posts, nil
}

func timeToString(t time.Time) string {
	b, _ := t.UTC().MarshalText()
	str := string(b)
	return str
}

func (s *Storage) CreatePost(post *entities.Post) error {

	if post.Title == "" || post.Content == "" || post.Author == nil {
		return storage.ErrMissingPostFields
	}

	post.Id = strconv.Itoa(s.posts_pk)
	post.CreatedAt = timeToString(time.Now())
	s.posts[post.Id] = *post
	s.posts_pk++
	return nil
}

func (s *Storage) EditPost(post *entities.Post) error {
	found, ok := s.posts[post.Id]
	if !ok || found.DeletedAt != "" {
		return storage.ErrPostNotFound
	}

	found.Title = post.Title
	found.Content = post.Content

	found.UpdatedAt = timeToString(time.Now())
	s.posts[found.Id] = found
	*post = found
	return nil
}

func (s *Storage) DeletePost(id string) error {
	post, ok := s.posts[id]
	if !ok {
		return storage.ErrPostNotFound
	}
	post.DeletedAt = timeToString(time.Now())
	s.posts[id] = post
	return nil
}

func (s *Storage) GetComments(post string) ([]entities.Comment, error) {
	if _, ok := s.posts[post]; ok {
		var res []entities.Comment
		for _, comments := range s.comments {
			for _, comment := range comments {
				if comment.DeletedAt == "" {
					res = append(res, comment)
				}
			}
		}
		return res, nil
	}
	return nil, storage.ErrPostNotFound
}

func (s *Storage) GetComment(post, comment string) (*entities.Comment, error) {
	if _, ok := s.posts[post]; ok {
		found, ok := s.comments[post][comment]
		if !ok || found.DeletedAt != "" {
			return nil, nil
		}
		return &entities.Comment{
			Id:        found.Id,
			Post:      found.Post,
			Content:   found.Content,
			Author:    found.Author,
			CreatedAt: found.CreatedAt,
			UpdatedAt: found.UpdatedAt,
		}, nil
	}
	return nil, storage.ErrPostNotFound
}

func (s *Storage) CreateComment(c *entities.Comment) error {

	if c.Content == "" || c.Author == nil {
		return storage.ErrMissingCommentFields
	}

	if _, hasPost := s.posts[c.Post]; !hasPost {
		return storage.ErrPostNotFound
	}

	_, hasComments := s.comments[c.Post]
	if !hasComments {
		s.comments[c.Post] = make(map[string]entities.Comment)
	}
	c.Id = strconv.Itoa(len(s.comments[c.Post]) + 1)
	c.CreatedAt = timeToString(time.Now())
	s.comments[c.Post][c.Id] = *c
	return nil
}

func (s *Storage) EditComment(comment *entities.Comment) error {
	if comments, hasComments := s.comments[comment.Post]; hasComments {
		if found, hasComment := comments[comment.Id]; hasComment && found.DeletedAt == "" {

			found.Content = comment.Content

			found.UpdatedAt = timeToString(time.Now())
			comments[found.Id] = found
			*comment = found
			return nil
		}
	}
	return storage.ErrCommentNotFound
}

func (s *Storage) DeleteComment(post, id string) error {
	if comments, hasComments := s.comments[post]; hasComments {
		comment, hasComment := comments[id]
		if !hasComment {
			return storage.ErrCommentNotFound
		}
		comment.DeletedAt = timeToString(time.Now())
		comments[id] = comment
		return nil
	}
	return storage.ErrPostNotFound
}

func (s *Storage) CreateCustomer(c *entities.Customer) error {

	if c.Name == "" {
		return storage.ErrMissingCustomerFields
	}

	c.Id = strconv.Itoa(s.customers_pk)
	c.CreatedAt = timeToString(time.Now())
	s.customers[c.Id] = *c
	s.customers_pk++
	return nil
}

func (s *Storage) GetCustomer(id string) (*entities.Customer, error) {
	found, ok := s.customers[id]
	if !ok || found.DeletedAt != "" {
		return nil, nil
	}
	return &entities.Customer{
		Id:        found.Id,
		Name:      found.Name,
		CreatedAt: found.CreatedAt,
		UpdatedAt: found.UpdatedAt,
	}, nil
}

func (s *Storage) GetCustomerByTag(tag string) (*entities.Customer, error) {
	id, ok := s.customers_tag[tag]
	if !ok {
		return nil, nil
	}
	return s.GetCustomer(id)
}

func (s *Storage) EditCustomer(customer *entities.Customer) error {
	found, ok := s.customers[customer.Id]
	if !ok || found.DeletedAt != "" {
		return storage.ErrCustomerNotFound
	}
	found.Name = customer.Name

	found.UpdatedAt = timeToString(time.Now())
	s.customers[found.Id] = found
	*customer = found
	return nil
}

func (s *Storage) DeleteCustomer(id string) error {
	customer, ok := s.customers[id]
	if !ok {
		return storage.ErrCustomerNotFound
	}
	customer.DeletedAt = timeToString(time.Now())
	s.customers[id] = customer
	return nil
}
