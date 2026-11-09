package model

import (
	"context"
	"crypto"
	"database/sql"
	"fmt"
	"server/database"
	"server/log"

	"uuid"
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

func registerUser(ctx context.Context, db database.DBTX, username string, passwordHash string) (uuid.UUID, error) {
	query := `INSERT INTO Users (UserUuid, username, password) Values ($1, $2, $3) Returning UserUuid;`
	userUuid := uuid.New()
	return database.QueryRow(ctx, db, query, func(row *sql.Row) (uuid.UUID, error) {
		if err := row.Scan(&userUuid); err != nil {
			return uuid.UUID{}, fmt.Errorf("failed to register user: %w", err)
		}
		return userUuid, nil
	}, userUuid, username, passwordHash)
}

func usernameTaken(ctx context.Context, db database.DBTX, username string) (bool, error) {
	query := `Select count(UserUuid) From Users Where username = $1;`
	count, err := database.QueryRowValue[int](ctx, db, query, username)
	if err != nil {
		return false, fmt.Errorf("failed to check username availability: %w", err)
	}
	return count > 0, nil
}

func getUserUuidByUsername(ctx context.Context, db database.DBTX, username string) (uuid.UUID, error) {
	query := `Select UserUuid From Users Where username = $1;`
	userUuid, err := database.QueryRowValue[uuid.UUID](ctx, db, query, username)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get user: %w", err)
	}
	return userUuid, nil
}

func getUsername(ctx context.Context, db database.DBTX, userUuid uuid.UUID) (string, error) {
	query := `Select Username From Users Where UserUuid = $1;`
	username, err := database.QueryRowValue[string](ctx, db, query, userUuid)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return username, nil
}

func getDiscordId(ctx context.Context, db database.DBTX, userUuid uuid.UUID) (string, error) {
	query := `Select Coalesce(discordId, '') From Users Where UserUuid = $1;`
	discordId, err := database.QueryRowValue[string](ctx, db, query, userUuid)
	if err != nil {
		return "", fmt.Errorf("failed to get discord id: %w", err)
	}
	return discordId, nil
}

func updateDiscordId(ctx context.Context, db database.DBTX, userUuid uuid.UUID, discordId string) error {
	query := `Update Users Set discordId = $1 Where UserUuid = $2;`
	if _, err := database.Exec(ctx, db, query, discordId, userUuid); err != nil {
		return fmt.Errorf("failed to update discord id: %w", err)
	}
	return nil
}

func getPasswordHashByUsername(ctx context.Context, db database.DBTX, username string) (string, error) {
	query := `Select password From Users Where username = $1;`
	passwordHash, err := database.QueryRowValue[string](ctx, db, query, username)
	if err != nil {
		return "", fmt.Errorf("failed to scan getPasswordHashByUsername row: %w", err)
	}
	return passwordHash, nil
}

// updatePassword writes the new password hash to the database.
// Password validation and old-password checking are intentionally done
// before calling this function so callers can return user-facing errors.
func updatePassword(ctx context.Context, db database.DBTX, username string, passwordHash string) error {
	query := `Update Users Set password = $1 Where username = $2;`
	if _, err := database.Exec(ctx, db, query, passwordHash, username); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

func registerSession(ctx context.Context, db database.DBTX, userUuid uuid.UUID, sessionToken string) error {
	query := `Insert Into UserSessions (userUuid, sessionToken, expirationTime) Values ($1, $2, now()::timestamptz + make_interval(days => $3));`
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	if _, err := database.Exec(ctx, db, query, userUuid, hasher.Sum(nil), SessionExpirationDays()); err != nil {
		return fmt.Errorf("failed to register session: %w", err)
	}
	return nil
}

func unregisterSession(ctx context.Context, db database.DBTX, sessionToken string) error {
	query := `Delete From UserSessions Where sessionToken = $1;`
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	if _, err := database.Exec(ctx, db, query, hasher.Sum(nil)); err != nil {
		return fmt.Errorf("failed to delete user session: %w", err)
	}
	return nil
}

func getUserBySessionToken(ctx context.Context, db database.DBTX, sessionToken string) (uuid.UUID, error) {
	query := `Select UserUuid From UserSessions Where sessionToken = $1 and now()::timestamptz <= expirationTime;`
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	userUuid, err := database.QueryRowValue[uuid.UUID](ctx, db, query, hasher.Sum(nil))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to get user: %w", err)
	}
	if err := updateSessionExpiration(ctx, db, userUuid, sessionToken); err != nil {
		log.Error(ctx, "Failed to update session expiration", "error", err)
	}
	return userUuid, nil
}

