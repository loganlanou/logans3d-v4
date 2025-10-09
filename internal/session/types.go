package session

// UserData represents the user information stored in the session
type UserData struct {
	ID        string
	Email     string
	FirstName string
	LastName  string
	FullName  string
	ImageURL  string
	Username  string
	HasImage  bool
}
