package access

type AccessState string

const (
	AccessTrial   AccessState = "trial"
	AccessFull    AccessState = "full"
	AccessLimited AccessState = "limited"
	AccessLocked  AccessState = "locked"
)

type PublicMode string

const (
	PublicFull    PublicMode = "full"
	PublicLimited PublicMode = "limited"
)

type EditorMode string

const (
	EditorFull    EditorMode = "full"
	EditorLimited EditorMode = "limited"
)