func userIsAdmin(ctx context.Context, db database.DBTX, userUuid uuid.UUID) (bool, error) {
	query := `Select COALESCE(IsAdmin, false) From Users Where UserUuid = $1;`
	isAdmin, err := database.QueryRowValue[bool](ctx, db, query, userUuid)
	if err != nil {
		return false, fmt.Errorf("failed to get user: %w", err)
	}
	return isAdmin, nil
}

func updateSessionExpiration(ctx context.Context, db database.DBTX, userUuid uuid.UUID, sessionToken string) error {
	// We want to make sure we only update the session token that the user logged in with
	query := `Update UserSessions Set expirationTime = now()::timestamptz + make_interval(days => $3) Where userUuid = $1 And sessionToken = $2;`
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	if _, err := database.Exec(ctx, db, query, userUuid, hasher.Sum(nil), SessionExpirationDays()); err != nil {
		return fmt.Errorf("failed to update session expiration: %w", err)
	}
	return nil
}

// Check if the session token is in the database and that it is not expired
func validateSessionToken(ctx context.Context, db database.DBTX, sessionToken string) (bool, error) {
	// I think <= is fine, it probably doesn't matter though
	query := `Select Count(*) From UserSessions Where sessionToken = $1 and now()::timestamptz <= expirationTime;`
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(sessionToken))
	count, err := database.QueryRowValue[int](ctx, db, query, hasher.Sum(nil))
	if err != nil {
		return false, fmt.Errorf("failed to validate session: %w", err)
	}
	if count > 1 {
		log.Error(ctx, "Duplicate session token detected, cleaning up", "count", count)
		deleteQuery := `Delete From UserSessions Where sessionToken = $1;`
		if _, err := database.Exec(ctx, db, deleteQuery, hasher.Sum(nil)); err != nil {
			return false, err
		}
		return false, nil
	}
	return count == 1, nil
}

// InvalidateAllUserSessionsExcept deletes all sessions for a user except the given token.
func invalidateAllUserSessionsExcept(ctx context.Context, db database.DBTX, userUuid uuid.UUID, keepSessionToken string) error {
	query := `Delete From UserSessions Where userUuid = $1 And sessionToken != $2;`
	hasher := crypto.SHA256.New()
	hasher.Write([]byte(keepSessionToken))
	if _, err := database.Exec(ctx, db, query, userUuid, hasher.Sum(nil)); err != nil {
		return fmt.Errorf("failed to invalidate sessions: %w", err)
	}
	return nil
}

func searchUsers(ctx context.Context, db database.DBTX, searchString string, draftId int) ([]User, error) {
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

	args := []any{draftId}
	if searchString != "" {
		query += " And Username ILike CONCAT('%', CAST($2 As VARCHAR), '%');"
		args = append(args, searchString)
	} else {
		query += ";"
	}

	return database.QueryRows(ctx, db, query, func(rows *sql.Rows) (User, error) {
		var user User
		if err := rows.Scan(&user.UserUuid, &user.Username); err != nil {
			return user, fmt.Errorf("failed to scan searchUsers rows: %w", err)
		}
		return user, nil
	}, args...)
}
