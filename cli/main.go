package main

import (
	"io/ioutil"
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/diecast"
	"github.com/ghetzel/diecast/util"
	"github.com/ghodss/yaml"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger(`main`)

func main() {
	app := cli.NewApp()
	app.Name = util.ApplicationName
	app.Usage = util.ApplicationSummary
	app.Version = util.ApplicationVersion
	app.EnableBashCompletion = false

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `bindings, B`,
			Usage: `A configuration file that holds binding configurations.`,
		},
		cli.StringFlag{
			Name:   `address, a`,
			Usage:  `Address the HTTP server should listen on`,
			Value:  diecast.DEFAULT_SERVE_ADDRESS,
			EnvVar: `HTTP_ADDR`,
		},
		cli.IntFlag{
			Name:   `port, p`,
			Usage:  `TCP port the HTTP server should listen on`,
			Value:  diecast.DEFAULT_SERVE_PORT,
			EnvVar: `HTTP_PORT`,
		},
		cli.StringFlag{
			Name:   `route-prefix`,
			Usage:  `The path prepended to all HTTP requests`,
			Value:  diecast.DEFAULT_ROUTE_PREFIX,
			EnvVar: `ROUTE_PREFIX`,
		},
		cli.StringSliceFlag{
			Name:  `template-pattern, P`,
			Usage: `A shell glob pattern matching a set of files that should be templated`,
		},
		cli.StringSliceFlag{
			Name:  `mount, m`,
			Usage: `Expose a given as MOUNT and SOURCE when requested from the server (formatted as "MOUNT:SOURCE"; e.g. "/js:/usr/share/javascript")`,
		},
	}

	app.Before = func(c *cli.Context) error {
		level := logging.DEBUG

		if lvl, err := logging.LogLevel(c.String(`log-level`)); err == nil {
			level = lvl
		}

		logging.SetFormatter(logging.MustStringFormatter(`%{color}%{level:.4s}%{color:reset}[%{id:04d}] %{module}: %{message}`))
		logging.SetLevel(level, ``)

		log.Infof("%s v%s started at %s", util.ApplicationName, util.ApplicationVersion, util.StartedAt)

		return nil
	}

	app.Action = func(c *cli.Context) {
		server := diecast.NewServer(c.Args().First())

		server.Address = c.String(`address`)
		server.Port = c.Int(`port`)
		server.RoutePrefix = c.String(`route-prefix`)

		if v := c.StringSlice(`template-pattern`); len(v) > 0 {
			server.TemplatePatterns = v
		}

		if v := c.String(`bindings`); v != `` {
			if file, err := os.Open(v); err == nil {
				if data, err := ioutil.ReadAll(file); err == nil && len(data) > 0 {
					var bindings []*diecast.Binding

					if err := yaml.Unmarshal(data, &bindings); err == nil {
						server.Bindings = bindings
					} else {
						log.Fatal(err)
					}
				} else {
					log.Fatal(err)
				}
			} else {
				log.Fatal(err)
			}
		}

		mounts := make([]diecast.Mount, 0)

		for _, mountSpec := range c.StringSlice(`mount`) {
			if mount, err := diecast.NewMountFromSpec(mountSpec); err == nil {
				mounts = append(mounts, mount)
			}
		}

		server.SetMounts(mounts)

		if err := server.Initialize(); err == nil {
			log.Infof("Starting HTTP server at http://%s:%d", server.Address, server.Port)
			server.Serve()
		} else {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}

	app.Run(os.Args)
}
