package adapter

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/lestrrat/go-server-starter/listener"
	"github.com/shogo82148/go-gracedown"
)

// Version is the version of go-nginx-oauth2-adapter.
const Version = "0.1.0"

// Main starts the go-nginx-oauth2-adapter server.
func Main(args []string) int {

	flagSet := flag.NewFlagSet(args[0], flag.ContinueOnError)
	var configFile string
	var configtest bool
	var showVersion bool
	var showHelp bool
	flagSet.StringVar(&configFile, "c", "", "configuration file")
	flagSet.StringVar(&configFile, "config", "", "configuration file")
	flagSet.BoolVar(&configtest, "t", false, "test configuration and exit")
	flagSet.BoolVar(&configtest, "configtest", false, "test configuration and exit")
	flagSet.BoolVar(&showVersion, "v", false, "show version information")
	flagSet.BoolVar(&showVersion, "version", false, "show version information")
	flagSet.BoolVar(&showHelp, "h", false, "show help")
	flagSet.BoolVar(&showHelp, "help", false, "show help")
	err := flagSet.Parse(args[1:])
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("error while parsing flags")
		return 2
	}

	if showVersion {
		fmt.Println("go-nginx-oauth2-adapter", Version)
		return 0
	}

	if showHelp {
		flagSet.Usage()
		return 0
	}

	c, err := parseConfig(configFile)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("error while parsing configure")
		return 1
	}

	startWatchSignal()
	l, err := getListener(c)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("listen error")
		return 1
	}

	s, err := NewServer(*c)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"err": err.Error(),
		}).Error("init server error")
		return 1
	}
	if configtest {
		return 0
	}

	gracedown.Serve(l, LoggingHandler(s))
	return 0
}

func parseConfig(configFile string) (*Config, error) {
	c := NewConfig()
	if err := c.LoadEnv(); err != nil {
		return nil, err
	}
	if configFile != "" {
		if err := c.LoadYaml(configFile); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func startWatchSignal() {
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan, syscall.SIGTERM)
	go func() {
		for {
			s := <-signalChan
			if s == syscall.SIGTERM {
				logrus.Info("received SIGTERM")
				gracedown.Close()
			}
		}
	}()
}

func getListener(c *Config) (net.Listener, error) {
	listeners, err := listener.ListenAll()
	if err != nil && err != listener.ErrNoListeningTarget {
		panic(err)
	}
	if err != listener.ErrNoListeningTarget {
		return listeners[0], nil
	}

	// Fallback if not running under Server::Starter
	return net.Listen("tcp", c.Address)
}
