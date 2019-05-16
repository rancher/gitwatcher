package main

import (
	"os"

	v1 "github.com/rancher/gitwatcher/pkg/apis/gitwatcher.cattle.io/v1"
	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"
)

var (
	basePackage = "github.com/rancher/rio/types"
)

func main() {
	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/gitwatcher/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"gitwatcher.cattle.io": {
				Types: []interface{}{
					v1.GitWatcher{},
					v1.GitCommit{},
				},
				GenerateTypes: true,
			},
		},
	})
}
