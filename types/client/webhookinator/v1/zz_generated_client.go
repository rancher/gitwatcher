package client

import (
	"github.com/rancher/norman/clientbase"
)

type Client struct {
	clientbase.APIBaseClient

	GitWebHookReceiver  GitWebHookReceiverOperations
	GitWebHookExecution GitWebHookExecutionOperations
}

func NewClient(opts *clientbase.ClientOpts) (*Client, error) {
	baseClient, err := clientbase.NewAPIClient(opts)
	if err != nil {
		return nil, err
	}

	client := &Client{
		APIBaseClient: baseClient,
	}

	client.GitWebHookReceiver = newGitWebHookReceiverClient(client)
	client.GitWebHookExecution = newGitWebHookExecutionClient(client)

	return client, nil
}
