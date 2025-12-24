package posixflag

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrUnknownFlag  = errors.New("unknown flag")
	ErrMissingValue = errors.New("flag missing value")
)

type FlagSet struct {
	flags       map[string]*Flag
	shortToLong map[rune]string
	parsed      bool
	args        []string
}

type Flag struct {
	Name     string
	Short    rune
	Usage    string
	Value    Value
	DefValue string
}

type Value interface {
	String() string
	Set(string) error
}

type boolValue bool

func (b *boolValue) String() string {
	return strconv.FormatBool(bool(*b))
}

func (b *boolValue) Set(s string) error {
	if s == "" {
		*b = true
		return nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*b = boolValue(v)
	return nil
}

type stringValue string

func (s *stringValue) String() string {
	return string(*s)
}

func (s *stringValue) Set(value string) error {
	*s = stringValue(value)
	return nil
}

func NewFlagSet() *FlagSet {
	return &FlagSet{
		flags:       make(map[string]*Flag),
		shortToLong: make(map[rune]string),
	}
}

func (fs *FlagSet) BoolVar(p *bool, name string, short rune, value bool, usage string) {
	*p = value
	fs.Var((*boolValue)(p), name, short, usage)
}

func (fs *FlagSet) StringVar(p *string, name string, short rune, value string, usage string) {
	*p = value
	fs.Var((*stringValue)(p), name, short, usage)
}

func (fs *FlagSet) Var(value Value, name string, short rune, usage string) {
	flag := &Flag{
		Name:     name,
		Short:    short,
		Usage:    usage,
		Value:    value,
		DefValue: value.String(),
	}
	fs.flags[name] = flag
	if short != 0 {
		fs.shortToLong[short] = name
	}
}

func (fs *FlagSet) Parse(args []string) error {
	fs.parsed = true
	fs.args = []string{}

	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			fs.args = append(fs.args, arg)
			continue
		}

		if arg == "--" {
			fs.args = append(fs.args, args[i+1:]...)
			break
		}

		if strings.HasPrefix(arg, "--") {
			name := arg[2:]
			var value string

			idx := strings.Index(name, "=")
			if idx >= 0 {
				value = name[idx+1:]
				name = name[:idx]
			}

			flag, exists := fs.flags[name]
			if !exists {
				return fmt.Errorf("%w: --%s", ErrUnknownFlag, name)
			}

			if _, isBool := flag.Value.(*boolValue); isBool && idx < 0 {
				value = ""
			} else if idx < 0 {
				if i+1 >= len(args) {
					return fmt.Errorf("%w: --%s", ErrMissingValue, name)
				}
				i++
				value = args[i]
			}

			if err := flag.Value.Set(value); err != nil {
				return fmt.Errorf("invalid value for --%s: %w", name, err)
			}

			continue
		}

		shortFlags := arg[1:]
		for j, short := range shortFlags {
			name, exists := fs.shortToLong[short]
			if !exists {
				return fmt.Errorf("%w: -%c", ErrUnknownFlag, short)
			}

			flag := fs.flags[name]

			if _, isBool := flag.Value.(*boolValue); isBool {
				if err := flag.Value.Set(""); err != nil {
					return fmt.Errorf("invalid value for -%c: %w", short, err)
				}
				continue
			}

			if j < len(shortFlags)-1 {
				value := shortFlags[j+1:]
				if err := flag.Value.Set(value); err != nil {
					return fmt.Errorf("invalid value for -%c: %w", short, err)
				}
				break
			}

			if i+1 >= len(args) {
				return fmt.Errorf("%w: -%c", ErrMissingValue, short)
			}
			i++
			if err := flag.Value.Set(args[i]); err != nil {
				return fmt.Errorf("invalid value for -%c: %w", short, err)
			}
		}
	}

	return nil
}

func (fs *FlagSet) Args() []string {
	if !fs.parsed {
		return nil
	}
	return fs.args
}

func (fs *FlagSet) Lookup(name string) *Flag {
	return fs.flags[name]
}

func (fs *FlagSet) Parsed() bool {
	return fs.parsed
}
