/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"k8s.io/helm/cmd/helm/require"
	"k8s.io/helm/pkg/action"
	"k8s.io/helm/pkg/chart/loader"
	"k8s.io/helm/pkg/storage/driver"
)

const upgradeDesc = `
This command upgrades a release to a new version of a chart.

The upgrade arguments must be a release and chart. The chart
argument can be either: a chart reference('stable/mariadb'), a path to a chart directory,
a packaged chart, or a fully qualified URL. For chart references, the latest
version will be specified unless the '--version' flag is set.

To override values in a chart, use either the '--values' flag and pass in a file
or use the '--set' flag and pass configuration from the command line, to force string
values, use '--set-string'.

You can specify the '--values'/'-f' flag multiple times. The priority will be given to the
last (right-most) file specified. For example, if both myvalues.yaml and override.yaml
contained a key called 'Test', the value set in override.yaml would take precedence:

	$ helm upgrade -f myvalues.yaml -f override.yaml redis ./redis

You can specify the '--set' flag multiple times. The priority will be given to the
last (right-most) set specified. For example, if both 'bar' and 'newbar' values are
set for a key called 'foo', the 'newbar' value would take precedence:

	$ helm upgrade --set foo=bar --set foo=newbar redis ./redis
`

func newUpgradeCmd(cfg *action.Configuration, out io.Writer) *cobra.Command {
	client := action.NewUpgrade(cfg)

	cmd := &cobra.Command{
		Use:   "upgrade [RELEASE] [CHART]",
		Short: "upgrade a release",
		Long:  upgradeDesc,
		Args:  require.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client.Namespace = getNamespace()

			if client.Version == "" && client.Devel {
				debug("setting version to >0.0.0-0")
				client.Version = ">0.0.0-0"
			}

			if err := client.ValueOptions.MergeValues(settings); err != nil {
				return err
			}

			chartPath, err := client.ChartPathOptions.LocateChart(args[1], settings)
			if err != nil {
				return err
			}

			if client.Install {
				// If a release does not exist, install it. If another error occurs during
				// the check, ignore the error and continue with the upgrade.
				histClient := action.NewHistory(cfg)
				histClient.Max = 1
				if _, err := histClient.Run(args[0]); err == driver.ErrReleaseNotFound {
					fmt.Fprintf(out, "Release %q does not exist. Installing it now.\n", args[0])
					instClient := action.NewInstall(cfg)
					instClient.ChartPathOptions = client.ChartPathOptions
					instClient.ValueOptions = client.ValueOptions
					instClient.DryRun = client.DryRun
					instClient.DisableHooks = client.DisableHooks
					instClient.Timeout = client.Timeout
					instClient.Wait = client.Wait
					instClient.Devel = client.Devel
					instClient.Namespace = client.Namespace

					_, err := runInstall(args, instClient, out)
					return err
				}
			}

			// Check chart dependencies to make sure all are present in /charts
			ch, err := loader.Load(chartPath)
			if err != nil {
				return err
			}
			if req := ch.Metadata.Dependencies; req != nil {
				if err := action.CheckDependencies(ch, req); err != nil {
					return err
				}
			}

			resp, err := client.Run(args[0], ch)
			if err != nil {
				return errors.Wrap(err, "UPGRADE FAILED")
			}

			if settings.Debug {
				action.PrintRelease(out, resp)
			}

			fmt.Fprintf(out, "Release %q has been upgraded. Happy Helming!\n", args[0])

			// Print the status like status command does
			statusClient := action.NewStatus(cfg)
			rel, err := statusClient.Run(args[0])
			if err != nil {
				return err
			}
			action.PrintRelease(out, rel)

			return nil
		},
	}

	client.AddFlags(cmd.Flags())

	return cmd
}
