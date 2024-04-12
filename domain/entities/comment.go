package entities

// Represents a comment.
type Comment struct {
	Id        string `json:"id"`
	Post      string `json:"postId"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

// Create a new comment representation.
func NewComment(post, content, author string) *Comment {
	return &Comment{
		Post:    post,
		Content: content,
		Author:  author,
	}
}

// It's like a helper to expose editable fields of the Comment.
// The editable fields are pointers because this way is possible
// to keep them optional to be edited.
type CommentInEditting struct {
	Post    string
	Id      string
	Content *string
}
