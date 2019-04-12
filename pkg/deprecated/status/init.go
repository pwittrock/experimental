package status

import (
	"github.com/google/wire"
	"k8s.io/client-go/kubernetes"
)

var ProviderSet = wire.NewSet(NewLister, NewProvider)

func NewLister(provider *Provider) *Lister {
	return &Lister{Provider: provider}
}

func NewProvider(client *kubernetes.Clientset) *Provider {
	return &Provider{Client: client}
}
