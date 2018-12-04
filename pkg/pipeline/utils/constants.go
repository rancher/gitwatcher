package utils

const (
	PipelineName            = "pipeline"
	DefaultRegistry         = "index.docker.io"
	WebhookEventPush        = "push"
	WebhookEventPullRequest = "pull_request"
	WebhookEventTag         = "tag"

	TriggerTypeWebhook = "webhook"

	StateWaiting        = "Waiting"
	StateBuilding       = "Building"
	StateSuccess        = "Success"
	StateFailed         = "Failed"
	StatePending        = "Pending"
	PipelineFinishLabel = "pipeline.project.cattle.io/finish"
	PipelineFileYml     = ".rancher-pipeline.yml"
	PipelineFileYaml    = ".rancher-pipeline.yaml"

	DefaultTimeout = 60
)
