package draft

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"server/model"
)

func TestDraftPremoveIndexShowsPickedAndMadeTeams(t *testing.T) {
	pageModel := PremovePage{
		Premoves: []model.Premove{
			{Id: 1, TeamTbaId: "frc254", Status: "Pending", PickOrder: 1},
			{Id: 2, TeamTbaId: "frc1678", Status: "Pending", PickOrder: 2},
		},
		MadePremoves: []model.Premove{
			{Id: 3, TeamTbaId: "frc118", Status: "Made", PickOrder: 3},
		},
		PickedPremoves: []model.Premove{
			{Id: 4, TeamTbaId: "frc999", Status: "Pending", PickOrder: 4},
		},
		ValidTeams:   []string{"254", "999", "1678", "118"},
		PickedTeams:  []string{"frc999"},
		PendingCount: 2,
		SaveUrl:      "/u/draft/42/premoves",
		MaxPremoves:  model.PicksPerPlayer,
		Saved:        true,
		SaveWarning:  "The following teams are not available in this draft: 999.",
	}

	var buf bytes.Buffer
	err := DraftPremoveIndex(pageModel, "csrf-token").Render(context.Background(), &buf)
	require.NoError(t, err)

	rendered := buf.String()
	require.Contains(t, rendered, `data-picked-teams="frc999"`)
	require.Contains(t, rendered, `data-move-button-class="btn btn-sm btn-square border border-base-content/30 bg-base-content/10 text-white hover:bg-base-content/20 disabled:opacity-50 disabled:cursor-not-allowed"`)
	require.Contains(t, rendered, `data-valid-teams="254,999,1678,118"`)
	require.NotContains(t, rendered, `id="premove-team-error"`)
	require.NotContains(t, rendered, `/js/premove.js`)
	require.Contains(t, rendered, `id="premove-list"`)
	require.Contains(t, rendered, `id="premove-completed-list"`)
	require.Contains(t, rendered, "window.__savePremovesOnExit")
	require.Contains(t, rendered, "window.Toast.error")
	require.Contains(t, rendered, `data-toast-message="Premoves saved."`)
	require.Contains(t, rendered, `data-toast-type="success"`)
	require.Contains(t, rendered, `data-toast-message="The following teams are not available in this draft: 999."`)
	require.Contains(t, rendered, `data-toast-type="warning"`)
	require.Less(t, strings.Index(rendered, `id="add-premove"`), strings.Index(rendered, `id="premove-completed-list"`))
	require.Contains(t, rendered, `data-team="frc254"><input type="hidden" name="premoves" value="frc254"`)
	require.Contains(t, rendered, `data-team="frc1678"><input type="hidden" name="premoves" value="frc1678"`)
	require.Contains(t, rendered, `class="premove-item flex items-center gap-3 border rounded-lg px-4 py-3 bg-success/20 border-success premove-picked" data-team="frc999"`)
	require.Contains(t, rendered, `class="premove-item flex items-center gap-3 border rounded-lg px-4 py-3 bg-success/20 border-success premove-made" data-team="frc118"`)
	require.Contains(t, rendered, ">Already picked</span>")
	require.Contains(t, rendered, ">Made</span>")
	require.Contains(t, rendered, "The following teams are not available in this draft: 999.")
	require.NotContains(t, rendered, `id="premove-team-options"`)
	require.Contains(t, rendered, ">Save</button>")
	require.Contains(t, rendered, ">Clear</button>")
	require.Contains(t, rendered, "btn btn-success")
	require.NotContains(t, rendered, `data-team="frc118"><input type="hidden" name="premoves"`)
	require.NotContains(t, rendered, `data-team="frc999"><input type="hidden" name="premoves"`)
}
