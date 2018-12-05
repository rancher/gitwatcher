package drivers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/rancher/rancher/pkg/ref"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/bitbucketserver"
	"github.com/rancher/webhookinator/pkg/pipeline/remote/model"
	"github.com/rancher/webhookinator/pkg/pipeline/utils"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
)

const (
	BitbucketServerWebhookHeader  = "X-Request-Id"
	bitbucketServerEventHeader    = "X-Event-Key"
	bitbucketServerPushEvent      = "repo:refs_changed"
	bitbucketServerPrCreatedEvent = "pr:opened"
	bitbucketServerPrUpdatedEvent = "pr:modified"
	bitbucketServerStateOpen      = "OPEN"
)

type BitbucketServerDriver struct {
	GitWebHookReceiverLister v1.GitWebHookReceiverLister
	GitWebHookExecutions     v1.GitWebHookExecutionInterface
}

func (b BitbucketServerDriver) Execute(req *http.Request) (int, error) {
	var signature string
	if signature = req.Header.Get(githubSignatureHeader); len(signature) == 0 {
		return http.StatusUnprocessableEntity, errors.New("webhook missing signature")
	}
	event := req.Header.Get(bitbucketServerEventHeader)
	if event != bitbucketServerPushEvent && event != bitbucketServerPrCreatedEvent && event != bitbucketServerPrUpdatedEvent {
		return http.StatusUnprocessableEntity, fmt.Errorf("not trigger for event:%s", event)
	}

	receiverID := req.URL.Query().Get(utils.GitWebHookParam)
	ns, name := ref.Parse(receiverID)
	receiver, err := b.GitWebHookReceiverLister.Get(ns, name)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return http.StatusUnprocessableEntity, err
	}
	if match := verifyBitbucketServerWebhookSignature([]byte(receiver.Status.Token), signature, body); !match {
		return http.StatusUnprocessableEntity, errors.New("invalid signature")
	}

	info := &model.BuildInfo{}
	if event == bitbucketServerPushEvent {
		info, err = parseBitbucketServerPushPayload(body)
		if err != nil {
			return http.StatusUnprocessableEntity, err
		}
	} else if event == bitbucketServerPrCreatedEvent || event == bitbucketServerPrUpdatedEvent {
		info, err = parseBitbucketServerPullRequestPayload(body)
		if err != nil {
			return http.StatusUnprocessableEntity, err
		}
	}

	return validateAndGenerateExecution(b.GitWebHookExecutions, info, receiver)
}

func parseBitbucketServerPushPayload(raw []byte) (*model.BuildInfo, error) {
	info := &model.BuildInfo{}
	payload := bitbucketserver.PushEventPayload{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	info.TriggerType = utils.TriggerTypeWebhook

	if len(payload.Changes) > 0 {
		change := payload.Changes[0]
		info.Commit = change.ToHash
		info.Ref = change.RefID
		//info.Message = change.New.Target.Message
		info.Author = payload.Actor.Name
		if len(payload.Actor.Links.Self) > 0 {
			info.AvatarURL = payload.Actor.Links.Self[0].Href + "/avatar.png"
		}

		if strings.HasPrefix(change.RefID, RefsTagPrefix) {
			//git tag is triggered as a push event
			info.Event = utils.WebhookEventTag
			info.Branch = strings.TrimPrefix(change.RefID, RefsTagPrefix)
			if change.Type != "ADD" {
				return nil, fmt.Errorf("filter '%s' changes for tag event", change.Type)
			}
		} else {
			info.Event = utils.WebhookEventPush
			info.Branch = strings.TrimPrefix(change.RefID, RefsBranchPrefix)
			if change.Type != "UPDATE" && change.Type != "ADD" {
				return nil, fmt.Errorf("filter '%s' changes for push event", change.Type)
			}
		}
	}
	return info, nil
}

func parseBitbucketServerPullRequestPayload(raw []byte) (*model.BuildInfo, error) {
	info := &model.BuildInfo{}
	payload := bitbucketserver.PullRequestEventPayload{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}

	if payload.PullRequest.State != bitbucketServerStateOpen {
		return nil, fmt.Errorf("no trigger for closed pull requests")
	}

	info.TriggerType = utils.TriggerTypeWebhook
	info.Event = utils.WebhookEventPullRequest
	info.Branch = payload.PullRequest.ToRef.DisplayID
	info.Ref = fmt.Sprintf("refs/pull-requests/%d/from", payload.PullRequest.ID)
	if len(payload.PullRequest.Links.Self) > 0 {
		info.HTMLLink = payload.PullRequest.Links.Self[0].Href
	}
	info.Title = payload.PullRequest.Title
	info.Message = payload.PullRequest.Title
	info.Commit = payload.PullRequest.FromRef.LatestCommit
	info.Author = payload.PullRequest.Author.User.Name
	if len(payload.PullRequest.Author.User.Links.Self) > 0 {
		info.AvatarURL = payload.PullRequest.Author.User.Links.Self[0].Href + "/avatar.png"
	}
	return info, nil
}

func verifyBitbucketServerWebhookSignature(secret []byte, signature string, body []byte) bool {
	const signaturePrefix = "sha256="
	const signatureLength = 71 // len(SignaturePrefix) + len(hex(sha1))
	if len(signature) != signatureLength || !strings.HasPrefix(signature, signaturePrefix) {
		return false
	}
	actual := make([]byte, 32)
	hex.Decode(actual, []byte(signature[7:]))
	computed := hmac.New(sha256.New, secret)
	computed.Write(body)
	return hmac.Equal(computed.Sum(nil), actual)
}
