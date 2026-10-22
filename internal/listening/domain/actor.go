package domain

// ActorKind tells which kind of principal performs an action.
type ActorKind string

const (
	// ActorKindUser is a human principal.
	ActorKindUser ActorKind = "user"
	// ActorKindSystem is an automated principal.
	ActorKindSystem ActorKind = "system"
)

// Actor is the principal performing a use case. The local listener app has no
// authentication, so the HTTP layer always injects LocalListener.
type Actor struct {
	id   string
	kind ActorKind
}

// LocalListener returns the single local listener actor.
func LocalListener() *Actor {
	return &Actor{id: string(LocalListenerID), kind: ActorKindUser}
}

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
