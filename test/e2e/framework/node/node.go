//go:build e2e
// +build e2e

package node

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
)

const (
	// nodeLabelRole specifies the role of a node
	nodeLabelRole = "kubernetes.io/role"
)

// IsMasterNode returns true if the node has a master role label.
// The master role is determined by looking for:
// * a kubernetes.io/role="master" label
func IsMasterNode(node v1.Node) bool {
	By(fmt.Sprintf("Checking node \"%s\" is master", node.Name))
	if node.Labels == nil {
		return false
	}
	return node.Labels[nodeLabelRole] == "master"
}
