package rconf

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var refreshLock sync.Mutex
var refreshTimer *time.Timer

func InitConfig(configBytes []byte, vpFunc func(viper2 *viper.Viper)) {
	v := viper.New()
	v.SetConfigType("yaml")

	if err := v.ReadConfig(bytes.NewBuffer(configBytes)); err != nil {
		panic(fmt.Errorf("viper.ReadConfig error: %w", err))
	}

	slog.Info(fmt.Sprintf("load ext config path: %s", v.GetString("app.ext-config-path")))

	v.SetConfigName("application_ext")
	v.AddConfigPath(v.GetString("app.ext-config-path"))

	// 如果找到外部配置文件，则合并配置（外部配置会覆盖嵌入配置）
	if err := v.MergeInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); !ok {
			panic(fmt.Errorf("failed to read external config: %w", err))
		}
		slog.Info("cannot find the external config file.")
	}

	// 支持环境变量覆盖
	v.AutomaticEnv()
	v.SetEnvPrefix("app_")

	vpFunc(v)

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		refreshLock.TryLock()
		defer refreshLock.Unlock()

		if refreshTimer != nil {
			refreshTimer.Stop()
		}
		refreshTimer = time.AfterFunc(5*time.Second, func() {
			slog.Info(fmt.Sprintf("refresh config file changed: %s", e.Name))
			vpFunc(v)
		})
	})
}
