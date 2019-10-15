module github.com/rancher/gitwatcher

go 1.12

replace github.com/matryer/moq => github.com/rancher/moq v0.0.0-20190404221404-ee5226d43009

require (
	github.com/google/go-github/v28 v28.0.0
	github.com/google/uuid v1.1.1
	github.com/gorilla/mux v1.7.1
	github.com/pkg/errors v0.8.1
	github.com/rancher/wrangler v0.2.1-0.20191015042916-f2a6ecca4f20
	github.com/rancher/wrangler-api v0.2.1-0.20191015045805-d3635aa0853a
	github.com/sirupsen/logrus v1.4.2
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/urfave/cli v1.20.0
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
)
