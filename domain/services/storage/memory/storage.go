package memory

import (
	"meshtalk/domain/entities"
	"meshtalk/domain/services/storage"
	"strconv"
	"time"
)

type Storage struct {
	posts_pk int
	posts    map[string]entities.Post
	comments map[string]map[string]entities.Comment
}

func NewStorage() *Storage {
	return &Storage{
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
		DeletedAt: found.DeletedAt,
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

func (s *Storage) StorePost(post *entities.Post) error {

	if post.Title == "" || post.Content == "" || post.Author == "" {
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

	if (found.Title != post.Title && post.Title == "") ||
		(found.Content != post.Content && post.Content == "") ||
		(found.Author != post.Author && post.Author == "") {
		return storage.ErrMissingPostFields
	}

	post.UpdatedAt = timeToString(time.Now())
	s.posts[post.Id] = *post
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
			DeletedAt: found.DeletedAt,
		}, nil
	}
	return nil, storage.ErrPostNotFound
}

func (s *Storage) StoreComment(c *entities.Comment) error {

	if c.Content == "" || c.Author == "" {
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

func (s *Storage) EditComment(c *entities.Comment) error {
	if comments, hasComments := s.comments[c.Post]; hasComments {
		if found, hasComment := comments[c.Id]; hasComment {

			if (found.Content != c.Content && c.Content == "") ||
				(found.Author != c.Author && c.Author == "") {
				return storage.ErrMissingCommentFields
			}

			c.UpdatedAt = timeToString(time.Now())
			comments[c.Id] = *c
			return nil
		}
	}
	return storage.ErrCommentNotFound
}
