package domain

import (
	"fmt"
	"strings"
)

type FederatedRoom struct {
	RoomName, Domain string
}

func ParseFederatedRoom(s string) (FederatedRoom, error) {
	s = strings.TrimSpace(s)
	roomName, domain, found := strings.Cut(s, "@")
	switch {
	case !found:
		return FederatedRoom{}, fmt.Errorf("invalid federated room %q: missing @", s)
	case roomName == "":
		return FederatedRoom{}, fmt.Errorf("invalid federated room %q: empty room name", s)
	case domain == "":
		return FederatedRoom{}, fmt.Errorf("invalid federated room %q: empty domain", s)
	case strings.Contains(domain, "@"):
		return FederatedRoom{}, fmt.Errorf("invalid federated room %q: multiple @ signs", s)
	}
	return FederatedRoom{RoomName: roomName, Domain: domain}, nil
}

func (f FederatedRoom) String() string {
	return fmt.Sprintf("%s@%s", f.RoomName, f.Domain)
}
