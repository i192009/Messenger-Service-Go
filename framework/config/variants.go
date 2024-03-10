package config

import "time"

var IsRunning bool = true
var IsDebug bool
var ServiceName string
var ServiceTimeout time.Duration

var OnConfigVariantChanged func()

func initConfigVariants() {
	IsDebug = GetBoolean("debug", false)
	ServiceName = GetString("service.name", "")
	ServiceTimeout = time.Duration(GetInt("service.timeout", 1000)) * time.Second

	if OnConfigVariantChanged != nil {
		OnConfigVariantChanged()
	}
}
