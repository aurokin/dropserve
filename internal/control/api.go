package control

type CreatePortalRequest struct {
	DestAbs              string `json:"dest_abs"`
	OpenMinutes          int    `json:"open_minutes"`
	Reusable             bool   `json:"reusable"`
	DefaultPolicy        string `json:"default_policy"`
	AutorenameOnConflict bool   `json:"autorename_on_conflict"`
}

type CreatePortalResponse struct {
	PortalID  string `json:"portal_id"`
	ExpiresAt string `json:"expires_at"`
}
