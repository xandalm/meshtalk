package entities

// Represents a post.
type Post struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

// Creates a new post representation.
func NewPost(title, content, author string) *Post {
	return &Post{
		Title:   title,
		Content: content,
		Author:  author,
	}
}

// It's like a helper to expose editable fields of the Post.
// The editable fields are pointers because this way is possible
// to keep them optional to be edited.
type PostInEditting struct {
	Id      string
	Title   *string
	Content *string
}
