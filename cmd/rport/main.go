package main

import (
	"errors"
	"fmt"
	"github.com/KonradKuznicki/must"
	"github.com/cloudradar-monitoring/rport/cmd/rport/cliboilerplate"
	"github.com/cloudradar-monitoring/rport/share/files"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"runtime"

	chclient "github.com/cloudradar-monitoring/rport/client"
	chshare "github.com/cloudradar-monitoring/rport/share"
	"github.com/cloudradar-monitoring/rport/share/clientconfig"
)

var (
	RootCmd *cobra.Command
)

func init() {
	// Assign root cmd late to avoid initialization loop
	RootCmd = &cobra.Command{
		Version: chshare.BuildVersion,
		Run:     runMain,
	}

	// set help message
	RootCmd.SetUsageFunc(func(*cobra.Command) error {
		fmt.Print(cliboilerplate.ClientHelp)
		os.Exit(1)
		return nil
	})

	pFlags := RootCmd.PersistentFlags()

	cliboilerplate.SetPFlags(pFlags)
}

// main this binary can be run in 3 ways
// 1 - interactive - for development or other advanced use
// 2 - as an OS service
// 3 - as interface for managing OS service (start, stop, install, uninstall, etc) (install needs to check config and create dirs)
func main() {
	must.MustF(RootCmd.Execute(), "failed executing RootCmd: %v")
}

func runMain(*cobra.Command, []string) {
	if isServiceManager() {
		manageService() // app run to change state of OS service
	} else {
		runClient() // app run as rport client
	}
}

func isServiceManager() bool {
	pFlags := RootCmd.PersistentFlags()
	svcCommand := must.Return(pFlags.GetString("service"))
	return svcCommand != ""
}

func decodeConfig(cfgPath string, overrideConfigWithCLIArgs bool) (*chclient.ClientConfigHolder, error) {

	viperCfg := viper.New()
	viperCfg.SetConfigType("toml")

	cliboilerplate.SetViperConfigDefaults(viperCfg)

	if cfgPath != "" {
		viperCfg.SetConfigFile(cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rport.conf")
	}

	config := &chclient.ClientConfigHolder{Config: &clientconfig.Config{}}

	pFlags := RootCmd.PersistentFlags()

	if overrideConfigWithCLIArgs {
		cliboilerplate.BindPFlagsToViperConfig(pFlags, viperCfg)
	}

	if err := chshare.DecodeViperConfig(viperCfg, config.Config, nil); err != nil {
		return nil, err
	}

	if config.InterpreterAliases == nil {
		config.InterpreterAliases = map[string]string{}
	}

	if overrideConfigWithCLIArgs {
		args := pFlags.Args()

		if len(args) > 0 {
			config.Client.Server = args[0]
			config.Client.Remotes = args[1:]
		}

		config.Tunnels.Scheme = must.Return(pFlags.GetString("scheme"))
		config.Tunnels.ReverseProxy = must.Return(pFlags.GetBool("enable-reverse-proxy"))
		config.Tunnels.HostHeader = must.Return(pFlags.GetString("host-header"))
	}

	return config, nil
}

func runClient() {
	pFlags := RootCmd.PersistentFlags()

	cfgPath := must.Return(pFlags.GetString("config"))

	config := must.ReturnF(decodeConfig(cfgPath, service.Interactive()))("Invalid config: %v. Check your config file.")

	must.MustF(config.Logging.LogOutput.Start(), "failed starting log output: %v")
	defer config.Logging.LogOutput.Shutdown()

	must.MustF(chclient.PrepareDirs(config), "failed preparing directories: %v")

	must.MustF(config.ParseAndValidate(false), "config validation failed: %v")

	must.MustF(checkRootOK(config), "root check failed: %v")

	fileAPI := files.NewFileSystem()
	c := must.ReturnF(chclient.NewClient(config, fileAPI))("failed creating client: %v")

	if service.Interactive() { // if run from command line

		go chshare.GoStats()

		must.MustF(c.Run(), "failed to run client: %v")

	} else { // if run as OS service

		must.MustF(runAsService(c, cfgPath), "failed to start service: %v")

	}

}

func checkRootOK(config *chclient.ClientConfigHolder) error {
	if !config.Client.AllowRoot && chshare.IsRunningAsRoot() {
		return errors.New("by default running as root is not allowed")
	}
	return nil
}

func manageService() {
	var svcUser string
	pFlags := RootCmd.PersistentFlags()
	cfgPath := must.Return(pFlags.GetString("config"))
	svcCommand := must.Return(pFlags.GetString("service"))

	if runtime.GOOS != "windows" {
		svcUser = must.Return(pFlags.GetString("service-user"))
	}

	if svcCommand == "install" {
		// validate config file without command line args before installing it for the service
		// other service commands do not change config file specified at install

		config := must.ReturnF(decodeConfig(cfgPath, false))("Invalid config: %v. Check your config file.")

		must.MustF(config.ParseAndValidate(true), "config validation failed: %v")

	}

	must.Must(handleSvcCommand(svcCommand, cfgPath, svcUser))
}
