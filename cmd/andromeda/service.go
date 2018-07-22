package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kardianos/service"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"jaytaylor.com/andromeda/crawler"
)

var (
	SystemUser     string
	SystemPassword string
)

func newServiceCmd() *cobra.Command {
	serviceCmd := &cobra.Command{
		Use:     "service",
		Aliases: []string{"svc", "sv"},
		Short:   "Andromeda system services management",
		Long:    "Install or remove Andromeda system services",
	}

	serviceCmd.AddCommand(
		newServerServiceCmd(),
		newCrawlerServiceCmd(),
	)

	return serviceCmd
}

func newServerServiceCmd() *cobra.Command {
	serverServiceCmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"srv", "s"},
		Short:   "Andromeda Server system service management",
		Long:    "Install or remove the Andromeda Crawler system service",
	}

	serverServiceCmd.AddCommand(
		newServerServiceInstallCmd(),
		newServerServiceRemoveCmd(),
	)

	return serverServiceCmd
}

func newServerServiceInstallCmd() *cobra.Command {
	serverServiceInstallCmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"inst", "i"},
		Short:   "Install Andromeda Server system service",
		Long:    "Install Andromeda Server system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			panic("not yet implemented")

			// svcConfig := &service.Config{
			// 	Name:        "andromeda-web",
			// 	DisplayName: "Andromeda Web Server",
			// 	Description: "Andromeda Web Server System Service",
			// 	Arguments: []string{
			// 		"-v",
			// 	},
			// 	Executable: os.Args[0],
			// 	WorkingDirectory: filepath.Dir(os.Args[0])
			// }

			// dbCfg := db.NewBoltConfig(DBFile)
			// dbCfg.BoltOptions.Timeout = 5 * time.Second
			// if err := db.WithClient(dbCfg, func(dbClient db.Client) error {
			// 	master := crawler.NewMaster(dbClient, crawler.NewConfig())
			// 	cfg := &web.Config{
			// 		Addr: WebAddr,
			// 		// TODO: DevMode
			// 		Master: master,
			// 	}
			// 	ws := web.New(dbClient, cfg)
			// 	if err := ws.Start(); err != nil {
			// 		return err
			// 	}
			// 	log.Infof("Web service started on %s", ws.Addr())
			// 	sigCh := make(chan os.Signal, 1)
			// 	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
			// 	select {
			// 	case s := <-sigCh:
			// 		log.WithField("sig", s).Info("Received signal, shutting down web service..")
			// 		if err := ws.Stop(); err != nil {
			// 			return err
			// 		}
			// 	}
			// 	return nil
			// }); err != nil {
			// 	log.Fatalf("main: %s", err)
			// }
		},
	}

	serverServiceInstallCmd.Flags().StringVarP(&WebAddr, "addr", "a", "", "Interface bind address:port spec")

	return serverServiceInstallCmd
}

func newServerServiceRemoveCmd() *cobra.Command {
	serverServiceRemoveCmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"uninst", "u", "remove", "rm", "r"},
		Short:   "Uninstall Andromeda Server system service",
		Long:    "Uninstall Andromeda Server system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			panic("not yet implemented")
		},
	}
	return serverServiceRemoveCmd
}

func newCrawlerServiceCmd() *cobra.Command {
	crawlerServiceCmd := &cobra.Command{
		Use:     "crawler",
		Aliases: []string{"c", "remote-crawler", "rc"},
		Short:   "Andromeda Crawler system service management",
		Long:    "Install or remove the Andromeda Crawler system service",
	}

	crawlerServiceCmd.AddCommand(
		newCrawlerServiceInstallCmd(),
		newCrawlerServiceRemoveCmd(),
		newCrawlerServiceStartCmd(),
		newCrawlerServiceStopCmd(),
		newCrawlerServiceRunCmd(),
	)

	return crawlerServiceCmd
}

func newCrawlerServiceInstallCmd() *cobra.Command {
	crawlerServiceInstallCmd := &cobra.Command{
		Use:     "install",
		Aliases: []string{"inst", "i"},
		Short:   "Install Andromeda Crawler system service",
		Long:    "Install Andromeda Crawler system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := doCrawlerServiceVerb("install"); err != nil {
				log.Fatal(err)
			}
			log.Info("Remote-Crawler service installed")
		},
	}

	addCrawlerFlags(crawlerServiceInstallCmd)

	addRemoteFlags(crawlerServiceInstallCmd)

	crawlerServiceInstallCmd.Flags().StringVarP(&SystemUser, "user", "u", "", "System username to run service as")
	crawlerServiceInstallCmd.Flags().StringVarP(&SystemPassword, "password", "p", "", "System user account password (windows only)")

	return crawlerServiceInstallCmd
}

func newCrawlerServiceRemoveCmd() *cobra.Command {
	crawlerServiceRemoveCmd := &cobra.Command{
		Use:     "uninstall",
		Aliases: []string{"uninst", "u", "remove", "rem", "rm", "r", "delete", "del", "d"},
		Short:   "Uninstall Andromeda Crawler system service",
		Long:    "Uninstall Andromeda Crawler system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := doCrawlerServiceVerb("uninstall"); err != nil {
				log.Fatal(err)
			}
			log.Info("Remote-Crawler service uninstalled")
		},
	}
	return crawlerServiceRemoveCmd
}

