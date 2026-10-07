package infra

import (
	"context"
	"errors"
	"time"
)

// LDAPEntry mimics an entry returned by an LDAP search.
type LDAPEntry struct {
	UID          string
	DisplayName  string
	CreationDate time.Time
}

// errNoSuchObject mimics the error a real LDAP library returns when the
// search base holds no matching entry.
var errNoSuchObject = errors.New("ldap: no such object")

// LDAPClient is a stub of a real LDAP client library (such as go-ldap/ldap).
// It keeps its directory entries in memory so the workshop has zero external
// dependencies, while still behaving like the real thing: Search resolves a
// UID to an entry or returns a driver-style error.
type LDAPClient struct {
	entries map[string]LDAPEntry
}

func NewLDAPClient(entries map[string]LDAPEntry) *LDAPClient {
	return &LDAPClient{entries: entries}
}

// Search resolves a UID to a directory entry.
func (c *LDAPClient) Search(ctx context.Context, uid string) (LDAPEntry, error) {
	entry, ok := c.entries[uid]
	if !ok {
		return LDAPEntry{}, errNoSuchObject
	}
	return entry, nil
}
