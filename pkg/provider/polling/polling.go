package polling

import (
	"context"
	"net/http"

	"github.com/drone/go-scm/scm"
	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	v1 "github.com/rancher/gitwatcher/pkg/generated/controllers/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/git"
	"github.com/rancher/gitwatcher/pkg/provider/scmprovider"
	v12 "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/name"
	"github.com/rancher/wrangler/pkg/objectset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	T = true
)

const (
	secretName = "gitcredential"
)

type Polling struct {
	scmprovider.SCM
	apply apply.Apply
}

func NewPolling(secrets v12.SecretCache, apply apply.Apply) *Polling {
	return &Polling{
		SCM: scmprovider.SCM{
			SecretsCache: secrets,
		},
		apply: apply.WithStrictCaching(),
	}
}

func (w *Polling) Supports(obj *webhookv1.GitWatcher) bool {
	return obj.Spec.Branch != ""
}

func (w *Polling) Create(ctx context.Context, obj *webhookv1.GitWatcher) (*webhookv1.GitWatcher, error) {
	var (
		auth git.Auth
	)

	secret, err := w.GetSecret(secretName, obj)
	if errors.IsNotFound(err) {
		secret = nil
	} else if err != nil {
		return obj, err
	}

	if secret != nil {
		auth, _ = git.FromSecret(secret.Data)
	}

	commit, err := git.BranchCommit(ctx, obj.Spec.RepositoryURL, obj.Spec.Branch, &auth)
	if err != nil {
		return obj, err
	}

	err = ApplyCommit(obj, commit, w.apply)
	if err != nil {
		return obj, err
	}

	if obj.Status.FirstCommit == "" {
		obj = obj.DeepCopy()
		obj.Status.FirstCommit = commit
	}

	return obj, nil
}

func (w *Polling) HandleHook(gitWatchers v1.GitWatcherCache, req *http.Request) (*webhookv1.GitWatcher, scm.Webhook, bool, int, error) {
	return nil, nil, false, 0, nil
}

func ApplyCommit(obj *webhookv1.GitWatcher, commit string, apply apply.Apply) error {
	gitCommit := webhookv1.NewGitCommit(obj.Namespace, name.SafeConcatName(obj.Name, commit), webhookv1.GitCommit{
		ObjectMeta: metav1.ObjectMeta{
			Labels: obj.Spec.ExecutionLabels,
			OwnerReferences: []metav1.OwnerReference{
				{
					Name:       obj.Name,
					Kind:       "GitWatcher",
					APIVersion: "gitwatcher.cattle.io/v1",
					Controller: &T,
					UID:        obj.UID,
				},
			},
		},
		Spec: webhookv1.GitCommitSpec{
			Branch:                 obj.Spec.Branch,
			RepositoryURL:          obj.Spec.RepositoryURL,
			Commit:                 commit,
			GitWebHookReceiverName: obj.Name,
		},
	})
	os := objectset.NewObjectSet()
	os.Add(gitCommit)
	return apply.WithSetID("gitcommit").WithOwner(obj).WithNoDelete().Apply(os)
}
