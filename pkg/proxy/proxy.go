package proxy

import (
	"context"

	"github.com/protomesh/protomesh"
	"github.com/protomesh/protomesh/pkg/resource"
	servicesv1 "github.com/protomesh/protomesh/proto/api/services/v1"
	typesv1 "github.com/protomesh/protomesh/proto/api/types/v1"
)

type ProxyHandler interface {
	// Context, active nodes and dropped nodes
	ProcessNodes(context.Context, []*typesv1.NetworkingNode, []*typesv1.NetworkingNode) error
}

type ProxyDependency interface {
	GetResourceStoreClient() servicesv1.ResourceStoreClient
}

type Proxy[D ProxyDependency] struct {
	*protomesh.Injector[D]

	SyncInterval           protomesh.Config `config:"sync.interval,duration" default:"60s" usage:"Interval between synchronization cycles"`
	ResourceStoreNamespace protomesh.Config `config:"resource.store.namespace,str" default:"default" usage:"Resource store namespace to use"`

	handlers []ProxyHandler

	updated []*typesv1.NetworkingNode
	dropped []*typesv1.NetworkingNode
}

func (ep *Proxy[D]) Initialize(handlers ...ProxyHandler) {

	ep.handlers = handlers

}

func (ep *Proxy[D]) BeforeBatch(ctx context.Context) error {

	ep.updated = []*typesv1.NetworkingNode{}
	ep.dropped = []*typesv1.NetworkingNode{}

	return nil

}

func (ep *Proxy[D]) OnUpdated(ctx context.Context, updatedRes *typesv1.Resource) error {

	edge := new(typesv1.NetworkingNode)

	err := updatedRes.Spec.UnmarshalTo(edge)
	if err != nil {
		return err
	}

	ep.updated = append(ep.updated, edge)

	return nil

}

func (ep *Proxy[D]) OnDropped(ctx context.Context, droppedRes *typesv1.Resource) error {

	edge := new(typesv1.NetworkingNode)

	err := droppedRes.Spec.UnmarshalTo(edge)
	if err != nil {
		return err
	}

	ep.dropped = append(ep.dropped, edge)

	return nil

}

func (ep *Proxy[D]) AfterBatch(ctx context.Context) error {

	for _, handler := range ep.handlers {

		if err := handler.ProcessNodes(ctx, ep.updated, ep.dropped); err != nil {
			return err
		}

	}

	return nil

}

func (ep *Proxy[D]) Sync(ctx context.Context) <-chan error {

	sync := &resource.ResourceStoreSynchronizer[D]{
		Injector:     ep.Injector,
		SyncInterval: ep.SyncInterval.DurationVal(),
		Namespace:    ep.ResourceStoreNamespace.StringVal(),
		IndexCursor:  0,

		EventHandler: ep,
	}

	return sync.Sync(ctx)

}
