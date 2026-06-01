package model

import (
	"context"
	"crypto"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/log"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	UserUuid  uuid.UUID
	Username  string
	Password  string
	DiscordId string
}

func (u *User) String() string {
	return fmt.Sprintf("User: {\n UserUuid: %s\n Username: %s\n}", u.UserUuid.String(), u.Username)
}

func registerUser(ctx context.Context, database *sql.DB, username string, password string) (uuid.UUID, error) {
	query := `INSERT INTO Users (UserUuid, username, password) Values ($1, $2, $3) Returning UserUuid;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "RegisterUser: Failed to close statement", "error", err)
		}
	}()
	userUuid := uuid.New()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to generate password hash: %w", err)
	}
	err = stmt.QueryRowContext(ctx, userUuid, username, string(hashedPassword)).Scan(&userUuid)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to register user: %w", err)
	}
	return userUuid, nil
}

func usernameTaken(ctx context.Context, database *sql.DB, username string) (bool, error) {
	query := `Select count(UserUuid) From Users Where username = $1;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UsernameTaken: Failed to close statement", "error", err)
		}
	}()
	var count int
	err = stmt.QueryRowContext(ctx, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getUserUuidByUsername(ctx context.Context, database *sql.DB, username string) (uuid.UUID, error) {
	query := `Select UserUuid From Users Where username = $1;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetUserUuidByUsername: Failed to close statement", "error", err)
		}
	}()
	var userUuid uuid.UUID
	err = stmt.QueryRowContext(ctx, username).Scan(&userUuid)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get user: %w", err)
	}
	return userUuid, nil
}

func getUsername(ctx context.Context, database *sql.DB, userUuid uuid.UUID) (string, error) {
	query := `Select Username From Users Where UserUuid = $1;`
	assert := assert.CreateAssertWithContext("Get Username")
	assert.AddContext("User Id", userUuid)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetUsername: Failed to close statement", "error", err)
		}
	}()
	var username string
	err = stmt.QueryRowContext(ctx, userUuid).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return username, nil
}

func getDiscordId(ctx context.Context, database *sql.DB, userUuid uuid.UUID) (string, error) {
	query := `Select Coalesce(discordId, '') From Users Where UserUuid = $1;`
	assert := assert.CreateAssertWithContext("Get Discord Id")
	assert.AddContext("User Id", userUuid)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetDiscordId: Failed to close statement", "error", err)
		}
	}()
	var discordId string
	err = stmt.QueryRowContext(ctx, userUuid).Scan(&discordId)
	if err != nil {
		return "", fmt.Errorf("failed to get discord id: %w", err)
	}
	return discordId, nil
}

func updateDiscordId(ctx context.Context, database *sql.DB, userUuid uuid.UUID, discordId string) error {
	query := `Update Users Set discordId = $1 Where UserUuid = $2;`
	assert := assert.CreateAssertWithContext("Update Discord Id")
	assert.AddContext("User Id", userUuid)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UpdateDiscordId: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, discordId, userUuid)
	if err != nil {
		return fmt.Errorf("failed to update discord id: %w", err)
	}
	return nil
}

// Precomputed dummy bcrypt hash for constant-time comparison on unknown usernames
var dummyPasswordHash = []byte("$2a$14$abcdefghijklmnopqrstuuxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

// ValidateLogin validates credentials in constant time regardless of username existence.
func validateLogin(ctx context.Context, database *sql.DB, username string, password string) (bool, error) {
	query := `Select password From Users Where username = $1;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "ValidateLogin: Failed to close statement", "error", err)
		}
	}()
	var dbPassword string
	err = stmt.QueryRowContext(ctx, username).Scan(&dbPassword)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Constant-time dummy comparison to prevent username enumeration
			_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
			return false, nil
		}
		return false, fmt.Errorf("failed to validate login: %w", err)
	}
	err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	return err == nil, nil
}

// The old password logic should happen before this
// Should we move more logic here? No, we want to be able to
// send back error messages which we should need to check the database for
func updatePassword(ctx context.Context, database *sql.DB, username string, newPassword string) error {
	query := `Update Users Set password = $1 Where username = $2;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UpdatePassword: Failed to close statement", "error", err)
		}
	}()
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 14)
	if err != nil {
		return fmt.Errorf("failed to generate password hash: %w", err)
	}
	_, err = stmt.ExecContext(ctx, string(hashedPassword), username)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

// This can probably clean up session that expired more than a month ago or something
// Actually it can probably be sooner than that because expire tokens should never be reissued
func registerSession(ctx context.Context, database *sql.DB, userUuid uuid.UUID, sessionToken string) error {
	query := `Insert Into UserSessions (userUuid, sessionToken, expirationTime) Values ($1, $2, now()::timestamp + '10 days');`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "RegisterSession: Failed to close statement", "error", err)
		}
	}()
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	_, err = stmt.ExecContext(ctx, userUuid, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}
	return nil
}

func unregisterSession(ctx context.Context, database *sql.DB, sessionToken string) error {
	query := `Delete From UserSessions Where sessionToken = $1;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UnRegisterSession: Failed to close statement", "error", err)
		}
	}()
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	_, err = stmt.ExecContext(ctx, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}
	return nil
}

