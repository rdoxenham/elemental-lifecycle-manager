package release

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Release manifest", func() {
	It("Fails to resolve non-existing release manifest", func() {
		ctx := context.Background()

		manifest, err := RetrieveManifest(ctx, "registry.example.com/release-manifest", "0.5")
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(ContainSubstring("fetching remote image registry.example.com/release-manifest:0.5")))
		Expect(manifest).To(BeNil())
	})
})
