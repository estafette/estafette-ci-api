package slackapi

// SlashCommand represents a slash command from Slack
type SlashCommand struct {
	Token          string `form:"token"`
	TeamID         string `form:"team_id"`
	TeamDomain     string `form:"team_domain"`
	EnterpriseID   string `form:"enterprise_id"`
	EnterpriseName string `form:"enterprise_name"`
	ChannelID      string `form:"channel_id"`
	ChannelName    string `form:"channel_name"`
	UserID         string `form:"user_id"`
	UserName       string `form:"user_name"`
	Command        string `form:"command"`
	Text           string `form:"text"`
	ResponseURL    string `form:"response_url"`
}

// GetUserProfileResponse represents the api response for retrieving a
type GetUserProfileResponse struct {
	OK      bool         `json:"ok"`
	Profile *UserProfile `json:"profile"`
	Error   string       `json:"error"`
}

// UserProfile represents a Slack user profile
type UserProfile struct {
	Email string `json:"email"`
}
