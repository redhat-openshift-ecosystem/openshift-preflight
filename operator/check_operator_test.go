package operator

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	preflighterr "github.com/redhat-openshift-ecosystem/openshift-preflight/errors"
)

var _ = Describe("Operator Check initialization", func() {
	When("Using options to initialize a check", func() {
		It("Should properly store the options with their correct values", func() {
			image := "placeholder"
			kubeconfigContents := "kubeconfig contents"
			kubeconfig := []byte(kubeconfigContents)
			indeximage := "indeximage:latest"
			scorecardImage := "scorecardimage:latest"
			scorecardNamespace := "scorecardnamespace"
			scorecardServiceAccount := "scorecardserviceaccount"
			scorecardWaitTime := "scorecardwaittime"
			operatorChannel := "operatorchannel"
			dockerConfigFilePath := "dockerconfigfilepath"
			insecure := true
			c := NewCheck(image, indeximage, kubeconfig,
				WithScorecardImage(scorecardImage),
				WithScorecardNamespace(scorecardNamespace),
				WithScorecardServiceAccount(scorecardServiceAccount),
				WithScorecardWaitTime(scorecardWaitTime),
				WithOperatorChannel(operatorChannel),
				WithDockerConfigJSONFromFile(dockerConfigFilePath),
				WithInsecureConnection(),
			)
			Expect(c.image).To(Equal(image))
			Expect(c.kubeconfig).To(Equal(kubeconfig))
			Expect(c.kubeconfig).To(Equal([]byte(kubeconfigContents)))
			Expect(c.indeximage).To(Equal(indeximage))
			Expect(c.scorecardImage).To(Equal(scorecardImage))
			Expect(c.scorecardNamespace).To(Equal(scorecardNamespace))
			Expect(c.scorecardServiceAccount).To(Equal(scorecardServiceAccount))
			Expect(c.scorecardWaitTime).To(Equal(scorecardWaitTime))
			Expect(c.operatorChannel).To(Equal(operatorChannel))
			Expect(c.dockerConfigFilePath).To(Equal(dockerConfigFilePath))
			Expect(c.insecure).To(Equal(insecure))
		})
	})
})

var _ = Describe("Operator Check Execution", func() {
	// NOTE: There's no unit test for running the operator check because it requires a cluster.

	When("Calling the check", func() {
		It("should fail if you passed an empty image", func() {
			chk := NewCheck("", "indeximage", []byte{})
			_, err := chk.Run(context.TODO())
			Expect(err).To(MatchError(preflighterr.ErrImageEmpty))
		})

		It("should fail if you passed an empty kubeconfig", func() {
			chk := NewCheck("image", "indeximage", nil)
			_, err := chk.Run(context.TODO())
			Expect(err).To(MatchError(preflighterr.ErrKubeconfigEmpty))
		})

		It("should fail if you passed an empty index image", func() {
			chk := NewCheck("image", "", []byte{})
			_, err := chk.Run(context.TODO())
			Expect(err).To(MatchError(preflighterr.ErrIndexImageEmpty))
		})
	})
})