func getUserBySessionToken(ctx context.Context, database *sql.DB, sessionToken string) (uuid.UUID, error) {
	query := `Select UserUuid From UserSessions Where sessionToken = $1 and now()::timestamp <= expirationTime;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetUserBySessionToken: Failed to close statement", "error", err)
		}
	}()
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	var userUuid uuid.UUID
	err = stmt.QueryRowContext(ctx, hasher.Sum(nil)).Scan(&userUuid)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get user: %w", err)
	}
	if err := updateSessionExpiration(ctx, database, userUuid, sessionToken); err != nil {
		log.Warn(ctx, "Failed to update session expiration", "error", err)
	}
	return userUuid, nil
}

func userIsAdmin(ctx context.Context, database *sql.DB, userUuid uuid.UUID) (bool, error) {
	query := `Select COALESCE(IsAdmin, false) From Users Where UserUuid = $1;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UserIsAdmin: Failed to close statement", "error", err)
		}
	}()
	var isAdmin bool
	err = stmt.QueryRowContext(ctx, userUuid).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}
	return isAdmin, nil
}

func updateSessionExpiration(ctx context.Context, database *sql.DB, userUuid uuid.UUID, sessionToken string) error {
	//We want to make sure we only update the session token that the user logged in with
	query := `Update UserSessions Set expirationTime = now()::timestamp + '10 days' Where userUuid = $1 And sessionToken = $2;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "UpdateSessionExpiration: Failed to close statement", "error", err)
		}
	}()
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	_, err = stmt.ExecContext(ctx, userUuid, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to update session expiration: %w", err)
	}
	return nil
}

// Check if the session token is in the database and that it is not expired
func validateSessionToken(ctx context.Context, database *sql.DB, sessionToken string) (bool, error) {
	//I think <= is fine, it probably doesn't matter though
	query := `Select Count(*) From UserSessions Where sessionToken = $1 and now()::timestamp <= expirationTime;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "ValidateSessionToken: Failed to close statement", "error", err)
		}
	}()
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	var count int
	err = stmt.QueryRowContext(ctx, hasher.Sum(nil)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to validate session: %w", err)
	}
	//If the count is greater than one there is a problem
	//It probably means that we inserted the same token twice which shouldn't happen
	//Do we want to invalidate the session in that case
	return count == 1, nil
}

// InvalidateAllUserSessionsExcept deletes all sessions for a user except the given token.
func invalidateAllUserSessionsExcept(ctx context.Context, database *sql.DB, userUuid uuid.UUID, keepSessionToken string) error {
	query := `Delete From UserSessions Where userUuid = $1 And sessionToken != $2;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoErrorCF(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "InvalidateAllUserSessionsExcept: Failed to close statement", "error", err)
		}
	}()
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(keepSessionToken))
	_, err = stmt.ExecContext(ctx, userUuid, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}
	return nil
}

func searchUsers(ctx context.Context, database *sql.DB, searchString string, draftId int) ([]User, error) {
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
                        	AND (
                        	    DRAFTINVITES.CANCELED = 'f'
                        	    OR DRAFTINVITES.CANCELED IS NULL
                        	)
                    ) U
                )`

	if searchString != "" {
		query += " And Username ILike CONCAT('%', CAST($2 As VARCHAR), '%');"
	} else {
		query += ";"
	}
	assert := assert.CreateAssertWithContext("Search Users")
	assert.AddContext("Search String", searchString)
	assert.AddContext("Query", query)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "SearchUsers: Failed to close statement", "error", err)
		}
	}()

	var userRows *sql.Rows
	if searchString != "" {
		userRows, err = stmt.QueryContext(ctx, draftId, searchString)
	} else {
		userRows, err = stmt.QueryContext(ctx, draftId)
	}
	assert.NoError(ctx, err, "Failed to search users")
	defer func() {
		if err := userRows.Close(); err != nil {
			log.Warn(ctx, "SearchUsers: Failed to close rows", "error", err)
		}
	}()

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
