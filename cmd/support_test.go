package cmd

import (
	"bytes"
	"net/url"
	"sync"
	"time"

	"github.com/manifoldco/promptui"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	keyNext  = 14
	keyEnter = 13
)

var _ = Describe("support command tests", func() {
	Context("When testing the project type selection", func() {
		var selectPrompt *promptui.Select

		BeforeEach(func() {
			selectPrompt = projectTypeSelect()
		})

		It("should have the container option", func() {
			var result string
			var err error
			// Container Image is the first option.
			timedout := runTestFuncWithTimeout(func() {
				result, err = executeSelect(selectPrompt, []byte{keyEnter})
			}, 2*time.Second)
			Expect(timedout).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(containerImage))
		})

		It("should have the operator option", func() {
			var result string
			var err error
			timedout := runTestFuncWithTimeout(func() {
				// Operator Bundle is the second option.
				result, err = executeSelect(selectPrompt, []byte{keyNext, keyEnter})
			}, 2*time.Second)
			Expect(timedout).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(operatorBundleImage))
		})

		It("should have two options", func() {
			// This is a sanity check. We only have two options at the time of
			// this writing, so if we add more, we should add tests to make sure
			// they're selectable in promptui.
			Expect(len(selectPrompt.Items.([]string))).To(Equal(2))
		})
	})

	Context("When validating a pull request URL", func() {
		urlNoScheme := "example.com"
		urlNoHost := "https:///foo"
		urlNoPath := "https://example.com"
		urlCorrect := "https://example.com/pull/example"

		It("should fail when no scheme is provided", func() {
			err := pullRequestURLValidation(urlNoScheme)
			Expect(err).To(HaveOccurred())
		})
		It("should fail when no host is provided", func() {
			err := pullRequestURLValidation(urlNoHost)
			Expect(err).To(HaveOccurred())
		})
		It("should fail when no path is provided", func() {
			err := pullRequestURLValidation(urlNoPath)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed when the url is in the proper format", func() {
			err := pullRequestURLValidation(urlCorrect)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When building out the query parameters", func() {
		It("Should properly include the base parameters, project ID, type, and pull URL", func() {
			params := queryParams("foo", "bar", "baz")
			Expect(params.Get(typeParam)).To(Equal(typeValue))
			Expect(params.Get(sourceParam)).To(Equal(sourceValue))
			Expect(params.Get(certProjectTypeParam)).To(Equal("foo"))
			Expect(params.Get(certProjectIDParam)).To(Equal("bar"))
			Expect(params.Get(pullRequestURLParam)).To(Equal("baz"))
		})

		It("Should not include the pull request URL if it's passed in as an empty value.", func() {
			params := queryParams("foo", "bar", "")
			Expect(params.Get(pullRequestURLParam)).To(Equal(""))
		})
	})

	Context("When rendering support instructions", func() {
		It("Should include the provided base URL and parameters", func() {
			params := url.Values{}
			params.Add("foo", "bar")
			baseURL := "https://example.com"
			instructions := supportInstructions(baseURL, params)
			Expect(instructions).To(ContainSubstring(baseURL))
			Expect(instructions).To(ContainSubstring("foo=bar"))
		})
	})

	Context("when prompting the user for a pull request URL", func() {
		It("should return with no error when the URL is in the proper format", func() {
			url := "https://example.com/pull/1"
			var returned string
			var err error
			timedout := runTestFuncWithTimeout(func() {
				returned, err = executePrompt(pullRequestURLPrompt(), []byte(url))
			}, 2*time.Second)
			Expect(timedout).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
			Expect(returned).To(Equal(url))
		})
	})

	Context("When validating a project ID", func() {
		projectIDEmpty := ""
		projectIDStartWithP := "p1"
		projectIDStartWithOSPID := "ospid-0000"
		projectIDContainSpecialChar := "$c$"
		projectIDCorrect := "000011112222"

		It("should fail if the input is empty", func() {
			err := projectIDValidation(projectIDEmpty)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if if the input contains a leading p", func() {
			err := projectIDValidation(projectIDStartWithP)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the input begins with ospid-", func() {
			err := projectIDValidation(projectIDStartWithOSPID)
			Expect(err).To(HaveOccurred())
		})

		It("should fail if the input has special characters", func() {
			err := projectIDValidation(projectIDContainSpecialChar)
			Expect(err).To(HaveOccurred())
		})

		It("should succeed if the format is correct", func() {
			err := projectIDValidation(projectIDCorrect)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("When prompting the user for their project ID", func() {
		It("should return with no error when using a valid project id format", func() {
			pid := "000000000000"

			var returned string
			var err error
			timedout := runTestFuncWithTimeout(func() {
				returned, err = executePrompt(projectIDPrompt(), []byte(pid))
			}, 2*time.Second)
			Expect(timedout).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
			Expect(returned).To(Equal(pid))
		})
	})
})

// executePrompt runs a promptui.Prompt and passes it input. The input is automatically
// appended with the Enter command. The prompt's output, or an error is returned.
// This should be executed with a timeout wrapper.
func executePrompt(p *promptui.Prompt, input []byte) (string, error) {
	promptStdin := promptBuffer{bytes.NewBuffer([]byte{})}
	promptStdout := promptBuffer{bytes.NewBuffer([]byte{})}

	p.Stdin = &promptStdin
	p.Stdout = &promptStdout
	var wg sync.WaitGroup
	var result string
	var err error

	wg.Add(1)
	go func() {
		result, err = p.Run()
		wg.Done()
	}()

	// send the input to the prompt
	inputwithEnter := bytes.Join([][]byte{input, {keyEnter}}, []byte{})
	promptStdin.Write(inputwithEnter)
	wg.Wait()

	return result, err
}

// executeSelect runs a promptui.Select and passes it input. The input is automatically
// appended with the Enter command. The prompt's output, or an error is returned.
// This should be executed with a timeout wrapper.
func executeSelect(s *promptui.Select, input []byte) (string, error) {
	promptStdin := promptBuffer{bytes.NewBuffer([]byte{})}
	promptStdout := promptBuffer{bytes.NewBuffer([]byte{})}

	s.Stdin = &promptStdin
	s.Stdout = &promptStdout
	var wg sync.WaitGroup
	var result string
	var err error

	wg.Add(1)
	go func() {
		_, result, err = s.Run()
		wg.Done()
	}()

	// send the input to the prompt
	inputwithEnter := bytes.Join([][]byte{input, {keyEnter}}, []byte{})
	promptStdin.Write(inputwithEnter)
	wg.Wait()

	return result, err
}

type promptBuffer struct {
	*bytes.Buffer
}

func (b *promptBuffer) Close() error {
	return nil
}
