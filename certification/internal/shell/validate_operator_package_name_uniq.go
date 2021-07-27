package shell

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"errors"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification"
	log "github.com/sirupsen/logrus"
)

var pyxisBaseUrl = "https://catalog.redhat.com/api/containers/v1/operators/packages"

type packageBody struct {
	PyxisPackages []PyxisPackages `json:"data"`
	Page          int             `json:"page"`
	Page_size     int             `json:"page_size"`
	Total         int             `json:"total"`
}

type PyxisPackages struct {
	Id               string `json:"_id"`
	Association      string `json:"association"`
	Create_date      string `json:"creation_date"`
	Last_update_date string `json:"last_update_date"`
	Package_name     string `json:"package_name"`
	Source           string `json:"source"`
}

type ValidateOperatorPkNameUniqCheck struct {
}

func (p *ValidateOperatorPkNameUniqCheck) Validate(bundle string) (bool, error) {
	//log.SetLevel(log.DebugLevel)
	var bundlePackageName []string
	var packageBody packageBody
	var err error

	packageName, err := p.getPackageName(bundle)
	if err != nil {
		log.Debug("Error found in getPackaName %s", err)
		return false, err
	}
	//packageName = "argocd-operator"
	log.Debugf("packagename %s", packageName)

	req, _ := http.NewRequest("GET", pyxisBaseUrl, nil)
	queryString := req.URL.Query()
	queryString.Add("filter", fmt.Sprintf("package_name==%s", packageName))
	req.URL.RawQuery = queryString.Encode()
	req.Header.Set("X-API-KEY", "RedHatChartVerifier")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Debug("Error while conneccting to Pyxis ", err)
		return false, err
	} else {
		if resp.StatusCode == 200 {
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			s := string(body)
			log.Debugf("pyxis output : " + s)

			json.Unmarshal(body, &packageBody)
			if packageBody.Total == 0 {
				log.Debug("Pass. Packagename not found in RedHat catalog registry.")
				return true, nil
			} else if len(packageBody.PyxisPackages) > 0 {
				for _, repo := range packageBody.PyxisPackages {
					bundlePackageName = append(bundlePackageName, repo.Package_name)
					log.Debugf("Check Failed. PackageName found exist in RedHat catalog registry, %s", bundlePackageName)
					log.Info("bundlePackageName :", bundlePackageName)
					return false, nil
				}
			} else {
				log.Debugf("Something went wrong with Unmarshalling: %s", s)
				err = errors.New(fmt.Sprintf("Something went wrong with Unmarshalling: %s", s))
			}
		} else {
			log.Debugf("Bad response code from Pyxis: %d : %s", resp.StatusCode, req.URL)
			err = errors.New(fmt.Sprintf("Bad response code %d from pyxis request : %s", resp.StatusCode, req.URL))
		}
	}
	return false, err
}

func (p *ValidateOperatorPkNameUniqCheck) getPackageName(bundle string) (string, error) {
	//log.SetLevel(log.DebugLevel)
	log.Debug("in getPackageName function")
	pattern := "bundle-*"
	var re = regexp.MustCompile("operators.operatorframework.io.bundle.package.*")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Println(err)
		return "Dir not found", err
	}

	//use the first one found, if bundle-* > 1
	match := matches[0]
	log.Debugln(match)
	filepath := "/metadata/annotations.yaml"
	annotation := fmt.Sprintf("%s%s", match, filepath)
	log.Debug(annotation)
	fileContents, err := ioutil.ReadFile(annotation)
	if err != nil {
		log.Error("fail to read annotation.yaml")
		return "Unable to read annotation.yaml", err
	}

	fileContentsToString := string(fileContents)
	foundMatch := re.FindString(fileContentsToString)
	foundPackageName := strings.Split(foundMatch, ":")
	packageName := strings.ReplaceAll(foundPackageName[1], " ", "")
	log.Debugf("\n%s\n", packageName)

	return packageName, nil
}

func (p *ValidateOperatorPkNameUniqCheck) Name() string {
	return "ValidateOperatorPackageNameUniqueness"
}

func (p *ValidateOperatorPkNameUniqCheck) Metadata() certification.Metadata {
	return certification.Metadata{
		Description:      "Validating Bundle image package name uniqueness",
		Level:            "best",
		KnowledgeBaseURL: "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
		CheckURL:         "https://connect.redhat.com/zones/containers/container-certification-policy-guide",
	}
}

func (p *ValidateOperatorPkNameUniqCheck) Help() certification.HelpText {
	return certification.HelpText{
		Message:    "Check ValidateOperatorpkNameUniqCheck encountered an error. It is possible that the bundle package name already exist in the RedHat Catalog registry.",
		Suggestion: "Bundle package name must be unique meaning that it doesn't already exist in the RedHat Catalog registry",
	}
}
