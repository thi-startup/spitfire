package vmctl

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/thi-startup/spitfire/internal/config"
	"github.com/thi-startup/spitfire/internal/vm"
	"github.com/thi-startup/spitfire/pkg/utils"
)

func start() *cobra.Command {
	command := &cobra.Command{
		Use:     "start",
		Short:   "Create and start vmms",
		Aliases: []string{"s"},
		Example: `
  $ spitfire vmctl start 
`,
	}

	opts := config.NewVMMOption()
	if err := bindFlags(command, opts); err != nil {
		fmt.Println(fmt.Errorf("failed to register options: %w", err))
		os.Exit(1)
	}
	command.Flags().StringP("config", "", "", "Spitfire compose file")
	command.Flags().IntP("count", "", 1, "Number of firecracker machines to spin up")

	command.RunE = func(cmd *cobra.Command, args []string) error {
		configPath, _ := cmd.Flags().GetString("config")

		var vmConfigs *config.Config
		if utils.Exists(configPath) && configPath != "" {
			parsedConfig, err := config.ParseYAML(configPath)
			if err != nil {
				return fmt.Errorf("you need a valid config to start vms :(", err)
			}
			vmConfigs = parsedConfig
		}
		if len(vmConfigs.VMMConfigs) == 0 {
			if opts.FcKernelImage == "" && opts.FcRootDrivePath == "" {
				return fmt.Errorf("either config file or kernel + rootfs are required to start vm")
			}
			vmConfigs.VMMConfigs = map[string]config.VMMOption{"1": *opts}
		}
		for _, cfg := range vmConfigs.VMMConfigs {
			_, err := vm.StartVMM(context.Background(), &cfg)
			if err != nil {
				return err
			}
		}
		return nil
	}
	return command
}

func Root() *cobra.Command {
	command := &cobra.Command{
		Use:     "vmctl",
		Short:   "Manage vms",
		Aliases: []string{"vm"},
	}

	command.RunE = func(cmd *cobra.Command, args []string) error {
		// get the compose config `spitfire-compose.[yml|yaml]
		return nil
	}

	command.AddCommand(start())

	return command
}

func bindFlags(cmd *cobra.Command, flags interface{}) error {
	v := reflect.ValueOf(flags).Elem()
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		name := field.Tag.Get("long")
		if name == "" {
			continue
		}

		short := field.Tag.Get("short")
		desc := field.Tag.Get("description")
		defaultVal := field.Tag.Get("default")

		switch value.Kind() {
		case reflect.String:
			cmd.Flags().StringVarP(value.Addr().Interface().(*string), name, short, defaultVal, desc)
		case reflect.Int:
			defInt, _ := strconv.Atoi(defaultVal)
			cmd.Flags().IntVarP(value.Addr().Interface().(*int), name, short, defInt, desc)
		case reflect.Int64:
			defInt, _ := strconv.ParseInt(defaultVal, 10, 64)
			cmd.Flags().Int64VarP(value.Addr().Interface().(*int64), name, short, defInt, desc)
		case reflect.Bool:
			defBool, _ := strconv.ParseBool(defaultVal)
			cmd.Flags().BoolVarP(value.Addr().Interface().(*bool), name, short, defBool, desc)
		case reflect.Slice:
			switch value.Type().Elem().Kind() {
			case reflect.String:
				cmd.Flags().StringSliceVarP(value.Addr().Interface().(*[]string), name, short, nil, desc)
			case reflect.Int:
				cmd.Flags().IntSliceVarP(value.Addr().Interface().(*[]int), name, short, nil, desc)
			}
		default:
			log.Fatal("unhandled type for field %s: %s", field.Name, value.Kind())
		}
	}
	return nil
}
