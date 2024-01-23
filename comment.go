package meshtalk

import "time"

type Comment struct {
	Id        string     `json:"id"`
	Post      string     `json:"postId"`
	Content   string     `json:"content"`
	Author    string     `json:"author"`
	CreatedAt *time.Time `json:"createdAt"`
	UpdatedAt *time.Time `json:"updatedAt"`
	DeletedAt *time.Time `json:"deletedAt"`
}

func NewComment(id, post, content, author string) *Comment {
	return &Comment{
		Id:      id,
		Post:    post,
		Content: content,
		Author:  author,
	}
}
