package model

import (
	"crypto"
	"database/sql"
	"fmt"
	"server/assert"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
    UserUuid uuid.UUID
    Username string
    Password string
}

func (u *User) String() string {
    return fmt.Sprintf("User: {\n UserUuid: %s\n Username: %s\n}", u.UserUuid.String(), u.Username)
}

func RegisterUser(database *sql.DB, username string, password string) uuid.UUID {
    query := `INSERT INTO Users (UserUuid, username, password) Values ($1, $2, $3) Returning UserUuid;`
    assert := assert.CreateAssertWithContext("Register User")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    userUuid := uuid.New()
    assert.NoError(err, "Failed to create uuid")
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
    assert.NoError(err, "Failed to generate password hash")
    err = stmt.QueryRow(userUuid, username, string(hashedPassword)).Scan(&userUuid)
    assert.NoError(err, "Failed to register user")
    return userUuid
}

func UsernameTaken(database *sql.DB, username string) bool {
    query := `Select count(UserUuid) From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Username Taken")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var count int
    err = stmt.QueryRow(username).Scan(&count)
    assert.NoError(err, "Failed to check username taken")
    return count > 0
}

func GetUserUuidByUsername(database *sql.DB, username string) uuid.UUID {
    query := `Select UserUuid From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Get User Uuid By Username")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var userUuid uuid.UUID
    err = stmt.QueryRow(username).Scan(&userUuid)
    assert.NoError(err, "Failed to get user")
    return userUuid
}

func GetUsername(database *sql.DB, userUuid uuid.UUID) string {
    query := `Select Username From Users Where UserUuid = $1;`
    assert := assert.CreateAssertWithContext("Get Username")
    assert.AddContext("User Id", userUuid)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var username string
    err = stmt.QueryRow(userUuid).Scan(&username)
    assert.NoError(err, "Failed to get user")
    return username
}

