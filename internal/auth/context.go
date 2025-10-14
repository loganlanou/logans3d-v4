package auth

// Context holds authentication data to be passed to templates
type Context struct {
	IsAuthenticated bool
	User            *UserData
}

// UserData contains user information for templates
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
