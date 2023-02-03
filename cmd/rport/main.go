package main

import (
	"errors"
	"fmt"
	"github.com/KonradKuznicki/must"
	"github.com/cloudradar-monitoring/rport/cmd/rport/cli_boilerplate"
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
		fmt.Print(cli_boilerplate.ClientHelp)
		os.Exit(1)
		return nil
	})

	pFlags := RootCmd.PersistentFlags()

	cli_boilerplate.SetPFlags(pFlags)
}

// main this binary can be run in 3 ways
// 1 - interactive - for development or other advanced use
// 2 - as an OS service
// 3 - as interface for managing OS service (start, stop, install, uninstall, etc) (install needs to check config and create dirs)
func main() {
	must.MustF(RootCmd.Execute(), "failed executing RootCmd: %v")
}

func runMain(*cobra.Command, []string) {
	pFlags := RootCmd.PersistentFlags()

	pFlags.Args()

	svcCommand := must.Return(pFlags.GetString("service"))

	if svcCommand == "" { // app run as rport client
		runClient()
	} else { // app run to change state of OS service
		manageService(svcCommand)
	}
}

func decodeConfig(cfgPath string) (*chclient.ClientConfigHolder, error) {

	viperCfg := viper.New()
	viperCfg.SetConfigType("toml")

	cli_boilerplate.SetViperConfigDefaults(viperCfg)

	if cfgPath != "" {
		viperCfg.SetConfigFile(cfgPath)
	} else {
		viperCfg.AddConfigPath(".")
		viperCfg.SetConfigName("rport.conf")
	}
	config := &chclient.ClientConfigHolder{Config: &clientconfig.Config{}}

	if err := chshare.DecodeViperConfig(viperCfg, config.Config, nil); err != nil {
		return nil, err
	}

	if config.InterpreterAliases == nil {
		config.InterpreterAliases = map[string]string{}
	}

	return config, nil
}

func overrideConfigWithCliParamsForDevelopment(config *chclient.ClientConfigHolder) {

	pFlags := RootCmd.PersistentFlags()
	args := pFlags.Args()

	if len(args) > 0 {
		config.Client.Server = args[0]
		config.Client.Remotes = args[1:]
	}

	// TODO: finish tunnel config override ---v does not compile on purpose
	config.Tunnels.Scheme = *tunnelsScheme
	config.Tunnels.ReverseProxy = *tunnelsReverseProxy
	config.Tunnels.HostHeader = *tunnelsHostHeader
}

func runClient() {
	pFlags := RootCmd.PersistentFlags()

	cfgPath := must.Return(pFlags.GetString("config"))

	config := must.ReturnF(decodeConfig(cfgPath))("Invalid config: %v. Check your config file.")

	overrideConfigWithCliParamsForDevelopment(config)

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

func manageService(svcCommand string) {
	var svcUser string
	pFlags := RootCmd.PersistentFlags()
	cfgPath := must.Return(pFlags.GetString("config"))

	if runtime.GOOS != "windows" {
		svcUser = must.Return(pFlags.GetString("service-user"))
	}

	if svcCommand == "install" {
		// validate config file without command line args before installing it for the service
		// other service commands do not change config file specified at install

		config := must.ReturnF(decodeConfig(cfgPath))("Invalid config: %v. Check your config file.")

		must.MustF(config.ParseAndValidate(true), "config validation failed: %v")

	}

	must.Must(handleSvcCommand(svcCommand, cfgPath, svcUser))
}
