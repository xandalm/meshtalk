package entities

// Represents a post.
type Post struct {
	Id        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Author    *Customer `json:"author"`
	CreatedAt string    `json:"createdAt"`
	UpdatedAt string    `json:"updatedAt"`
	DeletedAt string    `json:"deletedAt"`
}

// Creates a new post representation.
func NewPost(title, content string, author *Customer) *Post {
	return &Post{
		Title:   title,
		Content: content,
		Author:  author,
	}
}
