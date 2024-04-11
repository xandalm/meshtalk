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

// It's like a helper to expose editable fields of the Comment.
// The editable fields are pointers because this way is possible
// to keep them optional to be edited.
type CommentInEditting struct {
	Post    string
	Id      string
	Content *string
}
