package peer

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/honeycombio/refinery/config"
	"github.com/honeycombio/refinery/internal/kubeheadless"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	refreshCacheIntervalKube = 3 * time.Second
)

type kubeHeadlessPeers struct {
	service   *kubeheadless.KubeHeadlessMembership
	peers     []string
	peerLock  sync.Mutex
	callbacks []func()
}

func newKubeHeadlessPeers(c config.Config) (Peers, error) {
	serviceName, _ := c.GetKubeHeadlessService()
	usePodName, _ := c.GetKubeUsePodName()

	peers := &kubeHeadlessPeers{}

	if usePodName {
		// move kube client into khm to avoid putting this logic here
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}

		fmt.Println(clientset)

		peers = &kubeHeadlessPeers{
			service: &kubeheadless.KubeHeadlessMembership{
				Service:    serviceName,
				Client:     clientset,
				UsePodName: usePodName,
			},
		}
	} else {
		peers = &kubeHeadlessPeers{
			service: &kubeheadless.KubeHeadlessMembership{
				Service:    serviceName,
				UsePodName: usePodName,
			},
		}
	}

	address, err := publicAddr(c)

	if err != nil {
		return nil, err
	}

	peers.peers = append(peers.peers, address)

	go peers.watchPeers()

	return peers, nil
}

func (p *kubeHeadlessPeers) updatePeerListOnce() {
	currentPeers, err := p.service.GetMembers(context.TODO())
	if err != nil {
		logrus.Error(err)
		return
	}
	sort.Strings(currentPeers)
	p.peerLock.Lock()
	p.peers = currentPeers
	p.peerLock.Unlock()
}

func (p *kubeHeadlessPeers) GetPeers() ([]string, error) {
	p.peerLock.Lock()
	defer p.peerLock.Unlock()
	retList := make([]string, len(p.peers))
	copy(retList, p.peers)
	return retList, nil
}

func (p *kubeHeadlessPeers) RegisterUpdatedPeersCallback(cb func()) {
	p.callbacks = append(p.callbacks, cb)
}

func (p *kubeHeadlessPeers) watchPeers() {
	oldPeerList := p.peers
	sort.Strings(oldPeerList)
	tk := time.NewTicker(refreshCacheIntervalKube)

	for range tk.C {
		currentPeers, err := p.service.GetMembers(context.TODO())
		if err != nil {
			logrus.WithError(err).Warn("Unable to lookup headless service members")
			continue
		}
		sort.Strings(currentPeers)
		if !equal(oldPeerList, currentPeers) {
			p.peerLock.Lock()
			p.peers = currentPeers
			oldPeerList = currentPeers
			p.peerLock.Unlock()
			for _, callback := range p.callbacks {
				// don't block on any of the callbacks.
				go callback()
			}
		}
	}
}
