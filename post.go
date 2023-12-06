package meshtalk

import "time"

type Post struct {
	Id        string
	Title     string
	Content   string
	Author    string
	CreatedAt *time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

func NewPost(id, title, content, author string) *Post {
	return &Post{
		Id:      id,
		Title:   title,
		Content: content,
		Author:  author,
	}
}
