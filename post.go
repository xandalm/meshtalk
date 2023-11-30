package meshtalk

type Post struct {
	Id      string
	Title   string
	Content string
}

func NewPost(id, title, content string) *Post {
	return &Post{
		Id:      id,
		Title:   title,
		Content: content,
	}
}
