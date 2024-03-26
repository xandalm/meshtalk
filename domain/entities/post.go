package entities

type Post struct {
	Id        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

func NewPost(id, title, content, author string) *Post {
	return &Post{
		Id:      id,
		Title:   title,
		Content: content,
		Author:  author,
	}
}
