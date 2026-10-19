package content

import "regexp"

// ProfileUID: built-in content type for the user profile (entry ID = user ID). Its fields are set by the
// superadmin in the builder (e.g. relations to Site/Organization), so users can be linked to any master.
const ProfileUID = "user-profile"

// ExternalIDRe: externally assigned entry ID (e.g. user ID) — letters/digits/_/- only.
var ExternalIDRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
