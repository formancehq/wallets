package testserver

import (
	"github.com/formancehq/go-libs/v2/testing/deferred"
	. "github.com/onsi/ginkgo/v2"
)

func NewTestServer(configurationProvider func() Configuration) *deferred.Deferred[*Server] {
	d := deferred.New[*Server]()
	BeforeEach(func() {
		d.Reset()
		d.SetValue(New(GinkgoT(), configurationProvider()))
	})
	return d
}
