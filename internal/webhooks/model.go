package webhooks

type WebhookChirpyRequest struct {
	EVENT string            `json:"event"`
	DATA  webhookChirpyData `json:"data"`
}

type webhookChirpyData struct {
	USERID string `json:"user_id"`
}
