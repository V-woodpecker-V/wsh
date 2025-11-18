package main

import (
	"fmt"
	"os"
	"strings"
)

type WFlag struct {
	Short                 string
	Long                  string
	Help                  string
	Parent                *WFlag
	Children              []*WFlag
	ValueRequired         bool
	NonEmptyValueRequired bool
	IsSet                 bool
	Value                 string
}

func main() {
	addFlag := &WFlag{
		Short: "A",
		Long:  "add",
		Help:  "add a new flag",
	}
	addFlag.Children = []*WFlag{
		{
			Short:         "s",
			Long:          "short",
			Help:          "short version of a flag",
			Parent:        addFlag,
			ValueRequired: true,
		},
		{
			Short:         "l",
			Long:          "long",
			Help:          "long version of a flag",
			Parent:        addFlag,
			ValueRequired: true,
		},
		{
			Short:         "h",
			Long:          "help",
			Help:          "help message of a flag",
			Parent:        addFlag,
			ValueRequired: true,
		},
		{
			Short:                 "p",
			Long:                  "parent",
			Help:                  "which flag to put it under",
			Parent:                addFlag,
			NonEmptyValueRequired: true,
		},
		{
			Short:  "v",
			Long:   "value",
			Help:   "this flag requires a value",
			Parent: addFlag,
		},
		{
			Short:  "V",
			Long:   "non_empty_value",
			Help:   "this flag requires a value that is not empty",
			Parent: addFlag,
		},
	}

	flags := []*WFlag{addFlag}

	ParseArgs(flags, os.Args[1:])

	boolToInt := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	for _, f := range flags {
		fmt.Printf("%d -%s --%s - '%s'\n", boolToInt(f.IsSet), f.Short, f.Long, f.Value)
	}
}

func ParseArgs(flags []*WFlag, args []string) error {
	pArgs := preprocessArgs(args)

	var curValueFlag *WFlag
	curFlagContext := flags

	for _, arg := range pArgs {
		var f *WFlag
		if strings.HasPrefix(arg, "-") {
			for f == nil {
				f = matchFlag(curFlagContext, arg)
			}
		}
		if f == nil {
			if curValueFlag == nil || (strings.HasPrefix(arg, "-") && !strings.Contains(arg, " ")) {
				Error(fmt.Sprintf("unknown argument: %s", arg))
				return fmt.Errorf("unknown argument: %s", arg)
			}
			curValueFlag.Value = strings.Join([]string{curValueFlag.Value, arg}, " ")
		} else {
			f.IsSet = true
			if f.ValueRequired || f.NonEmptyValueRequired {
				curValueFlag = f
			}
		}
	}
	return nil
}

func preprocessArgs(args []string) []string {
	processedArgs := []string{}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "--") {
			processedArgs = append(processedArgs, strings.Trim(arg, " "))
		} else {
			for _, char := range []rune(arg)[1:] {
				if char == ' ' {
					continue
				}
				processedArgs = append(processedArgs, fmt.Sprintf("-%c", char))
			}
		}
	}
	return processedArgs
}

func matchFlag(flags []*WFlag, arg string) *WFlag {
	for _, wFlag := range flags {
		a := strings.TrimLeft(arg, "-")
		if (strings.HasPrefix(arg, "--") && a == wFlag.Long) ||
			(strings.HasPrefix(arg, "-") && a == wFlag.Short) {
			return wFlag
		}
	}
	return nil
}

func Log(s string) {
	fmt.Println(s)
}

func Stdout(s string) {

}

func Debug(s string) {

}

func Warn(s string) {

}

func Error(s string) {
	Log(s)
}
