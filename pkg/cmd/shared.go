package cmd

import (
	"context"
	"reflect"
	"strings"

	"github.com/bombsimon/logrusr/v4"
	"github.com/mohamed-rafraf/kubectl-foren/pkg/state"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8c.io/kubeone/pkg/fail"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

type globalOptions struct {
	Verbose   bool   `longflag:"verbose" shortflag:"v"`
	Debug     bool   `longflag:"debug" shortflag:"d"`
	LogFormat string `longflag:"log-format" shortflag:"l"`
}

func (opts *globalOptions) BuildState() (*state.State, error) {
	rootContext := context.Background()
	s, err := state.New(rootContext)

	if err != nil {
		return nil, err
	}
	s.Logger = newLogger(opts.Verbose, opts.LogFormat)

	s.Verbose = opts.Verbose

	return s, nil
}

func persistentGlobalOptions(fs *pflag.FlagSet) (*globalOptions, error) {
	gf := &globalOptions{}

	verbose, err := fs.GetBool(longFlagName(gf, "Verbose"))
	if err != nil {
		return nil, fail.Runtime(err, "getting global flags")
	}
	gf.Verbose = verbose

	logFormat, err := fs.GetString(longFlagName(gf, "LogFormat"))
	if err != nil {
		return nil, fail.Runtime(err, "getting global flags")
	}
	gf.LogFormat = logFormat

	return gf, nil
}

func newLogger(verbose bool, format string) *logrus.Logger {
	logger := logrus.New()

	switch format {
	case "json":
		logger.Formatter = &logrus.JSONFormatter{}
	default:
		logger.Formatter = &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "15:04:05 MST",
		}
	}

	// Required by controller-runtime
	// https://github.com/kubernetes-sigs/controller-runtime/blob/658c55227830b2d895b12fc1c86cbc731e36d291/TMP-LOGGING.md#
	ctrlruntimelog.SetLogger(logrusr.New(logger))

	if verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}

func longFlagName(obj interface{}, fieldName string) string {
	elem := reflect.TypeOf(obj).Elem()
	field, ok := elem.FieldByName(fieldName)
	if !ok {
		return strings.ToLower(fieldName)
	}

	return field.Tag.Get("longflag")
}

func shortFlagName(obj interface{}, fieldName string) string {
	elem := reflect.TypeOf(obj).Elem()
	field, _ := elem.FieldByName(fieldName)

	return field.Tag.Get("shortflag")
}
