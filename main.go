package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/alecthomas/kingpin"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"

	"github.com/goreleaser/goreleaser/goreleaserlib"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func init() {
	log.SetHandler(cli.New(os.Stdout))
}

var (
	app = kingpin.New("goreleaser", "Deliver Go binaries as fast and easily as possible")

	config = app.Flag("config", "file to read the config from").
		Short('c').
		Short('f').
		String()

	releaseNotes = app.Flag("release-notes", "load custom release notes from a markdown file").
			ExistingFile()

	skipValidate = app.Flag("skip-validate", "skip all the validations against the release").
			Bool()

	skipPublish = app.Flag("skip-publish", "skip all publishing pipes of the release").
			Bool()

	snapshot = app.Flag("snapshot", "generate an unversioned snapshot release").
			Bool()

	rmDist = app.Flag("rm-dist", "remove ./dist before building").
		Bool()

	parallelism = app.Flag("parallelism", "maximum amount of tasks to launch in parallel").
			Short('p').
			Default(strconv.Itoa(runtime.NumCPU())).
			Int()

	debug = app.Flag("debug", "enable debug mode").
		Bool()

	initCmd = app.Command("init", "generate the .goreleaser.yml file").
		Alias("i")
)

func main() {
	app.Version(fmt.Sprintf("%v, commit %v, built at %v", version, commit, date))
	app.HelpFlag.Short('h')
	app.VersionFlag.Short('v')
	app.Author("Carlos Alexandro Becker <caarlos0@gmail.com>")

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {
	case initCmd.FullCommand():
		initProject()
	default:
		log.Infof("running goreleaser %v", version)
		app.FatalIfError(
			goreleaserlib.Release(goreleaserlib.Flags{
				Config:       *config,
				Debug:        *debug,
				Parallelism:  *parallelism,
				ReleaseNotes: *releaseNotes,
				RmDist:       *rmDist,
				SkipPublish:  *skipPublish,
				SkipValidate: *skipValidate,
				Snapshot:     *snapshot,
			}),
			"failed to release",
		)
	}
}

func initProject() {
	var filename = ".goreleaser.yml"
	app.FatalIfError(
		goreleaserlib.InitProject(filename),
		"failed to init project",
	)
	log.WithField("file", filename).
		Info("config created; please edit accordingly to your needs")
}
