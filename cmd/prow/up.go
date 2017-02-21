package main

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rjeczalik/notify"
	"github.com/spf13/cobra"

	"github.com/deis/prow/pkg/prow"
)

const upDesc = `
This command archives the current directory into a tar archive and uploads it to the prow server.
`

type upCmd struct {
	appName      string
	client       *prow.Client
	namespace    string
	out          io.Writer
	buildTarPath string
	chartTarPath string
	wait         bool
	watch        bool
}

func newUpCmd(out io.Writer) *cobra.Command {
	up := &upCmd{
		out: out,
	}

	cmd := &cobra.Command{
		Use:     "up",
		Short:   "upload the current directory to the prow server for deployment",
		Long:    upDesc,
		PreRunE: setupConnection,
		RunE: func(cmd *cobra.Command, args []string) error {
			up.client = ensureProwClient(up.client)
			return up.run()
		},
	}

	f := cmd.Flags()
	f.StringVarP(&up.appName, "app", "a", "", "name of the helm release. By default this is the basename of the current working directory")
	f.StringVarP(&up.namespace, "namespace", "n", "default", "kubernetes namespace to install the chart")
	f.StringVar(&up.buildTarPath, "build-tar", "", "path to a gzipped build tarball. --chart-tar must also be set.")
	f.StringVar(&up.chartTarPath, "chart-tar", "", "path to a gzipped chart tarball. --build-tar must also be set.")
	f.BoolVarP(&up.wait, "wait", "w", false, "specifies whether or not to wait for all resources to be ready")
	f.BoolVarP(&up.watch, "watch", "", false, "whether to deploy the app automatically when local files change")

	return cmd
}

func (u *upCmd) run() (err error) {
	cwd, e := os.Getwd()
	if e != nil {
		return e
	}
	if u.appName == "" {
		u.appName = path.Base(cwd)
	}
	u.client.OptionWait = u.wait

	if err = u.doUp(cwd); err != nil {
		return err
	}

	// if `--watch=false`, return now
	if !u.watch {
		return nil
	}

	watchingMsg := "Watching local files for changes..."
	fmt.Println(watchingMsg)

	notifyPath := filepath.Join(cwd, "...")
	notifyTypes := []notify.Event{notify.Create, notify.Remove, notify.Rename, notify.Write}
	// make a buffered channel of filesystem notification events
	ch := make(chan notify.EventInfo, 1)

	// watch the current directory and everything under it, sending events to the channel
	if err := notify.Watch(notifyPath, ch, notifyTypes...); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(ch)

	// create a timer to enforce a "quiet period" before deploying the app
	timer := time.NewTimer(time.Hour)
	timer.Stop()
	// TODO: allow the user to set this quiet period?
	quietPeriod := time.Second * 5

	for {
		select {
		case evt := <-ch:
			log.Debugf("Event %s", evt)
			// reset the timer when files have changed
			timer.Reset(quietPeriod)
		case <-timer.C:
			if err = u.doUp(cwd); err != nil {
				return err
			}
			fmt.Println(watchingMsg)
		}
	}
	return
}

func (u *upCmd) doUp(cwd string) (err error) {
	if u.buildTarPath != "" && u.chartTarPath != "" {
		buildTar, e := os.Open(u.buildTarPath)
		if e != nil {
			return e
		}
		chartTar, e := os.Open(u.chartTarPath)
		if e != nil {
			return e
		}
		err = u.client.Up(u.appName, u.namespace, u.out, buildTar, chartTar)
	} else {
		err = u.client.UpFromDir(u.appName, u.namespace, u.out, cwd)
	}

	// format error before returning
	if err != nil {
		err = fmt.Errorf("there was an error running 'prow up': %v", err)
	}
	return
}