// All crypto should happen before this since this just communicates with the DB
func ValidateLogin(database *sql.DB, username string, password string) bool {
    query := `Select password From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Validate Login")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var dbPassword string
    err = stmt.QueryRow(username).Scan(&dbPassword)
    assert.NoError(err, "Failed to validate login")
    err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
    return err == nil
}

// The old password logic should happen before this
// Should we move more logic here? No, we want to be able to
// send back error messages which we should need to check the database for
func UpdatePassword(database *sql.DB, username string, newPassword string) {
    query := `Update Users Set password = $1 Where username = $2;`
    assert := assert.CreateAssertWithContext("Update Password")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
    assert.NoError(err, "Failed to generate password hash")
    _, err = stmt.Exec(string(hashedPassword), username)
    assert.NoError(err, "Failed to Update Password")
}

// This can probably clean up session that expired more than a month ago or something
// Actually it can probably be sooner than that because expire tokens should never be reissued
func RegisterSession(database *sql.DB, userUuid uuid.UUID, sessionToken string) {
    query := `Insert Into UserSessions (userUuid, sessionToken, expirationTime) Values ($1, $2, now()::timestamp + '10 days');`
    assert := assert.CreateAssertWithContext("Register Session")
    assert.AddContext("User Uuid", userUuid)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    _, err = stmt.Exec(userUuid, hasher.Sum(nil))
    assert.NoError(err, "Failed to register session")
}

func UnRegisterSession(database *sql.DB, sessionToken string) {
    query := `Delete From UserSessions Where sessionToken = $1;`
    assert := assert.CreateAssertWithContext("Unregister Session")
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    _, err = stmt.Exec(hasher.Sum(nil))
    assert.NoError(err, "Failed to delete user session")
}

func GetUserBySessionToken(database *sql.DB, sessionToken string) uuid.UUID {
    query := `Select UserUuid From UserSessions Where sessionToken = $1 and now()::timestamp <= expirationTime;`
    assert := assert.CreateAssertWithContext("Get User By Session Token")
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    var userUuid uuid.UUID
    err = stmt.QueryRow(hasher.Sum(nil)).Scan(&userUuid)
    assert.NoError(err, "Failed to get user")
    UpdateSessionExpiration(database, userUuid, sessionToken)
    return userUuid
}

func UserIsAdmin(database *sql.DB, userUuid uuid.UUID) bool {
    query := `Select COALESCE(IsAdmin, false) From Users Where UserUuid = $1;`
    assert := assert.CreateAssertWithContext("User Is Admin")
    assert.AddContext("User Id", userUuid)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var isAdmin bool
    err = stmt.QueryRow(userUuid).Scan(&isAdmin)
    assert.NoError(err, "Failed to get user")
    return isAdmin
}

func UpdateSessionExpiration(database *sql.DB, userUuid uuid.UUID, sessionToken string) {
    //We want to make sure we only update the session token that the user logged in with
    query := `Update UserSessions Set expirationTime = now()::timestamp + '10 days' Where userUuid = $1 And sessionToken = $2;`
    assert := assert.CreateAssertWithContext("Update Session Expiration")
    assert.AddContext("User Uuid", userUuid)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    _, err = stmt.Exec(userUuid, hasher.Sum(nil))
    assert.NoError(err, "Failed to update session expiraton")
}

// Check if the session token is in the database and that it is not expired
func ValidateSessionToken(database *sql.DB, sessionToken string) bool {
    //I think <= is fine, it probably doesn't matter though
    query := `Select Count(*) From UserSessions Where sessionToken = $1 and now()::timestamp <= expirationTime;`
    assert := assert.CreateAssertWithContext("Validate Session Token")
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")
    hasher := crypto.SHA256.New()
    hasher.Write([]byte(sessionToken))
    var count int
    err = stmt.QueryRow(hasher.Sum(nil)).Scan(&count)
    assert.NoError(err, "Failed to validate session")
    //If the count is greater than one there is a problem
    //It probably means that we inserted the same token twice which shouldn't happen
    //Do we want to invalidate the session in that case
    return count == 1
}

func SearchUsers(database *sql.DB, searchString string, draftId int) ([]User, error) {
    query := `SELECT
                    Users.UserUuid,
                    Users.Username
                From Users
                Where Users.UserUuid Not In (
                    SELECT
                        u.UserUuid
                    FROM (
                        SELECT
                        USERS.UserUuid AS UserUuid,
                        USERS.USERNAME,
                        't' AS ACCEPTED,
                        DRAFTPLAYERS.PLAYERORDER,
                        DraftPlayers.Id As PlayerId
                        FROM USERS
                        INNER JOIN DRAFTPLAYERS ON DRAFTPLAYERS.UserUuid = USERS.UserUuid
                        WHERE DRAFTPLAYERS.DRAFTID = $1
                        UNION
                        SELECT
                        USERS.USERUUID AS USERID,
                        USERS.USERNAME,
                        DRAFTINVITES.ACCEPTED AS ACCEPTED,
                        -1 AS PLAYERORDER,
                        -1 As PlayerId
                        FROM USERS
                        INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
                        WHERE DRAFTINVITES.DRAFTID = $1
                    ) U
                )`

    if searchString != "" {
        query += " And Username Like CONCAT('%', CAST($2 As VARCHAR), '%');"
    } else {
        query += ";"
    }
    assert := assert.CreateAssertWithContext("Search Users")
    assert.AddContext("Search String", searchString)
    assert.AddContext("Query", query)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")

    var userRows *sql.Rows
    if searchString != "" {
        userRows, err = stmt.Query(draftId, searchString)
    } else {
        userRows, err = stmt.Query(draftId)
    }
    assert.NoError(err, "Failed to search users")

    users := make([]User, 0)

    for userRows.Next() {
        var userUuid uuid.UUID
        var username string

        err = userRows.Scan(&userUuid, &username)

        if err != nil {
            return nil, err
        }

        user := User{
            UserUuid: userUuid,
            Username: username,
        }

        users = append(users, user)
    }

    return users, nil
}
