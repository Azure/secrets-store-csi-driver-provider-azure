package converter

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestPromMdmConverter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PromMdmConverter Suite")
}
