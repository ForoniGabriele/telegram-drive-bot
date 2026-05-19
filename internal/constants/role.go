package constants

// Role is the user permission role.
type Role string

const (
	RoleOwner Role = "owner"
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) String() string { return string(r) }

// IsAdminOrOwner reports whether the role has admin-level privileges.
func (r Role) IsAdminOrOwner() bool {
	return r == RoleAdmin || r == RoleOwner
}

// IsOwner reports whether the role is the owner.
func (r Role) IsOwner() bool { return r == RoleOwner }
