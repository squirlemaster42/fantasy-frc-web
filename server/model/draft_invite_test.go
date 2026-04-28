package model

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCancelInvite_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare(`Update DraftInvites Set Canceled = true Where Id = \$1;`).
		ExpectExec().
		WithArgs(42).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = CancelInvite(db, 42)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCancelInvite_ReturnsErrorOnFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare(`Update DraftInvites Set Canceled = true Where Id = \$1;`).
		ExpectExec().
		WithArgs(99).
		WillReturnError(sql.ErrConnDone)

	err = CancelInvite(db, 99)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUninvitePlayer_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mock.ExpectPrepare(`Select OwnerUserUuid From Drafts Where Id = \$1;`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"OwnerUserUuid"}).AddRow(ownerUuid.String()))

	mock.ExpectPrepare(`Update DraftInvites Set Canceled = true Where Id = \$1 And DraftId = \$2;`).
		ExpectExec().
		WithArgs(10, 1).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = UninvitePlayer(db, 1, ownerUuid, 10)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUninvitePlayer_NotOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	requesterUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")

	mock.ExpectPrepare(`Select OwnerUserUuid From Drafts Where Id = \$1;`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"OwnerUserUuid"}).AddRow(ownerUuid.String()))

	err = UninvitePlayer(db, 1, requesterUuid, 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is not the owner")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUninvitePlayer_InviteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	ownerUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mock.ExpectPrepare(`Select OwnerUserUuid From Drafts Where Id = \$1;`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"OwnerUserUuid"}).AddRow(ownerUuid.String()))

	mock.ExpectPrepare(`Update DraftInvites Set Canceled = true Where Id = \$1 And DraftId = \$2;`).
		ExpectExec().
		WithArgs(99, 1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = UninvitePlayer(db, 1, ownerUuid, 99)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOutstandingInvitesForDraft(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "username", "InvitedUserUuid"}).
		AddRow(1, "player1", "550e8400-e29b-41d4-a716-446655440000").
		AddRow(2, "player2", "550e8400-e29b-41d4-a716-446655440001")

	mock.ExpectPrepare(`SELECT(.+)From DraftInvites di(.+)Where di.DraftId = \$1(.+)And di.Accepted = false(.+)And COALESCE\(di.Canceled, false\) = false;`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(rows)

	invites := GetOutstandingInvitesForDraft(db, 1)
	assert.Len(t, invites, 2)
	assert.Equal(t, "player1", invites[0].InvitedPlayerName)
	assert.Equal(t, "player2", invites[1].InvitedPlayerName)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOutstandingInvitesForDraft_ReturnsEmpty(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "username", "InvitedUserUuid"})

	mock.ExpectPrepare(`SELECT(.+)From DraftInvites di(.+)Where di.DraftId = \$1(.+)And di.Accepted = false(.+)And COALESCE\(di.Canceled, false\) = false;`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(rows)

	invites := GetOutstandingInvitesForDraft(db, 1)
	assert.Empty(t, invites)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInvite_ExcludesCanceled(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectPrepare(`SELECT(.+)From DraftInvites di(.+)Where di.Id = \$1(.+)And COALESCE\(di.Canceled, false\) = false;`).
		ExpectQuery().
		WithArgs(42).
		WillReturnError(sql.ErrNoRows)

	invite, err := GetInvite(db, 42)
	assert.Error(t, err)
	assert.Equal(t, sql.ErrNoRows, err)
	assert.Zero(t, invite.Id)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetInvites_ExcludesCanceled(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"Id", "username", "DisplayName"}).
		AddRow(1, "inviter", "Test Draft")

	mock.ExpectPrepare(`SELECT(.+)From DraftInvites di(.+)Where di.InvitedUserUuid = \$1(.+)And di.Accepted = false(.+)And COALESCE\(di.Canceled, false\) = false;`).
		ExpectQuery().
		WithArgs("550e8400-e29b-41d4-a716-446655440000").
		WillReturnRows(rows)

	invites := GetInvites(db, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"))
	assert.Len(t, invites, 1)
	assert.Equal(t, "inviter", invites[0].InvitingPlayerName)
	assert.NoError(t, mock.ExpectationsWereMet())
}
