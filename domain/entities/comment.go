package entities

// Represents a comment.
type Comment struct {
	Id        string    `json:"id"`
	Post      string    `json:"postId"`
	Content   string    `json:"content"`
	Author    *Customer `json:"author"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
	DeletedAt string    `json:"deletedAt"`
}

// Create a new comment representation.
func NewComment(post, content string, author *Customer) *Comment {
	return &Comment{
		Post:    post,
		Content: content,
		Author:  author,
	}
}
