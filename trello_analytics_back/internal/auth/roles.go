package auth

// Application roles (ADR-012). Hierarchy: Admin ⊃ Team Lead ⊃ Member.
const (
	RoleAdmin    = "admin"
	RoleTeamLead = "team_lead"
	RoleMember   = "member"
)

var roleRank = map[string]int{
	RoleMember:   1,
	RoleTeamLead: 2,
	RoleAdmin:    3,
}

// ValidRole reports whether role is one of the known v1 roles.
func ValidRole(role string) bool {
	_, ok := roleRank[role]
	return ok
}

// RoleAtLeast reports whether have satisfies the minimum needed role.
func RoleAtLeast(have, need string) bool {
	hr, okH := roleRank[have]
	nr, okN := roleRank[need]
	if !okH || !okN {
		return false
	}
	return hr >= nr
}
