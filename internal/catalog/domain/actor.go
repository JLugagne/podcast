package domain

// LocalAdminID is the stable identifier of the single local administrator.
const LocalAdminID = "local-admin"

// ActorKind tells which kind of principal performs an action.
type ActorKind string

const (
	// ActorKindUser is a human principal.
	ActorKindUser ActorKind = "user"
	// ActorKindSystem is an automated principal.
	ActorKindSystem ActorKind = "system"
)

// Actor is the principal performing a use case. Podcat has no authentication:
// the HTTP layer always injects LocalAdmin.
type Actor struct {
	id   string
	kind ActorKind
}

// LocalAdmin returns the single local administrator actor.
func LocalAdmin() *Actor { return &Actor{id: LocalAdminID, kind: ActorKindUser} }

// NewActor builds an actor from trusted input.
func NewActor(id string, kind ActorKind) *Actor { return &Actor{id: id, kind: kind} }

// ID returns the actor identifier, or an empty string for a nil actor.
func (a *Actor) ID() string {
	if a == nil {
		return ""
	}
	return a.id
}

// Kind returns the actor kind, or an empty string for a nil actor.
func (a *Actor) Kind() ActorKind {
	if a == nil {
		return ""
	}
	return a.kind
}
