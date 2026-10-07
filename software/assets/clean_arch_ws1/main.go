package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/adapters/infra"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/domain"
	"github.com/sokoide/workshop/software/assets/clean_arch_ws1/internal/usecase"
)

func main() {
	ctx := context.Background()

	// Phase 1: users are served by the SQL database.
	checkWith(ctx, "SQL database", infra.NewSQLUserRepository())

	// Phase 2: policy change — the user directory moved to Active Directory.
	// Only this wiring changes; internal/domain and internal/usecase are untouched.
	now := time.Now()
	directory := infra.NewLDAPClient(map[string]infra.LDAPEntry{
		"alice": {UID: "alice", DisplayName: "Alice", CreationDate: now.AddDate(-10, 0, 0)},
		"bob":   {UID: "bob", DisplayName: "Bob", CreationDate: now.AddDate(-1, 0, 0)},
	})
	checkWith(ctx, "Active Directory (LDAP)", infra.NewADUserRepository(directory))
}

// checkWith wires the given repository into the use case and prints the
// veteran verdicts. It works with ANY implementation of domain.UserRepository.
func checkWith(ctx context.Context, source string, repo domain.UserRepository) {
	fmt.Printf("== data source: %s ==\n", source)
	uc := usecase.NewCheckVeteranUseCase(repo, domain.VeteranService{})
	for _, id := range []string{"alice", "bob"} {
		veteran, err := uc.Execute(ctx, id)
		if err != nil {
			fmt.Printf("  %s: error: %v\n", id, err)
			continue
		}
		fmt.Printf("  %s: veteran=%v\n", id, veteran)
	}
}
