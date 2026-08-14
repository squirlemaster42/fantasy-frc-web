package model

import (
	"context"
	"crypto"
	"database/sql"
	"fmt"
	"server/database"
	"server/log"

	"github.com/google/uuid"
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

func registerUser(ctx context.Context, db *sql.DB, username string, passwordHash string) (uuid.UUID, error) {
	query := `INSERT INTO Users (UserUuid, username, password) Values ($1, $2, $3) Returning UserUuid;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer database.CloseStatement(ctx, stmt, "RegisterUser")
	userUuid := uuid.New()
	err = stmt.QueryRowContext(ctx, userUuid, username, passwordHash).Scan(&userUuid)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to register user: %w", err)
	}
	return userUuid, nil
}

func usernameTaken(ctx context.Context, db *sql.DB, username string) (bool, error) {
	query := `Select count(UserUuid) From Users Where username = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return false, err
	}
	defer database.CloseStatement(ctx, stmt, "UsernameTaken")
	var count int
	err = stmt.QueryRowContext(ctx, username).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getUserUuidByUsername(ctx context.Context, db *sql.DB, username string) (uuid.UUID, error) {
	query := `Select UserUuid From Users Where username = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer database.CloseStatement(ctx, stmt, "GetUserUuidByUsername")
	var userUuid uuid.UUID
	err = stmt.QueryRowContext(ctx, username).Scan(&userUuid)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get user: %w", err)
	}
	return userUuid, nil
}

func getUsername(ctx context.Context, db *sql.DB, userUuid uuid.UUID) (string, error) {
	query := `Select Username From Users Where UserUuid = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return "", err
	}
	defer database.CloseStatement(ctx, stmt, "GetUsername")
	var username string
	err = stmt.QueryRowContext(ctx, userUuid).Scan(&username)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return username, nil
}

func getDiscordId(ctx context.Context, db *sql.DB, userUuid uuid.UUID) (string, error) {
	query := `Select Coalesce(discordId, '') From Users Where UserUuid = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return "", err
	}
	defer database.CloseStatement(ctx, stmt, "GetDiscordId")
	var discordId string
	err = stmt.QueryRowContext(ctx, userUuid).Scan(&discordId)
	if err != nil {
		return "", fmt.Errorf("failed to get discord id: %w", err)
	}
	return discordId, nil
}

func updateDiscordId(ctx context.Context, db *sql.DB, userUuid uuid.UUID, discordId string) error {
	query := `Update Users Set discordId = $1 Where UserUuid = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "UpdateDiscordId")
	_, err = stmt.ExecContext(ctx, discordId, userUuid)
	if err != nil {
		return fmt.Errorf("failed to update discord id: %w", err)
	}
	return nil
}

func getPasswordHashByUsername(ctx context.Context, db *sql.DB, username string) (string, error) {
	query := `Select password From Users Where username = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return "", err
	}
	defer database.CloseStatement(ctx, stmt, "GetPasswordHashByUsername")
	var passwordHash string
	err = stmt.QueryRowContext(ctx, username).Scan(&passwordHash)
	if err != nil {
		return "", err
	}
	return passwordHash, nil
}

// The old password logic should happen before this
// Should we move more logic here? No, we want to be able to
// send back error messages which we should need to check the database for
func updatePassword(ctx context.Context, db *sql.DB, username string, passwordHash string) error {
	query := `Update Users Set password = $1 Where username = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "UpdatePassword")
	_, err = stmt.ExecContext(ctx, passwordHash, username)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

func registerSession(ctx context.Context, db *sql.DB, userUuid uuid.UUID, sessionToken string) error {
	query := `Insert Into UserSessions (userUuid, sessionToken, expirationTime) Values ($1, $2, now()::timestamptz + '10 days');`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "RegisterSession")
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	_, err = stmt.ExecContext(ctx, userUuid, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}
	return nil
}

func unregisterSession(ctx context.Context, db *sql.DB, sessionToken string) error {
	query := `Delete From UserSessions Where sessionToken = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "UnRegisterSession")
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	_, err = stmt.ExecContext(ctx, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}
	return nil
}

func getUserBySessionToken(ctx context.Context, db *sql.DB, sessionToken string) (uuid.UUID, error) {
	query := `Select UserUuid From UserSessions Where sessionToken = $1 and now()::timestamptz <= expirationTime;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return uuid.UUID{}, err
	}
	defer database.CloseStatement(ctx, stmt, "GetUserBySessionToken")
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	var userUuid uuid.UUID
	err = stmt.QueryRowContext(ctx, hasher.Sum(nil)).Scan(&userUuid)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get user: %w", err)
	}
	if err := updateSessionExpiration(ctx, db, userUuid, sessionToken); err != nil {
		log.Error(ctx, "Failed to update session expiration", "error", err)
	}
	return userUuid, nil
}

