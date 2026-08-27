package test_utils

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
)

const defaultDBConfigPath = "/var/run/redis/sonic-db/database_config.json"

func dbConfigPath() string {
	if path := os.Getenv("DB_CONFIG_PATH"); path != "" {
		return path
	}
	return defaultDBConfigPath
}

func dbRuntimeRoot() string {
	if root := os.Getenv("SONIC_GNMI_DB_RUNTIME_DIR"); root != "" {
		return root
	}
	return "/var/run"
}

type dbConfigInclude struct {
	Include       string `json:"include"`
	Namespace     string `json:"namespace,omitempty"`
	ContainerName string `json:"container_name,omitempty"`
}

func writeGlobalConfig(include dbConfigInclude) error {
	globalDir := filepath.Dir(dbConfigPath())
	baseInclude := filepath.Base(dbConfigPath())
	otherInclude, err := filepath.Rel(globalDir, include.Include)
	if err != nil {
		return err
	}
	include.Include = otherInclude
	config := struct {
		Includes []dbConfigInclude `json:"INCLUDES"`
		Version  string            `json:"VERSION"`
	}{
		Includes: []dbConfigInclude{{Include: baseInclude}, include},
		Version:  "1.0",
	}
	data, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(globalDir, "database_global.json"), data, 0644); err != nil {
		return err
	}
	return nil
}

func SetupMultiNamespace() error {
	redis0Dir := filepath.Join(dbRuntimeRoot(), "redis0", "sonic-db")
	err := os.MkdirAll(redis0Dir, 0755)
	if err != nil {
		return err
	}
	srcFileName := [1]string{"../testdata/database_config_asic0.json"}
	dstFileName := [1]string{filepath.Join(redis0Dir, "database_config_asic0.json")}
	for i := 0; i < len(srcFileName); i++ {
		sourceFileStat, err := os.Stat(srcFileName[i])
		if err != nil {
			return err
		}

		if !sourceFileStat.Mode().IsRegular() {
			return err
		}

		source, err := os.Open(srcFileName[i])
		if err != nil {
			return err
		}
		defer source.Close()

		destination, err := os.Create(dstFileName[i])
		if err != nil {
			return err
		}
		defer destination.Close()
		_, err = io.Copy(destination, source)
		if err != nil {
			return err
		}
	}
	return writeGlobalConfig(dbConfigInclude{Include: dstFileName[0], Namespace: "asic0"})
}

func CleanUpMultiNamespace() error {
	err := os.Remove(filepath.Join(filepath.Dir(dbConfigPath()), "database_global.json"))
	if err != nil {
		return err
	}
	err = os.RemoveAll(filepath.Join(dbRuntimeRoot(), "redis0", "sonic-db"))
	if err != nil {
		return err
	}
	return nil
}
func GetMultiNsNamespace() string {
	return "asic0"
}

func SetupMultiInstance() error {
	redisDPU0Dir := filepath.Join(dbRuntimeRoot(), "redisdpu0", "sonic-db")
	err := os.MkdirAll(redisDPU0Dir, 0755)
	if err != nil {
		return err
	}
	srcFileName := [1]string{"../testdata/database_config_dpu.json"}
	dstFileName := [1]string{filepath.Join(redisDPU0Dir, "database_config.json")}
	for i := 0; i < len(srcFileName); i++ {
		sourceFileStat, err := os.Stat(srcFileName[i])
		if err != nil {
			return err
		}

		if !sourceFileStat.Mode().IsRegular() {
			return err
		}

		source, err := os.Open(srcFileName[i])
		if err != nil {
			return err
		}
		defer source.Close()

		destination, err := os.Create(dstFileName[i])
		if err != nil {
			return err
		}
		defer destination.Close()
		_, err = io.Copy(destination, source)
		if err != nil {
			return err
		}
	}
	return writeGlobalConfig(dbConfigInclude{Include: dstFileName[0], ContainerName: "dpu0"})
}

func CleanUpMultiInstance() error {
	err := os.Remove(filepath.Join(filepath.Dir(dbConfigPath()), "database_global.json"))
	if err != nil {
		return err
	}
	err = os.RemoveAll(filepath.Join(dbRuntimeRoot(), "redisdpu0", "sonic-db"))
	if err != nil {
		return err
	}
	return nil
}
