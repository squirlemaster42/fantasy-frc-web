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

//Validate login
//Update password
