package model

//TODO The primary key here is not good. I should fix that.
type User struct {
    Id int
    Username string
    Password string
}

func RegisterUser(user User) error {
    return nil
}

func UsernameTaken(username string) bool {
    return false
}

func ValidateLogin(username string, password string) bool {
    return false
}

func UpdatePassword(username, oldPassword string, newPassword string) bool {
    return false
}