func newCrawlerServiceStartCmd() *cobra.Command {
	crawlerServiceStartCmd := &cobra.Command{
		Use:   "start",
		Short: "Start Andromeda Crawler system service",
		Long:  "Start Andromeda Crawler system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := doCrawlerServiceVerb("start"); err != nil {
				log.Fatal(err)
			}
			log.Info("Remote-Crawler service started")
		},
	}
	return crawlerServiceStartCmd
}

func newCrawlerServiceStopCmd() *cobra.Command {
	crawlerServiceStopCmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop Andromeda Crawler system service",
		Long:  "Stop Andromeda Crawler system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := doCrawlerServiceVerb("stop"); err != nil {
				log.Fatal(err)
			}
			log.Info("Remote-Crawler service stopped")
		},
	}
	return crawlerServiceStopCmd
}

func newCrawlerServiceRestartCmd() *cobra.Command {
	crawlerServiceRestartCmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart Andromeda Crawler system service",
		Long:  "Restart Andromeda Crawler system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := doCrawlerServiceVerb("restart"); err != nil {
				log.Fatal(err)
			}
			log.Info("Remote-Crawler service restarted")
		},
	}
	return crawlerServiceRestartCmd
}

func newCrawlerServiceRunCmd() *cobra.Command {
	crawlerServiceRunCmd := &cobra.Command{
		Use:   "run",
		Short: "Run Andromeda Crawler system service",
		Long:  "Run Andromeda Crawler system service",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := doCrawlerServiceVerb("run"); err != nil {
				log.Fatal(err)
			}
			log.Info("Remote-Crawler service ran")
		},
	}

	addCrawlerFlags(crawlerServiceRunCmd)
	addRemoteFlags(crawlerServiceRunCmd)

	return crawlerServiceRunCmd
}

func doCrawlerServiceVerb(action string) error {
	var (
		svcConfig = &service.Config{
			Name: "andromeda-crawler",
		}
		w *crawlerWrapper
	)

	switch action {
	case "run":
		w = newCrawlerWrapper()
		crawlerCfg := &crawler.Config{
			DeleteAfter: crawler.DefaultDeleteAfter,
			MaxItems:    crawler.DefaultMaxItems,
			SrcPath:     crawler.DefaultSrcPath,
		}
		w.rc = crawler.NewRemote(CrawlServerAddr, crawlerCfg)

	case "start", "stop", "restart", "uninstall":

	case "install":
		if SystemUser == "" {
			return errors.New("-u/--user flag must be specified")
		}

		svcConfig = &service.Config{
			Name:             "andromeda-crawler",
			DisplayName:      "Andromeda Remote-Crawler",
			Description:      "Andromeda Remote-Crawler System Service",
			WorkingDirectory: filepath.Join(filepath.Dir(os.Args[0]), "..", "src", "jaytaylor.com", "andromeda"),
			Arguments: []string{
				"service", "remote-crawler", "run",
				"--addr", CrawlServerAddr,
				"--src-path", crawler.DefaultSrcPath,
				"--max-items", fmt.Sprint(crawler.DefaultMaxItems),
				"--cert", TLSCertFile,
				"--key", TLSKeyFile,
				"--ca", TLSCAFile,
			},
			UserName: SystemUser,
			Option: map[string]interface{}{
				"Password": SystemPassword,
			},
		}
		if crawler.DefaultDeleteAfter {
			svcConfig.Arguments = append(svcConfig.Arguments, "--delete-after")
		}
		if log.GetLevel() == log.DebugLevel {
			svcConfig.Arguments = append(svcConfig.Arguments, "-v")
		} else if log.GetLevel() == log.ErrorLevel {
			svcConfig.Arguments = append(svcConfig.Arguments, "-q")
		}
		log.Debugf("Service arguments: %v", svcConfig.Arguments)

		w = newCrawlerWrapper()
		crawlerCfg := &crawler.Config{
			DeleteAfter: crawler.DefaultDeleteAfter,
			MaxItems:    crawler.DefaultMaxItems,
			SrcPath:     crawler.DefaultSrcPath,
		}
		w.rc = crawler.NewRemote(CrawlServerAddr, crawlerCfg)

	default:
		return fmt.Errorf("Unrecognized crawler service action: %q, must be one of: %v, or run", action, service.ControlAction)
	}

	s, err := service.New(w, svcConfig)
	if err != nil {
		return err
	}

	if action == "run" {
		return s.Run()
	}

	if err := service.Control(s, action); err != nil {
		return err
	}
	return nil
}

type crawlerWrapper struct {
	rc     *crawler.Remote
	stopCh chan struct{}
}

func newCrawlerWrapper() *crawlerWrapper {
	w := &crawlerWrapper{
		stopCh: make(chan struct{}),
	}
	return w
}

func (w *crawlerWrapper) Start(s service.Service) error {
	go w.rc.Run(w.stopCh)
	return nil
}

func (w *crawlerWrapper) Stop(s service.Service) error {
	w.stopCh <- struct{}{}
	return nil
}
