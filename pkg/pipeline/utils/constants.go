package utils

const (
	WebhookEventPush        = "push"
	WebhookEventPullRequest = "pull_request"
	WebhookEventTag         = "tag"

	TriggerTypeWebhook = "webhook"

	StatusContext       = "continuous-integration/rancher"
	HooksEndpointPrefix = "hooks?gitwebhookId="
	GitWebHookParam     = "gitwebhookId"
)
