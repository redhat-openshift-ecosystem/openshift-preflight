package cmd

import (
	"fmt"
	"path/filepath"
	rt "runtime"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/artifacts"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/container"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/cli"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/formatters"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/lib"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/runtime"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/internal/viper"
	"github.com/redhat-openshift-ecosystem/openshift-preflight/version"

	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
)

func checkContainerKonfluxCmd(runpreflight runPreflight) *cobra.Command {
	checkContainerKonfluxCmd := &cobra.Command{
		Use:   "konflux",
		Short: "Run checks for a container",
		Long:  `This command will run the Certification checks for a container image. `,
		Args:  checkContainerKonfluxPositionalArgs,
		// this fmt.Sprintf is in place to keep spacing consistent with cobras two spaces that's used in: Usage, Flags, etc
		Example: fmt.Sprintf("  %s", "preflight check konflux quay.io/repo-name/container-name:version"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkContainerKonfluxRunE(cmd, args, runpreflight)
		},
	}

	flags := checkContainerKonfluxCmd.Flags()

	viper := viper.Instance()
	flags.String("platform", rt.GOARCH, "Architecture of image to pull. Defaults to runtime platform.")
	_ = viper.BindPFlag("platform", flags.Lookup("platform"))

	return checkContainerKonfluxCmd
}

func checkContainerKonfluxPositionalArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("a container image positional argument is required")
	}

	return nil
}

func checkContainerKonfluxRunE(cmd *cobra.Command, args []string, runpreflight runPreflight) error {
	ctx := cmd.Context()
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("invalid logging configuration")
	}
	logger.Info("certification library version", "version", version.Version.String())

	containerImage := args[0]

	// Render the Viper configuration as a runtime.Config
	cfg, err := runtime.NewConfigFrom(*viper.Instance())
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	cfg.Image = containerImage

	containerImagePlatforms, err := platformsToBeProcessed(cmd, cfg)
	if err != nil {
		return err
	}

	for _, platform := range containerImagePlatforms {
		logger.Info(fmt.Sprintf("running checks for %s for platform %s", containerImage, platform))
		artifactsWriter, err := artifacts.NewFilesystemWriter(artifacts.WithDirectory(filepath.Join(cfg.Artifacts, platform)))
		if err != nil {
			return err
		}

		// Add the artifact writer to the context for use by checks.
		ctx := artifacts.ContextWithWriter(ctx, artifactsWriter)

		formatter, err := formatters.NewByName(formatters.DefaultFormat)
		if err != nil {
			return err
		}

		opts := generateContainerKonfluxCheckOptions(cfg)
		opts = append(opts, container.WithPlatform(platform))

		checkcontainer := container.NewCheck(
			containerImage,
			opts...,
		)

		// Run the  container check.
		cmd.SilenceUsage = true

		if err := runpreflight(
			ctx,
			checkcontainer.Run,
			cli.CheckConfig{
				IncludeJUnitResults: cfg.WriteJUnit,
				SubmitResults:       cfg.Submit,
			},
			formatter,
			&runtime.ResultWriterFile{},
			&lib.NoopSubmitter{},
		); err != nil {
			return err
		}
	}

	return nil
}

func generateContainerKonfluxCheckOptions(cfg *runtime.Config) []container.Option {
	o := []container.Option{
		container.WithDockerConfigJSONFromFile(cfg.DockerConfig),
		container.WithManifestListDigest(cfg.ManifestListDigest),
		container.WithKonflux(),
	}

	return o
}
