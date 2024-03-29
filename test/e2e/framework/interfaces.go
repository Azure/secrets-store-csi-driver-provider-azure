//go:build e2e
// +build e2e

package framework

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Interfaces to scope down client.Client

// Getter can get resources.
type Getter interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
}

// Creator can create resources.
type Creator interface {
	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
}

// Lister can list resources.
type Lister interface {
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

// Deleter can delete resources.
type Deleter interface {
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

// Updater can update resources.
type Updater interface {
	Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error
}

// GetLister can get and list resources.
type GetLister interface {
	Getter
	Lister
}
