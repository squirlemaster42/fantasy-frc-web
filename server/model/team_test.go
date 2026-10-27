package model

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestGetTeams(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	mock.ExpectPrepare(`From Teams`).
		ExpectQuery().
		WillReturnRows(
			sqlmock.NewRows([]string{"tbaId", "name", "allianceScore"}).
				AddRow("frc254", "The Cheesy Poofs", 10).
				AddRow("frc1678", "Citrus Circuits", 5),
		)

	teams, err := getTeams(context.Background(), db)

	require.NoError(t, err)
	require.Len(t, teams, 2)
	require.Equal(t, "frc254", teams[0].TbaId)
	require.Equal(t, "The Cheesy Poofs", teams[0].Name)
	require.Equal(t, 10, teams[0].AllianceScore)
	require.NoError(t, mock.ExpectationsWereMet())
}
