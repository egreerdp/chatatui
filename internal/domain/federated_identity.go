package domain

import (
	"fmt"
	"strings"
)

type FederatedIdentity struct {
	Username, Domain string
}

func ParseFederatedIdentity(s string) (FederatedIdentity, error) {
	s = strings.TrimSpace(s)
	username, domain, found := strings.Cut(s, "@")
	switch {
	case !found:
		return FederatedIdentity{}, fmt.Errorf("invalid federated identity %q: missing @", s)
	case username == "":
		return FederatedIdentity{}, fmt.Errorf("invalid federated identity %q: empty username", s)
	case domain == "":
		return FederatedIdentity{}, fmt.Errorf("invalid federated identity %q: empty domain", s)
	case strings.Contains(domain, "@"):
		return FederatedIdentity{}, fmt.Errorf("invalid federated identity %q: multiple @ signs", s)
	}
	return FederatedIdentity{Username: username, Domain: domain}, nil
}

func (f FederatedIdentity) String() string {
	return fmt.Sprintf("%s@%s", f.Username, f.Domain)
}

func (f FederatedIdentity) IsLocal(localDomain string) bool {
	return f.Domain == localDomain
}
