package kubeheadless

import (
	"context"
	"errors"
	"net"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Membership interface {
	GetMembers(ctx context.Context) ([]string, error)
}

type KubeHeadlessMembership struct {
	// worth namespacing?
	//Prefix string
	Service    string
	Client     *kubernetes.Clientset
	UsePodName bool
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

	if khm.UsePodName {
		endpoints, err := khm.Client.CoreV1().Endpoints("default").Get(ctx, khm.Service, v1.GetOptions{})

		if err != nil {
			logrus.Error(err, "trouble looking up service")
		}

		for _, subset := range endpoints.Subsets {
			for _, address := range subset.Addresses {
				memberAddress := "http://" + address.TargetRef.Name + ":8081"
				memberList = append(memberList, memberAddress)
			}
		}
	} else {
		hosts, err := net.LookupHost(khm.Service)

		if err != nil {
			return nil, err
		}

		for i, host := range hosts {
			hosts[i] = "http://" + host + ":8081"
		}
		memberList = append(memberList, hosts...)
	}

	return memberList, nil
}
