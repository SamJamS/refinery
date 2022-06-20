package peer

import (
	"context"
	"fmt"
	"time"

	"github.com/honeycombio/refinery/config"
	"github.com/honeycombio/refinery/internal/kubeheadless"
	"github.com/sirupsen/logrus"
)

const (
	refreshCacheIntervalKube = 3 * time.Second
)

type kubeHeadlessPeers struct {
	serviceConfig *kubeheadless.KubeHeadlessMembership
	peers         []string
}

func newKubeHeadlessPeers(c config.Config) (Peers, error) {
	service, _ := c.GetKubeHeadlessService()

	peers := &kubeHeadlessPeers{
		serviceConfig: &kubeheadless.KubeHeadlessMembership{
			Service: service,
		},
	}

	address, err := publicAddr(c)
	if err != nil {
		return nil, err
	}
	peers.peers = append(peers.peers, address)

	go peers.watchPeers()

	return peers, nil
}

func (p *kubeHeadlessPeers) GetPeers() ([]string, error) {
	return p.peers, nil
}

func (p *kubeHeadlessPeers) RegisterUpdatedPeersCallback(cb func()) {
	fmt.Println("do something")
}

func (p *kubeHeadlessPeers) watchPeers() {
	tk := time.NewTicker(refreshCacheIntervalKube)

	for range tk.C {
		currentPeers, err := p.serviceConfig.GetMembers(context.TODO())
		if err != nil {
			logrus.WithError(err).Warn("Unable to lookup headless service members")
			continue
		}
		p.peers = currentPeers
		logrus.Info(p.peers)
	}
}