func userIsAdmin(ctx context.Context, db *sql.DB, userUuid uuid.UUID) (bool, error) {
	query := `Select COALESCE(IsAdmin, false) From Users Where UserUuid = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return false, err
	}
	defer database.CloseStatement(ctx, stmt, "UserIsAdmin")
	var isAdmin bool
	err = stmt.QueryRowContext(ctx, userUuid).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}
	return isAdmin, nil
}

func updateSessionExpiration(ctx context.Context, db *sql.DB, userUuid uuid.UUID, sessionToken string) error {
	//We want to make sure we only update the session token that the user logged in with
	query := `Update UserSessions Set expirationTime = now()::timestamptz + '10 days' Where userUuid = $1 And sessionToken = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "UpdateSessionExpiration")
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	_, err = stmt.ExecContext(ctx, userUuid, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to update session expiration: %w", err)
	}
	return nil
}

// Check if the session token is in the database and that it is not expired
func validateSessionToken(ctx context.Context, db *sql.DB, sessionToken string) (bool, error) {
	//I think <= is fine, it probably doesn't matter though
	query := `Select Count(*) From UserSessions Where sessionToken = $1 and now()::timestamptz <= expirationTime;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return false, err
	}
	defer database.CloseStatement(ctx, stmt, "ValidateSessionToken")
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	var count int
	err = stmt.QueryRowContext(ctx, hasher.Sum(nil)).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to validate session: %w", err)
	}
	if count > 1 {
		log.Error(ctx, "Duplicate session token detected, cleaning up", "count", count)
		deleteQuery := `Delete From UserSessions Where sessionToken = $1;`
		delStmt, err := database.Prepare(ctx, db, deleteQuery)
		if err != nil {
			return false, err
		}
		defer database.CloseStatement(ctx, delStmt, "CleanupDuplicateSessions")
		_, _ = delStmt.ExecContext(ctx, hasher.Sum(nil))
		return false, nil
	}
	return count == 1, nil
}

// InvalidateAllUserSessionsExcept deletes all sessions for a user except the given token.
func invalidateAllUserSessionsExcept(ctx context.Context, db *sql.DB, userUuid uuid.UUID, keepSessionToken string) error {
	query := `Delete From UserSessions Where userUuid = $1 And sessionToken != $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "InvalidateAllUserSessionsExcept")
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(keepSessionToken))
	_, err = stmt.ExecContext(ctx, userUuid, hasher.Sum(nil))
	if err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}
	return nil
}

func searchUsers(ctx context.Context, db *sql.DB, searchString string, draftId int) ([]User, error) {
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
                        'accepted' AS Status,
                        DRAFTPLAYERS.PLAYERORDER,
                        DraftPlayers.Id As PlayerId
                        FROM USERS
                        INNER JOIN DRAFTPLAYERS ON DRAFTPLAYERS.UserUuid = USERS.UserUuid
                        WHERE DRAFTPLAYERS.DRAFTID = $1
                        UNION
                        SELECT
                        USERS.USERUUID AS USERID,
                        USERS.USERNAME,
                        DRAFTINVITES.STATUS AS STATUS,
                        -1 AS PLAYERORDER,
                        -1 As PlayerId
                        FROM USERS
                        INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
                        WHERE DRAFTINVITES.DRAFTID = $1
                        	AND DRAFTINVITES.Status != 'canceled'
                    ) U
                )`

	if searchString != "" {
		query += " And Username ILike CONCAT('%', CAST($2 As VARCHAR), '%');"
	} else {
		query += ";"
	}
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, err
	}
	defer database.CloseStatement(ctx, stmt, "SearchUsers")

	var userRows *sql.Rows
	if searchString != "" {
		userRows, err = stmt.QueryContext(ctx, draftId, searchString)
	} else {
		userRows, err = stmt.QueryContext(ctx, draftId)
	}
	if err != nil {
		return nil, err
	}
	defer database.CloseRows(ctx, userRows, "SearchUsers")

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
