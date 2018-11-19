package main

import (
	"github.com/rancher/norman/generator"
	"github.com/rancher/webhookinator/types/apis/webhookinator.cattle.io/v1"
	"github.com/sirupsen/logrus"
)

func main() {
	if err := generator.DefaultGenerate(v1.Schemas, "github.com/rancher/webhookinator/types", true, nil); err != nil {
		logrus.Fatal(err)
	}
}
