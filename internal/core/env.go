package core

import (
	"os"
	"path/filepath"
)

func SystemEnv(base string) Env {
	env := Env{}
	add := func(key, value string) {
		if value != "" {
			env[key] = value
		}
	}
	add("base", base)
	home := os.Getenv("USERPROFILE")
	if home == "" {
		home = os.Getenv("HOME")
	}
	add("home", home)
	add("winAppData", os.Getenv("APPDATA"))
	add("winLocalAppData", os.Getenv("LOCALAPPDATA"))
	add("winPublic", os.Getenv("PUBLIC"))
	add("winProgramData", os.Getenv("PROGRAMDATA"))
	docs := systemDocuments(home)
	if docs == "" && home != "" {
		docs = filepath.Join(home, "Documents")
	}
	add("winDocuments", docs)
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" && home != "" {
		data = filepath.Join(home, ".local", "share")
	}
	add("xdgData", data)
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" && home != "" {
		config = filepath.Join(home, ".config")
	}
	add("xdgConfig", config)
	return env
}
