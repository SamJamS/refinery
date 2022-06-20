package kubeheadless

import (
	"context"
	"errors"
	"net"
)

type Membership interface {
	GetMembers(ctx context.Context) ([]string, error)
}

type KubeHeadlessMembership struct {
	// worth namespacing?
	//Prefix string
	Service string
}

func (khm *KubeHeadlessMembership) validateDefaults() error {

	if khm.Service == "" {
		return errors.New("can't use KubeHeadlessMembership without specifying a Service")
	}
	return nil
}

func (khm *KubeHeadlessMembership) GetMembers(ctx context.Context) ([]string, error) {
	err := khm.validateDefaults()
	if err != nil {
		return nil, err
	}

	allMembers := make([]string, 0)
	mems, err := khm.getMembers(ctx)
	if err != nil {
		return nil, err
	}
	allMembers = append(allMembers, mems...)

	return allMembers, nil
}

func (khm *KubeHeadlessMembership) getMembers(ctx context.Context) ([]string, error) {
	memberList := make([]string, 0)
	hosts, err := net.LookupHost(khm.Service)
	if err != nil {
		return nil, err
	}
	memberList = append(memberList, hosts...)

	return memberList, nil
}
