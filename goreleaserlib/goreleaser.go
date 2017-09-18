package goreleaserlib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/goreleaser/goreleaser/pipeline/archive"
	"github.com/goreleaser/goreleaser/pipeline/brew"
	"github.com/goreleaser/goreleaser/pipeline/build"
	"github.com/goreleaser/goreleaser/pipeline/checksums"
	"github.com/goreleaser/goreleaser/pipeline/cleandist"
	"github.com/goreleaser/goreleaser/pipeline/defaults"
	"github.com/goreleaser/goreleaser/pipeline/docker"
	"github.com/goreleaser/goreleaser/pipeline/env"
	"github.com/goreleaser/goreleaser/pipeline/fpm"
	"github.com/goreleaser/goreleaser/pipeline/git"
	"github.com/goreleaser/goreleaser/pipeline/release"
	"github.com/goreleaser/goreleaser/pipeline/snapcraft"
	yaml "gopkg.in/yaml.v2"
)

var pipes = []pipeline.Pipe{
	defaults.Pipe{},  // load default configs
	git.Pipe{},       // get and validate git repo state
	env.Pipe{},       // load and validate environment variables
	cleandist.Pipe{}, // ensure ./dist is clean
	build.Pipe{},     // build
	archive.Pipe{},   // archive (tar.gz, zip, etc)
	fpm.Pipe{},       // archive via fpm (deb, rpm, etc)
	snapcraft.Pipe{}, // archive via snapcraft (snap)
	checksums.Pipe{}, // checksums of the files
	docker.Pipe{},    // create and push docker images
	release.Pipe{},   // release to github
	brew.Pipe{},      // push to brew tap
}

type Flags struct {
	Config       *string
	ReleaseNotes *string
	SkipValidate *bool
	SkipPublish  *bool
	Snapshot     *bool
	RmDist       *bool
	Parallelism  *int
	Debug        *bool
}

// Release runs the release process with the given flags
func Release(flags Flags) error {
	var file = getConfigFile(flags)
	if *flags.Debug {
		log.SetLevel(log.DebugLevel)
	}
	cfg, err := config.Load(file)
	if err != nil {
		// Allow file not found errors if config file was not
		// explicitly specified
		_, statErr := os.Stat(file)
		if !os.IsNotExist(statErr) || flags.Config != nil {
			return err
		}
		log.WithField("file", file).Warn("could not load config, using defaults")
	}
	var ctx = context.New(cfg)
	ctx.Parallelism = *flags.Parallelism
	log.Debugf("parallelism: %v", ctx.Parallelism)
	ctx.Validate = !*flags.SkipValidate
	ctx.Publish = !*flags.SkipPublish
	if *flags.ReleaseNotes != "" {
		bts, err := ioutil.ReadFile(*flags.ReleaseNotes)
		if err != nil {
			return err
		}
		log.WithField("notes", *flags.ReleaseNotes).
			Info("loaded custom release notes")
		ctx.ReleaseNotes = string(bts)
	}
	ctx.Snapshot = *flags.Snapshot
	if ctx.Snapshot {
		log.Info("publishing disabled in snapshot mode")
		ctx.Publish = false
	}
	ctx.RmDist = *flags.RmDist
	for _, pipe := range pipes {
		log.Infof("\033[1m%s\033[0m", strings.ToUpper(pipe.Description()))
		if err := handle(pipe.Run(ctx)); err != nil {
			return err
		}
	}
	log.Infof("\033[1mSUCCESS!\033[0m")
	return nil
}

func handle(err error) error {
	if err == nil {
		return nil
	}
	if pipeline.IsSkip(err) {
		log.WithField("reason", err.Error()).Warn("skipped")
		return nil
	}
	return err
}

// InitProject creates an example goreleaser.yml in the current directory
func InitProject(filename string) error {
	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		if err != nil {
			return err
		}
		return fmt.Errorf("%s already exists", filename)
	}

	var ctx = context.New(config.Project{})
	var pipe = defaults.Pipe{}
	if err := pipe.Run(ctx); err != nil {
		return err
	}
	out, err := yaml.Marshal(ctx.Config)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, out, 0644)
}

func getConfigFile(flags Flags) string {
	var config = flags.Config
	if config != nil {
		return *config
	}
	for _, f := range []string{
		".goreleaser.yml",
		".goreleaser.yaml",
		"goreleaser.yml",
		"goreleaser.yaml",
	} {
		_, ferr := os.Stat(f)
		if ferr == nil || os.IsExist(ferr) {
			return f
		}
	}
	return *config
}
