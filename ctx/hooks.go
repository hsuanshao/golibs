package ctx

import (
	"errors"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// HookConfig struct define logrus hook setup
type HookConfig struct {
	Environment    string           `json:"env"`
	ServiceName    string           `json:"service_name"`
	AdapterName    string           `json:"adapter_name"`
	ProjectFolder  string           `json:"project_folder"`
	LogFormat      logrus.Formatter `json:"log_format"`
	ExportLevel    []logrus.Level   `json:"export_level"`
	ExportToFile   *string          `json:"export_filepath"`
	SetAsStdOutput bool             `json:"set_as_std_output"`
}

// NewHook to new a logrus hook
// HookConfig struct define logrus hook setup
func NewHook(conf *HookConfig) (logrus.Hook, error) {
	if conf == nil {
		return nil, errors.New("input nil config")
	}

	if len(conf.ExportLevel) == 0 {
		conf.ExportLevel = logrus.AllLevels
	}

	mLock := new(sync.Mutex)
	impl := &defaultHook{
		Env:           conf.Environment,
		ServiceName:   conf.ServiceName,
		AdapterName:   conf.AdapterName,
		ProjectFolder: conf.ProjectFolder,
		ExportLevels:  conf.ExportLevel,
		Formater:      conf.LogFormat,
		ExportToFile:  conf.ExportToFile,
		mu:            mLock,
	}

	if conf.LogFormat == nil {
		impl.Formater = &logrus.JSONFormatter{}
		if impl.Env == "localhost" {
			impl.Formater = &logrus.TextFormatter{ForceColors: true}
		}
	}

	return impl, nil
}

type defaultHook struct {
	Env           string
	ServiceName   string
	AdapterName   string
	ProjectFolder string
	ExportLevels  []logrus.Level
	ExportToFile  *string
	Formater      logrus.Formatter
	mu            *sync.Mutex
}

func (d *defaultHook) SetFormatter(format logrus.Formatter) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Formater = format
}

func (d *defaultHook) Levels() []logrus.Level {
	if len(d.ExportLevels) == 0 {
		d.ExportLevels = []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		}
	}
	return d.ExportLevels
}

func (d *defaultHook) Fire(entry *logrus.Entry) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pc := make([]uintptr, 3)
	cnt := runtime.Callers(6, pc)

	contains := func(sl []logrus.Level, lvl logrus.Level) bool {
		for _, l := range sl {
			if lvl == l {
				return true
			}
		}
		return false
	}(d.ExportLevels, entry.Level)

	if !contains {
		return nil
	}

	for i := 0; i < cnt; i++ {
		fu := runtime.FuncForPC(pc[i] - 1)
		name := fu.Name()
		if !strings.Contains(name, "github.com/sirupsen/logrus") {
			file, line := fu.FileLine(pc[i] - 1)
			//NOTE: IT ONLY WORKABLE FOR OUR INTERNAL MODULE
			cut := "/"
			if strings.Contains(file, "github.com/") {
				cut = "github.com/"
			}

			folderSplits := strings.Split(file, cut)
			pathArr := strings.Split(strings.Join(folderSplits[1:], "/"), "/")

			folder := ""

			for i, v := range pathArr {
				if i == len(pathArr)-1 {
					break
				}
				folder += v
				if i < len(pathArr)-2 {
					folder += "/"
				}
			}
			if d.ProjectFolder != "" {
				entry.Data["project"] = d.ProjectFolder
			}
			entry.Data["folder"] = folder
			entry.Data["file"] = pathArr[len(pathArr)-1]
			entry.Data["func"] = path.Base(name)
			entry.Data["line"] = line
			entry.Data["unix"] = entry.Time.UTC().Unix()
			break
		}
	}

	if d.ExportToFile != nil {
		logFile, err := os.OpenFile(*d.ExportToFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			settleLogger.WithField("err", err).Panic("ctx set logger export file path failed")
			return err
		}
		defer logFile.Close()
		b, _ := d.Formater.Format(entry)
		logFile.Write(b)
		return nil
	}

	d.Formater.Format(entry)
	return nil
}
