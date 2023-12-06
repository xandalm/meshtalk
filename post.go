package meshtalk

import "time"

type Post struct {
	Id        string     `json:"id"`
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	Author    string     `json:"author"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
}

func NewPost(id, title, content, author string) *Post {
	return &Post{
		Id:      id,
		Title:   title,
		Content: content,
		Author:  author,
	}
}
