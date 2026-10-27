package draft

const (
	premoveItemBaseClass = "premove-item flex items-center gap-3 border rounded-lg px-4 py-3 "

	premoveMoveButtonClass = "btn btn-sm btn-square border border-base-content/30 bg-base-content/10 text-white hover:bg-base-content/20 disabled:opacity-50 disabled:cursor-not-allowed"
)

func premoveRowClass(made bool, picked bool) string {
	switch {
	case made:
		return premoveItemBaseClass + "bg-success/20 border-success premove-made"
	case picked:
		return premoveItemBaseClass + "bg-success/20 border-success premove-picked"
	default:
		return premoveItemBaseClass + "bg-base-100 border-base-300"
	}
}

func premoveLabelClass(made bool, picked bool) string {
	switch {
	case made, picked:
		return "font-mono text-lg font-semibold flex-1 text-success"
	default:
		return "font-mono text-lg font-semibold flex-1 text-base-content"
	}
}
