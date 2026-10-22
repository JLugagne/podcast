package domain

import "github.com/google/uuid"

// ListenerID is the typed identifier of a listener.
type ListenerID string

// LocalListenerID is the identifier of the single local listener. The listener
// id is modelled explicitly so multi-user support can arrive without changing
// the progress aggregate.
const LocalListenerID ListenerID = "local"

// EpisodeRef is the opaque catalog episode identifier tracked by the listening
// context. It is parsed as a UUID so malformed path values fail early.
type EpisodeRef string

// ParseEpisodeRef validates and canonicalizes a catalog episode identifier.
func ParseEpisodeRef(raw string) (EpisodeRef, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return "", ErrInvalidID
	}
	return EpisodeRef(id.String()), nil
}
