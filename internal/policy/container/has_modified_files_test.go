package container

import (
	"bytes"
	"context"
	"path"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/image"

	"github.com/bombsimon/logrusr/v4"
	"github.com/go-logr/logr"
	rpmdb "github.com/knqyf263/go-rpmdb/pkg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("HasModifiedFiles", func() {
	var (
		hasModifiedFiles HasModifiedFilesCheck
		layers           []string
		pkgRef           map[string]packageFilesRef
		dist             string
	)

	BeforeEach(func() {
		pkgRef = make(map[string]packageFilesRef)
		pkgRef["firstlayer"] = packageFilesRef{
			LayerFiles: []string{
				"this",
				"is",
				"not",
				"prohibited",
			},
			LayerPackages: map[string]packageMeta{
				"foo-1.0-1.d9": {
					Name:    "foo",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"bar-1.0-1.d9": {
					Name:    "bar",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"baz-2.0-1.d9": {
					Name:    "baz",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
			},
			LayerPackageFiles: map[string]string{
				"this": "foo-1.0-1.d9",
				"is":   "bar-1.0-1.d9",
				"not":  "baz-2.0-1.d9",
			},
			HasRPMDB: true,
		}

		pkgRef["secondlayer"] = packageFilesRef{
			LayerFiles: []string{
				"there",
				"are",
				"no",
				"prohibited",
				"duplicates",
			},
			LayerPackages: map[string]packageMeta{
				"foo-1.0-1.d9": {
					Name:    "foo",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"bar-1.0-1.d9": {
					Name:    "bar",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"baz-2.0-1.d9": {
					Name:    "baz",
					Version: "2.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"boz-3.0-1.d9": {
					Name:    "boz",
					Version: "3.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
			},
			LayerPackageFiles: map[string]string{
				"this":  "foo-1.0-1.d9",
				"is":    "bar-1.0-1.d9",
				"not":   "baz-2.0-1.d9",
				"no":    "boz-3.0-1.d9",
				"there": "boz-3.0-1.d9",
			},
			HasRPMDB: true,
		}
		pkgRef["lastlayer"] = packageFilesRef{
			LayerFiles: []string{
				"prohibited",
			},
			LayerPackages: map[string]packageMeta{
				"foo-1.0-1.d9": {
					Name:    "foo",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"bar-1.0-1.d9": {
					Name:    "bar",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"baz-2.0-1.d9": {
					Name:    "baz",
					Version: "2.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
				"boz-3.0-1.d9": {
					Name:    "boz",
					Version: "3.0",
					Release: "1.d9",
					Arch:    "fooarch",
					Vendor:  "Red Hat, Inc.",
				},
			},
			LayerPackageFiles: map[string]string{
				"this": "foo-1.0-1.d9",
				"is":   "bar-1.0-1.d9",
				"not":  "baz-2.0-1.d9",
			},
		}
		layers = []string{
			"firstlayer",
			"secondlayer",
			"lastlayer",
		}
		dist = "d9"
	})

	Context("Checking if it has any modified RPM files", func() {
		When("there are no modified RPM files found", func() {
			It("should pass validate", func() {
				ok, err := hasModifiedFiles.validate(context.Background(), layers, pkgRef, dist)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		When("there is a modified RPM file found", func() {
			var pkgs map[string]packageFilesRef
			BeforeEach(func() {
				pkgs = pkgRef
				pkgSecondLayer := pkgRef["secondlayer"]
				pkgSecondLayer.LayerFiles = append(pkgs["secondlayer"].LayerFiles, "this")
				pkgs["secondlayer"] = pkgSecondLayer
			})
			It("should not pass Validate", func() {
				ok, err := hasModifiedFiles.validate(context.Background(), layers, pkgs, dist)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		When("a package is updated", func() {
			var pkgs map[string]packageFilesRef
			BeforeEach(func() {
				pkgs = pkgRef
				pkgSecondLayer := pkgs["secondlayer"]
				pkgSecondLayerPackageFiles := pkgSecondLayer.LayerPackageFiles
				pkgSecondLayerPackageFiles["this"] = "foo-2.0-d9"
				pkgSecondLayer.LayerPackageFiles = pkgSecondLayerPackageFiles
				pkgs["secondlayer"] = pkgSecondLayer
			})
			It("should pass validate", func() {
				ok, err := hasModifiedFiles.validate(context.Background(), layers, pkgs, dist)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		When("a package is removed", func() {
			var pkgs map[string]packageFilesRef
			BeforeEach(func() {
				pkgs = pkgRef
				pkgSecondLayer := pkgs["secondlayer"]
				pkgSecondLayerPackageFiles := pkgSecondLayer.LayerPackageFiles
				delete(pkgSecondLayerPackageFiles, "this")
				pkgSecondLayer.LayerPackageFiles = pkgSecondLayerPackageFiles
				pkgs["secondlayer"] = pkgSecondLayer
			})
			It("should pass validate", func() {
				ok, err := hasModifiedFiles.validate(context.Background(), layers, pkgs, dist)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeTrue())
			})
		})
		When("the package release dist changes", func() {
			var pkgs map[string]packageFilesRef
			BeforeEach(func() {
				pkgs = pkgRef

				pkgSecondLayerPackageFiles := pkgs["secondlayer"].LayerPackageFiles
				delete(pkgSecondLayerPackageFiles, "this")
				pkgSecondLayerPackageFiles["this"] = "foo-1.0-1.d10"

				pkgSecondLayerPackages := pkgs["secondlayer"].LayerPackages
				delete(pkgSecondLayerPackages, "foo-1.0-1.d9")
				pkgSecondLayerPackages["foo-1.0-1.d10"] = packageMeta{
					Name:    "foo",
					Version: "1.0",
					Release: "1.d10",
					Arch:    "fooarch",
				}

				pkgs["secondlayer"] = packageFilesRef{
					LayerPackages:     pkgSecondLayerPackages,
					LayerPackageFiles: pkgSecondLayerPackageFiles,
					LayerFiles:        append(pkgs["secondlayer"].LayerFiles, "this"),
					HasRPMDB:          true,
				}
			})
			It("should fail because of different release dist", func() {
				ok, err := hasModifiedFiles.validate(context.Background(), layers, pkgs, dist)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		When("the package architecture changes", func() {
			var pkgs map[string]packageFilesRef
			BeforeEach(func() {
				pkgs = pkgRef

				pkgSecondLayerPackageFiles := pkgs["secondlayer"].LayerPackageFiles
				delete(pkgSecondLayerPackageFiles, "this")
				pkgSecondLayerPackageFiles["this"] = "foo-1.0-1.d10"

				pkgSecondLayerPackages := pkgs["secondlayer"].LayerPackages
				delete(pkgSecondLayerPackages, "foo-1.0-1.d9")
				pkgSecondLayerPackages["foo-1.0-1.d10"] = packageMeta{
					Name:    "foo",
					Version: "1.0",
					Release: "1.d9",
					Arch:    "differentarch",
				}

				pkgs["secondlayer"] = packageFilesRef{
					LayerPackages:     pkgSecondLayerPackages,
					LayerPackageFiles: pkgSecondLayerPackageFiles,
					LayerFiles:        append(pkgs["secondlayer"].LayerFiles, "this"),
					HasRPMDB:          true,
				}
			})
			It("should fail because of different architectures dist", func() {
				ok, err := hasModifiedFiles.validate(context.Background(), layers, pkgs, dist)
				Expect(err).ToNot(HaveOccurred())
				Expect(ok).To(BeFalse())
			})
		})
		When("release dist does not match installed OS", func() {
			When("package is a net-new", func() {
				When("a file is modified", func() {
					var pkgs map[string]packageFilesRef
					var ctx context.Context
					var logOutput bytes.Buffer
					BeforeEach(func() {
						pkgs = pkgRef

						pkgSecondLayerPackages := pkgs["secondlayer"].LayerPackages
						pkgs["secondlayer"].LayerPackages["other-1.0-1.oth"] = packageMeta{
							Name:    "other",
							Version: "1.0",
							Release: "1.oth",
							Arch:    "x86_64",
						}
						pkgSecondLayerPackageFiles := pkgs["secondlayer"].LayerPackageFiles
						pkgSecondLayerPackageFiles["otherfile"] = "other-1.0-1.oth"
						pkgs["secondlayer"] = packageFilesRef{
							LayerPackages:     pkgSecondLayerPackages,
							LayerPackageFiles: pkgSecondLayerPackageFiles,
							LayerFiles:        append(pkgs["secondlayer"].LayerFiles, "otherfile"),
							HasRPMDB:          true,
						}

						pkgs["lastlayer"] = packageFilesRef{
							LayerPackages:     pkgs["secondlayer"].LayerPackages,
							LayerPackageFiles: pkgs["secondlayer"].LayerPackageFiles,
							LayerFiles:        append(pkgs["lastlayer"].LayerFiles, "otherfile"),
							HasRPMDB:          false,
						}

						l := logrus.New()
						l.SetLevel(logrus.DebugLevel)
						l.SetOutput(&logOutput)
						logger := logrusr.New(l)
						ctx = logr.NewContext(context.Background(), logger)
					})
					It("should warn but not fail", func() {
						ok, err := hasModifiedFiles.validate(ctx, layers, pkgs, dist)
						Expect(err).ToNot(HaveOccurred())
						Expect(ok).To(BeTrue())
						Expect(logOutput.String()).To(ContainSubstring("WARN"))
					})
				})
			})
		})
	})

	When("the first layer is empty", func() {
		var zeroPkgRef map[string]packageFilesRef
		var zeroLayers []string
		BeforeEach(func() {
			zeroPkgRef = pkgRef
			zeroPkgRef["zerolayer"] = packageFilesRef{
				LayerFiles:        []string{},
				LayerPackages:     make(map[string]packageMeta),
				LayerPackageFiles: make(map[string]string),
				HasRPMDB:          false,
			}
			zeroLayers = append([]string{"zerolayer"}, layers...)
		})
		It("should ignore it", func() {
			ok, err := hasModifiedFiles.validate(context.Background(), zeroLayers, zeroPkgRef, dist)
			Expect(err).ToNot(HaveOccurred())
			Expect(ok).To(BeTrue())
		})
	})

	When("building the installed file list for installed packages", func() {
		const (
			basename = "foobasename"
			dirname  = "foodirname"
			dirindex = 0
		)
		var goodPkgList []*rpmdb.PackageInfo

		BeforeEach(func() {
			// goodPkgList represents three mock RPM's,
			// the first is a basic one to cover the happy path.
			// the second is need to test our filtering logic for files from another architecture.
			// the third is to test our filtering logic for directories.
			goodPkgList = []*rpmdb.PackageInfo{
				{
					BaseNames:  []string{basename},
					DirIndexes: []int32{dirindex},
					DirNames:   []string{dirname},
					Name:       "foo",
					Version:    "1.0.0",
					Release:    "100",
					Arch:       "x86_64",
				},
				{
					BaseNames:  []string{basename},
					DirIndexes: []int32{dirindex},
					DirNames:   []string{dirname},
					Name:       "foo",
					Version:    "1.0.0",
					Release:    "100",
					Arch:       "i686",
				},
				{
					BaseNames:  []string{""},
					DirIndexes: []int32{dirindex},
					DirNames:   []string{"/"},
					Name:       "just-dirs",
					Version:    "1.0.0",
					Release:    "100",
					Arch:       "x86_64",
				},
			}
		})
		It("should contain all files installed by the package according to its metadata", func() {
			files, err := installedFileMapWithExclusions(context.TODO(), goodPkgList)
			Expect(err).ToNot(HaveOccurred())

			_, ok := files[path.Join(dirname, basename)]
			Expect(ok).To(BeTrue())
		})

		It("should fail if the rpm is invalid", func() {
			badPkgList := goodPkgList
			badPkgList[0].DirNames = []string{dirname, "extradir"}
			_, err := installedFileMapWithExclusions(context.TODO(), badPkgList)
			Expect(err).To(HaveOccurred())
		})
	})

	When("calling the top level Validate", func() {
		It("should fail with an invalid ImageReference", func() {
			passed, err := hasModifiedFiles.Validate(context.TODO(), image.ImageReference{})
			Expect(err).To(HaveOccurred())
			Expect(passed).To(BeFalse())
		})
	})

	AssertMetaData(&hasModifiedFiles)
})
