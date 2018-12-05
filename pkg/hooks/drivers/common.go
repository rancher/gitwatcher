package drivers

import (
	"fmt"
	"net/http"

	"github.com/rancher/webhookinator/pkg/providers/model"
	"github.com/rancher/webhookinator/pkg/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

const (
	RefsBranchPrefix = "refs/heads/"
	RefsTagPrefix    = "refs/tags/"
)

func validateAndGenerateExecution(
	gitWebHookExecutions v1.GitWebHookExecutionInterface,
	info *model.BuildInfo,
	receiver *v1.GitWebHookReceiver) (int, error) {
	if !isEventActivated(info, receiver) {
		return http.StatusUnavailableForLegalReasons, fmt.Errorf("trigger for event '%s' is disabled", info.Event)
	}

	execution := initExecution(info, receiver)
	_, err := gitWebHookExecutions.Create(execution)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}

func isEventActivated(info *model.BuildInfo, receiver *v1.GitWebHookReceiver) bool {
	if (info.Event == utils.WebhookEventPush && receiver.Spec.Push) ||
		(info.Event == utils.WebhookEventTag && receiver.Spec.Tag) ||
		(info.Event == utils.WebhookEventPullRequest && receiver.Spec.PR) {
		return true
	}
	return false
}

func initExecution(info *model.BuildInfo, receiver *v1.GitWebHookReceiver) *v1.GitWebHookExecution {
	execution := &v1.GitWebHookExecution{}
	execution.Spec.Branch = info.Branch
	execution.Spec.Author = info.Author
	execution.Spec.AuthorAvatar = info.AvatarURL
	execution.Spec.AuthorEmail = info.Email
	execution.Spec.Message = info.Message
	execution.Spec.SourceLink = info.HTMLLink
	execution.Spec.Title = info.Title
	execution.Spec.Commit = info.Commit
	execution.Spec.RepositoryURL = receiver.Spec.RepositoryURL
	if info.RepositoryURL != "" {
		execution.Spec.RepositoryURL = info.RepositoryURL
	}
	return execution
}
