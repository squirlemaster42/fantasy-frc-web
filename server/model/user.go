package model

import (
	"crypto"
	"database/sql"
	"server/assert"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
    Id int
    Username string
    Password string
}

func RegisterUser(database *sql.DB, username string, password string) {
    query := `INSERT INTO Users (username, password) Values ($1, $2);`
    assert := assert.CreateAssertWithContext("Register User")
    assert.AddContext("Username", username)
    assert.AddContext("Password", password)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    assert.NoError(err, "Failed to generate password hash")
    //TODO Do we want to change this to a byte array in the database
    _, err = stmt.Exec(username, string(hashedPassword))
    assert.NoError(err, "Failed to register user")
}

func UsernameTaken(database *sql.DB, username string) bool {
    query := `Select count(Id) From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Username Taken")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var count int
    err = stmt.QueryRow(username).Scan(&count)
    assert.NoError(err, "Failed to check username taken")
    return count > 0
}

func GetUserIdByUsername(database *sql.DB, username string) int {
    query := `Select Id From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Get User Id By Username")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var id int
    err = stmt.QueryRow(username).Scan(&id)
    assert.NoError(err, "Failed to get user")
    return id
}

func GetUsername(database *sql.DB, userId int) string {
    query := `Select Username From Users Where Id = $1;`
    assert := assert.CreateAssertWithContext("Get Username")
    assert.AddContext("User Id", userId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var username string
    err = stmt.QueryRow(userId).Scan(&username)
    assert.NoError(err, "Failed to get user")
    return username
}

//All crypto should happen before this since this just communicates with the DB
func ValidateLogin(database *sql.DB, username string, password string) bool {
    query := `Select password From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Validate Login")
    assert.AddContext("Username", username)
    assert.AddContext("Password", password)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var dbPassword string
    err = stmt.QueryRow(username).Scan(&dbPassword)
    assert.NoError(err, "Failed to validate login")
    err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
    return err == nil
}

//The old password logic should happen before this
//Should we move more logic here? No, we want to be able to
//send back error messages which we should need to check the database for
func UpdatePassword(database *sql.DB, username string, newPassword string) {
    query := `Update Users Set password = $1 Where username = $2;`
    assert := assert.CreateAssertWithContext("Update Password")
    assert.AddContext("Username", username)
    assert.AddContext("New Password", newPassword)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
    assert.NoError(err, "Failed to generate password hash")
    //TODO Do we want to change this to a byte array in the database
    _, err = stmt.Exec(string(hashedPassword), username)
    assert.NoError(err, "Failed to Update Password")
}

//This can probably clean up session that expired more than a month ago or something
//Actually it can probably be sooner than that because expire tokens should never be reissued
func RegisterSession(database *sql.DB, userId int, sessionToken string) {
    query := `Insert Into UserSessions (userId, sessionToken, expirationTime) Values ($1, $2, now()::timestamp + '10 days');`
    assert := assert.CreateAssertWithContext("Register Session")
    assert.AddContext("UserId", userId)
    //I dont think I'm worried about the session tokens being here because if this fails we have bigger issues
    assert.AddContext("SessionToken", sessionToken)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    _, err = stmt.Exec(userId, hasher.Sum(nil))
    assert.NoError(err, "Fauled to register session")
}

//TODO refactor this to return an error if the user does not exists
func GetUserBySessionToken(database *sql.DB, sessionToken string) int {
    query := `Select UserId From UserSessions Where sessionToken = $1 and now()::timestamp <= expirationTime;`
    assert := assert.CreateAssertWithContext("Get User By Session Token")
    assert.AddContext("Session Token", sessionToken)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    assert.AddContext("Hashed Session Token", hasher.Sum(nil))
    var userId int
    err = stmt.QueryRow(hasher.Sum(nil)).Scan(&userId)
    assert.NoError(err, "Failed to get user")
    UpdateSessionExpiration(database, userId, sessionToken)
    return userId
}

func UpdateSessionExpiration(database *sql.DB, userId int, sessionToken string) {
    //We want to make sure we only update the session token that the user logged in with
    query := `Update UserSessions Set expirationTime = now()::timestamp + '10 days' Where userId = $1 And sessionToken = $2;`
    assert := assert.CreateAssertWithContext("Update Session Expiration")
    assert.AddContext("User Id", userId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    _, err = stmt.Exec(userId, hasher.Sum(nil))
    assert.NoError(err, "Failed to update session expiraton")
}

//Check if the session token is in the database and that it is not expired
func ValidateSessionToken(database *sql.DB, userId int, sessionToken string) bool {
    //I think <= is fine, it probably doesn't matter though
    query := `Select Count(*) From UserSessions Where userId = $1 and sessionToken = $2 and now()::timestamp <= expirationTime;`
    assert := assert.CreateAssertWithContext("Validate Session Token")
    assert.AddContext("User Id", userId)
    //This one is a little more concerning, but its probably fine
    assert.AddContext("Session Token", sessionToken)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    var count int
    err = stmt.QueryRow(userId, hasher.Sum(nil)).Scan(&count)
    assert.NoError(err, "Failed to validate session")
    //If the count is greater than one there is a problem
    //It probably means that we inserted the same token twice which shouldn't happen
    //Do we want to invalidate the session in that case
    return count == 1
}
