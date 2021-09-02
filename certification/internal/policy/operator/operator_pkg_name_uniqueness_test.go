package operator

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// fakeAPIClient implements the necessary methods to test API query logic for this check, which includes
// implementing the local apiClient interface as well as the io.ReadCloser, used for http.Response.Body.
type fakeAPIClient struct {
	response http.Response
}

func (c *fakeAPIClient) Do(req *http.Request) (*http.Response, error) {
	return &c.response, nil
}

var _ = Describe("OperatorPkgNameIsUniqueCheck", func() {
	check := OperatorPkgNameIsUniqueCheck{}

	Describe("While ensuring that an operator's package is unique", func() {
		// tests: getPackageNAme
		Context("with the annotations map", func() {
			key := "operators.operatorframework.io.bundle.package.v1"
			pkg := "my-custom-operator"

			Context("that has the expected package key", func() {
				goodMap := map[string]string{key: pkg}

				It("should return the package name and no error", func() {
					pkgName, err := check.getPackageName(goodMap)
					Expect(err).ToNot(HaveOccurred())
					Expect(pkgName).To(Equal(pkg))
				})
			})

			Context("that does not have the expected key", func() {
				mapWithoutKey := map[string]string{}

				It("should return an error", func() {
					_, err := check.getPackageName(mapWithoutKey)
					Expect(err).To(HaveOccurred())
				})
			})

			Context("that is nil", func() {
				var nilMap map[string]string

				It("should return an error", func() {
					_, err := check.getPackageName(nilMap)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		// tests: buildRequest
		Context("and building the HTTP request parameters", func() {
			url := "https://example.com"
			packageName := "my-custom-package"

			It("should accurately reflect the input url and package name in the request data", func() {
				request, err := check.buildRequest(url, packageName)
				Expect(fmt.Sprintf("%s://%s", request.URL.Scheme, request.URL.Host)).To(Equal(url))
				Expect(request.URL.RawQuery).To(ContainSubstring(packageName))
				Expect(err).ToNot(HaveOccurred())
			})
		})

		// tests: queryAPI, parseAPIResponse, validate
		Context("and making requests to the package uniqueness API", func() {
			fakeRequest := http.Request{
				URL: &url.URL{},
			}

			Context("and receiving a response that no package exists with the defined name", func() {
				fakeRequest := http.Request{
					URL: &url.URL{},
				}
				successBody := `{"data":[],"page":0,"page_size":100,"total":0}`

				goodClient := fakeAPIClient{
					response: http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(successBody)),
					},
				}

				It("should not throw an error when the api responses with a 200 ok", func() {
					goodResp, goodErr := check.queryAPI(&goodClient, &fakeRequest)
					Expect(goodErr).ToNot(HaveOccurred())

					data, err := check.parseAPIResponse(goodResp)
					Expect(err).ToNot(HaveOccurred())

					isValid, err := check.validate(data)
					Expect(err).ToNot(HaveOccurred())
					Expect(isValid).To(BeTrue())
				})
			})

			Context("and receiving a response that no a exists with the defined name", func() {
				failBody := `{"data":[{"_id":"someID","association":"some association","creation_date":"2021-08-02","last_update_date":"2021-08-02","package_name":"some-package","source":"some source"}],"page":0,"page_size":100,"total":1}`

				failClient := fakeAPIClient{
					response: http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(failBody)),
					},
				}

				It("should not throw an error when the api responses with a 200 ok, but should not validate", func() {
					failResp, failErr := check.queryAPI(&failClient, &fakeRequest)
					Expect(failErr).ToNot(HaveOccurred())

					data, err := check.parseAPIResponse(failResp)
					Expect(err).ToNot(HaveOccurred())

					isValid, err := check.validate(data)
					Expect(err).ToNot(HaveOccurred())
					Expect(isValid).To(BeFalse())
				})
			})

			Context("and receiving a non-200 response", func() {
				errClient := fakeAPIClient{
					response: http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(strings.NewReader("")),
					},
				}

				It("should throw an error", func() {
					_, err := check.queryAPI(&errClient, &fakeRequest)
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
