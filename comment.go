package meshtalk

type Comment struct {
	Id        string `json:"id"`
	Post      string `json:"postId"`
	Content   string `json:"content"`
	Author    string `json:"author"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
	DeletedAt string `json:"deletedAt"`
}

func NewComment(id, post, content, author string) *Comment {
	return &Comment{
		Id:      id,
		Post:    post,
		Content: content,
		Author:  author,
	}
}
