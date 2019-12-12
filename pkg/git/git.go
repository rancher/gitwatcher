package git

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/sirupsen/logrus"
)

func BranchCommit(ctx context.Context, url string, branch string, auth *Auth) (string, error) {
	url, env, close := auth.Populate(url)
	defer close()

	lines, err := git(ctx, env, "ls-remote", url, formatRefForBranch(branch))
	if err != nil {
		return "", err
	}

	return firstField(lines, fmt.Sprintf("no commit for branch: %s", branch))
}

func CloneRepo(ctx context.Context, url string, commit string, auth *Auth) error {
	url, env, close := auth.Populate(url)
	defer close()

	lines, err := git(ctx, env, "clone", "-n", url, ".")
	if err != nil {
		return err
	}

	logrus.Infof("Output from git clone %v", lines)

	lines, err = git(ctx, env, "checkout", commit)
	if err != nil {
		return err
	}

	logrus.Infof("Output from git checkout %v", lines)

	return nil
}

// returns nil if tag qualifies, otherwise returns specific error
func TagMatch(query, exclude, tagRef string) error {
	if query != "" {
		match, err := regexp.MatchString(query, tagRef)
		if err != nil {
			return err
		}
		if match == false {
			return errors.New("tag ref did not match tag query")
		}
	}
	if exclude != "" {
		excludeMatch, err := regexp.MatchString(exclude, tagRef)
		if err != nil {
			return err
		}
		if excludeMatch == true {
			return errors.New("tag ref matched exclude query")
		}
	}
	return nil
}
