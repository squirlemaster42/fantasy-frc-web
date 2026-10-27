package model

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

var premoveColumns = []string{
	"Id",
	"TeamTBAId",
	"Status",
	"Error",
	"PickOrder",
	"DraftPlayerId",
}

func TestHasPremoveWhenTrue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`SELECT EXISTS`).
		ExpectQuery().
		WithArgs(1, premoveStatusPending).
		WillReturnRows(
			sqlmock.NewRows([]string{"exists"}).
				AddRow(true),
		)

	has, err := hasPremove(context.Background(), db, 1)

	require.NoError(t, err)
	require.True(t, has)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestHasPremoveWhenFalse(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`SELECT EXISTS`).
		ExpectQuery().
		WithArgs(1, premoveStatusPending).
		WillReturnRows(
			sqlmock.NewRows([]string{"exists"}).
				AddRow(false),
		)

	has, err := hasPremove(context.Background(), db, 1)

	require.NoError(t, err)
	require.False(t, has)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPremoveReturnsOldestPendingPremove(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1, premoveStatusPending).
		WillReturnRows(
			sqlmock.NewRows(premoveColumns).
				AddRow(10, "frc254", premoveStatusPending, "", 1, 1),
		)

	premove, err := getPremove(context.Background(), db, 1)

	require.NoError(t, err)
	require.Equal(t, Premove{
		Id:            10,
		TeamTbaId:     "frc254",
		Status:        premoveStatusPending,
		Error:         "",
		PickOrder:     1,
		DraftPlayerId: 1,
	}, premove)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPremoveReturnsErrNoRows(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1, premoveStatusPending).
		WillReturnRows(sqlmock.NewRows(premoveColumns))

	_, err = getPremove(context.Background(), db, 1)

	require.ErrorIs(t, err, sql.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestGetPremovesReturnsAllPremovesForPlayer(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows(premoveColumns).
				AddRow(10, "frc254", premoveStatusPending, "", 1, 1).
				AddRow(11, "frc1678", premoveStatusMade, "", 2, 1),
		)

	premoves, err := getPremoves(context.Background(), db, 1)

	require.NoError(t, err)
	require.Equal(t, []Premove{
		{
			Id:            10,
			TeamTbaId:     "frc254",
			Status:        premoveStatusPending,
			Error:         "",
			PickOrder:     1,
			DraftPlayerId: 1,
		},
		{
			Id:            11,
			TeamTbaId:     "frc1678",
			Status:        premoveStatusMade,
			Error:         "",
			PickOrder:     2,
			DraftPlayerId: 1,
		},
	}, premoves)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesReplacesQueueInOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(premoveColumns))

	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveParkOrderBase).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveMadeOrderBase, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPrepare(`INSERT INTO PickMoves`).
		ExpectExec().
		WithArgs(1, "frc254", premoveStatusPending, 1).
		WillReturnResult(sqlmock.NewResult(20, 1))
	mock.ExpectExec(`INSERT INTO PickMoves`).
		WithArgs(1, "frc1678", premoveStatusPending, 2).
		WillReturnResult(sqlmock.NewResult(21, 1))

	_, err = replacePremoves(context.Background(), db, 1, []string{"frc254", "frc1678"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesUpdatesExistingOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows(premoveColumns).
				AddRow(10, "frc254", premoveStatusPending, "", 2, 1).
				AddRow(11, "frc1678", premoveStatusPending, "", 1, 1),
		)

	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveParkOrderBase).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveMadeOrderBase, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPrepare(`UPDATE PickMoves SET PickOrder = \$1`).
		ExpectExec().
		WithArgs(1, 10, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`UPDATE PickMoves SET PickOrder = \$1`).
		WithArgs(2, 11, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = replacePremoves(context.Background(), db, 1, []string{"frc254", "frc1678"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesKeepsUnchangedAndRemovesDroppedTeams(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows(premoveColumns).
				AddRow(10, "frc254", premoveStatusPending, "", 1, 1).
				AddRow(11, "frc1678", premoveStatusPending, "", 2, 1),
		)

	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveParkOrderBase).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveMadeOrderBase, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPrepare(`UPDATE PickMoves SET PickOrder = \$1`).
		ExpectExec().
		WithArgs(1, 10, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectPrepare(`INSERT INTO PickMoves`).
		ExpectExec().
		WithArgs(1, "frc999", premoveStatusPending, 2).
		WillReturnResult(sqlmock.NewResult(20, 1))

	mock.ExpectPrepare(`DELETE FROM PickMoves`).
		ExpectExec().
		WithArgs(1, "frc1678", premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = replacePremoves(context.Background(), db, 1, []string{"frc254", "frc999"})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesClearsQueue(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows(premoveColumns).
				AddRow(10, "frc254", premoveStatusPending, "", 1, 1).
				AddRow(11, "frc1678", premoveStatusPending, "", 2, 1).
				AddRow(12, "frc4414", premoveStatusMade, "", 3, 1),
		)

	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveParkOrderBase).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveMadeOrderBase, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPrepare(`DELETE FROM PickMoves`).
		ExpectExec().
		WithArgs(1, "frc254", premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM PickMoves`).
		WithArgs(1, "frc1678", premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, err = replacePremoves(context.Background(), db, 1, nil)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesSkipsUnavailableTeam(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(premoveColumns))

	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveParkOrderBase).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveMadeOrderBase, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectPrepare(`INSERT INTO PickMoves`).
		ExpectExec().
		WithArgs(1, "frc254", premoveStatusPending, 1).
		WillReturnResult(sqlmock.NewResult(20, 1))
	mock.ExpectExec(`INSERT INTO PickMoves`).
		WithArgs(1, "frc1699", premoveStatusPending, 2).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO PickMoves`).
		WithArgs(1, "frc1678", premoveStatusPending, 2).
		WillReturnResult(sqlmock.NewResult(21, 1))

	unavailable, err := replacePremoves(context.Background(), db, 1, []string{"frc254", "frc1699", "frc1678"})

	require.NoError(t, err)
	require.Equal(t, []string{"frc1699"}, unavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesIgnoresAlreadyMadeTeam(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`Status::text`).
		ExpectQuery().
		WithArgs(1).
		WillReturnRows(
			sqlmock.NewRows(premoveColumns).
				AddRow(10, "frc254", premoveStatusMade, "", premoveMadeOrderBase+1, 1),
		)

	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveParkOrderBase).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectPrepare(`UPDATE PickMoves p`).
		ExpectExec().
		WithArgs(1, premoveMadeOrderBase, premoveStatusPending).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectPrepare(`INSERT INTO PickMoves`).
		ExpectExec().
		WithArgs(1, "frc1678", premoveStatusPending, 1).
		WillReturnResult(sqlmock.NewResult(21, 1))

	unavailable, err := replacePremoves(context.Background(), db, 1, []string{"frc254", "frc1678"})

	require.NoError(t, err)
	require.Empty(t, unavailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAddPremoveRejectsUnavailableTeam(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`INSERT INTO PickMoves`).
		ExpectExec().
		WithArgs(1, "frc1699", premoveStatusPending, 1).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = addPremove(context.Background(), db, 1, "frc1699", 1)

	require.ErrorIs(t, err, ErrTeamNotAvailable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesRejectsDuplicateTeams(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = replacePremoves(context.Background(), db, 1, []string{"frc254", "frc254"})

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplacePremovesRejectsInvalidDraftPlayer(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	_, err = replacePremoves(context.Background(), db, 0, []string{"frc254"})

	require.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPremoveIsPending(t *testing.T) {
	require.True(t, Premove{Status: premoveStatusPending}.IsPending())
	require.True(t, Premove{Status: ""}.IsPending())
	require.False(t, Premove{Status: premoveStatusMade}.IsPending())
}
