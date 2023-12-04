package meshtalk

type Post struct {
	Id      string
	Title   string
	Content string
	Author  string
}

func NewPost(id, title, content, author string) *Post {
	return &Post{
		Id:      id,
		Title:   title,
		Content: content,
		Author:  author,
	}
}
