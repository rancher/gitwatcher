package polling

import (
	"context"
	"net/http"

	webhookv1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	"github.com/rancher/gitwatcher/pkg/git"
	corev1controller "github.com/rancher/wrangler-api/pkg/generated/controllers/core/v1"
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
	defaultSecretName = "gitcredential"
)

type Polling struct {
	secretCache corev1controller.SecretCache
	apply       apply.Apply
}

func NewPolling(secrets v12.SecretCache, apply apply.Apply) *Polling {
	return &Polling{
		secretCache: secrets,
		apply:       apply.WithStrictCaching(),
	}
}

func (w *Polling) Supports(obj *webhookv1.GitWatcher) bool {
	return obj.Spec.Branch != ""
}

func (w *Polling) Create(ctx context.Context, obj *webhookv1.GitWatcher) (*webhookv1.GitWatcher, error) {
	var (
		auth git.Auth
	)

	secretName := defaultSecretName
	if obj.Spec.RepositoryCredentialSecretName != "" {
		secretName = obj.Spec.RepositoryCredentialSecretName
	}
	secret, err := w.secretCache.Get(obj.Namespace, secretName)
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

func (w *Polling) HandleHook(ctx context.Context, req *http.Request) (int, error) {
	return 0, nil
}

func ApplyCommit(obj *webhookv1.GitWatcher, commit string, apply apply.Apply) error {
	gitCommit := webhookv1.NewGitCommit(obj.Namespace, name.SafeConcatName(obj.Name, name.Hex(commit, 5)), webhookv1.GitCommit{
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
			Branch:         obj.Spec.Branch,
			RepositoryURL:  obj.Spec.RepositoryURL,
			Commit:         commit,
			GitWatcherName: obj.Name,
		},
	})
	os := objectset.NewObjectSet()
	os.Add(gitCommit)
	return apply.WithSetID("gitcommit").WithOwner(obj).WithNoDelete().Apply(os)
}
