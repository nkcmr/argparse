package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func main() {
	_ = rootCommand().Execute()
}

var configPrefixPattern = regexp.MustCompile(`^#\s+(flag|param):(.*)$`)
var paramConfigPattern = regexp.MustCompile(`(?i)^(?P<name>[a-z_][a-z0-9_]*)\s+(?P<help>.+)$`)
var flagConfigPattern = regexp.MustCompile(`(?i)^((?P<name>[a-z_][a-z0-9-_]*)(,(?P<short>[a-z]))?)\s+(?P<help>.+)$`)
var flagDefaultPattern = regexp.MustCompile(`(?i)\(\s*default\s*:\s*(?P<default_value>[^\)]+)\)`)
var codegenBoundaryPattern = regexp.MustCompile(`^#\s*argparse:(?P<kind>start|stop).*$`)

func namedMatches(r *regexp.Regexp, s string) map[string]string {
	groupNames := r.SubexpNames()
	matches := r.FindAllStringSubmatch(s, -1)
	if matches == nil {
		return nil
	}
	out := map[string]string{}
	for groupIdx, group := range matches[0] {
		name := groupNames[groupIdx]
		if name == "" {
			continue
		}
		out[name] = group
	}
	return out
}

type flag struct {
	LineNumber                 int
	Name, Short, Default, Help string
}

func (f flag) ToVarName() string {
	return regexp.MustCompile(`[^a-zA-Z0-9_]+`).ReplaceAllString(f.Name, "_")
}

type param struct {
	LineNumber int
	Name, Help string
}

func rootCommand() *cobra.Command {
	var args struct {
		inplace bool
	}
	cmd := &cobra.Command{
		Use:  "argparse SCRIPT_FILE",
		Args: cobra.ExactArgs(1),
		Run: runWithError(func(cmd *cobra.Command, pargs []string) error {
			scriptFileMode, err := os.Stat(pargs[0])
			if err != nil {
				return errors.Wrap(err, "failed to stat script file")
			}
			scriptContents, err := os.ReadFile(pargs[0])
			if err != nil {
				return errors.Wrap(err, "failed to read script file")
			}

			scriptLines := bytes.Split(scriptContents, osLineEnding())

			foundPreviousStart, foundPreviousStop := int(-1), int(-1)
			flags := []flag{}
			params := []param{}
			for lineIndex, line := range scriptLines {
				configMatches := configPrefixPattern.FindAllStringSubmatch(string(line), -1)
				if configMatches != nil {
					configType, config := configMatches[0][1], configMatches[0][2]
					switch configType {
					case "flag":
						matches := namedMatches(flagConfigPattern, config)
						if matches == nil {
							return fmt.Errorf(`invalid flag config on line %d, expected "flag-name(,f) help text" (parenthesis indicate optional parts)`, lineIndex+1)
						}
						defaultMatch := namedMatches(flagDefaultPattern, matches["help"])
						defaultValue := ""
						if defaultMatch != nil {
							defaultValue = defaultMatch["default_value"]
						}
						flags = append(flags, flag{
							LineNumber: lineIndex,
							Name:       matches["name"],
							Short:      matches["short"],
							Default:    defaultValue,
							Help:       matches["help"],
						})
					case "param":
						matches := namedMatches(paramConfigPattern, config)
						if matches == nil {
							return fmt.Errorf(`invalid param config on line %d, expected "param-name help text"`, lineIndex+1)
						}
						params = append(params, param{
							LineNumber: lineIndex,
							Name:       matches["name"],
							Help:       matches["help"],
						})
					default:
						panic(fmt.Sprintf("unknown argparse config type '%s'", configType))
					}
					continue
				}

				boundaryPatternMatch := namedMatches(codegenBoundaryPattern, string(line))
				if boundaryPatternMatch != nil {
					switch which := boundaryPatternMatch["kind"]; which {
					case "start":
						if foundPreviousStart >= 0 {
							return fmt.Errorf("2 'argparse:start' markers found on line %d and %d, expecting only 1", foundPreviousStart, lineIndex+1)
						}
						if foundPreviousStop >= 0 {
							return fmt.Errorf("'argparse:stop' (line: %d) found before 'argparse:start' (line: %d)", foundPreviousStop, lineIndex+1)
						}
						foundPreviousStart = lineIndex
					case "stop":
						if foundPreviousStop >= 0 {
							return fmt.Errorf("2 'argparse:stop' markers found on line %d and %d, expected only 1", foundPreviousStop, lineIndex+1)
						}
						if foundPreviousStart < 0 {
							return fmt.Errorf("expected 'argparse:start' to be found before 'argparse:stop' (line: %d)", lineIndex+1)
						}
						foundPreviousStop = lineIndex
					default:
						panic(fmt.Sprintf("unknown argparse marker type '%s'", which))
					}
					continue
				}
			}
			if foundPreviousStart >= 0 && foundPreviousStop < 0 {
				return fmt.Errorf("unterminated argparse section started on line %d", foundPreviousStart+1)
			}

			knownShort := map[string]flag{
				"h": {Name: "help"},
			}
			for _, f := range flags {
				if f.Short == "" {
					continue
				}
				first, taken := knownShort[f.Short]
				if taken {
					return fmt.Errorf("both %s and %s have conflicting short names", first.Name, f.Name)
				}
				knownShort[f.Short] = f
			}
			knownLong := map[string]flag{
				"help": {Name: "help"},
			}
			for _, f := range flags {
				first, taken := knownLong[f.Name]
				if taken {
					return fmt.Errorf("both %s and %s have conflicting names", first.Name, f.Name)
				}
				knownLong[f.Name] = f
			}

			result := codegenArgparse(params, flags)

			lastMatchedLine := max(
				reduceSlice(params, func(mem int, p param, _ int) int { return max(mem, p.LineNumber) }, int(-1)),
				reduceSlice(flags, func(mem int, f flag, _ int) int { return max(mem, f.LineNumber) }, int(-1)),
			)
			newLines := slices.Clone(mapSlice[[]byte, string, [][]byte](scriptLines, func(b []byte) string { return string(b) }))
			if foundPreviousStart == -1 || foundPreviousStop == -1 {
				if lastMatchedLine == -1 {
					fmt.Fprintln(os.Stderr, "# (DO NOT COPY THIS LINE) Unable to find a place to splice argparse section, just printing it out instead:")
					fmt.Println(result)
				} else {
					newLines = slices.Insert(newLines, lastMatchedLine+1, strings.Split(result, "\n")...)
				}
			} else {
				newLines = slices.Delete(newLines, foundPreviousStart, foundPreviousStop+1)
				newLines = slices.Insert(newLines, lastMatchedLine+1, strings.Split(result, "\n")...)
			}

			if args.inplace {
				if err := os.WriteFile(pargs[0], []byte(strings.Join(newLines, "\n")), scriptFileMode.Mode()); err != nil {
					return errors.Wrap(err, "failed to write script file")
				}
			} else {
				fmt.Print(strings.Join(newLines, "\n"))
			}

			return nil
		}),
	}
	cmd.Flags().BoolVarP(&args.inplace, "in-place", "i", false, "Should the edit be done in place?")
	return cmd
}

func runWithError(fn func(cmd *cobra.Command, pargs []string) error) func(*cobra.Command, []string) {
	return func(c *cobra.Command, s []string) {
		if err := fn(c, s); err != nil {
			fatal(err.Error())
			return
		}
	}
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, filepath.Base(os.Args[0])+": error: "+format+"\n", a...)
	_ = os.Stderr.Sync()
	os.Exit(1)
}

func osLineEnding() []byte {
	var b bytes.Buffer
	_, _ = fmt.Fprintln(&b)
	return b.Bytes()
}

type templateData struct {
	Params            []param
	Flags             []flag
	MaxFlagNameLength int
}

func (t templateData) HasParams() bool {
	return len(t.Params) > 0
}

var argParseTemplate = template.Must(template.New("argparse_bash").Funcs(template.FuncMap{
	"ToUpper": strings.ToUpper,
	"sub":     func(a, b int) int { return a - b },
	"quote_escape": func(s string) string {
		return strings.NewReplacer(`"`, `\"`).Replace(s)
	},
	"FormatFlagHelp": func(name, short, help string, maxNameWidth int) string {
		const shortFlagSize = 2
		const longFlagPrefix = 2
		const formattingSpace = 2 // ", "
		finalPaddingSize := shortFlagSize + formattingSpace + longFlagPrefix + maxNameWidth
		flagRep := fmt.Sprintf("--%s", name)
		if short != "" {
			flagRep = "-" + short + ", " + flagRep
		}
		return fmt.Sprintf("  %-"+strconv.Itoa(finalPaddingSize)+"s   %s", flagRep, help)
	},
}).Parse(`# argparse:start BELOW IS AUTO-GENERATED - DO NOT TOUCH (by: code.nkcmr.net/argparse)
{{- range $p := .Params }}
param_{{ $p.Name }}=""
{{- end }}
{{- if .HasParams }}
_arg_parse_params_set=0
{{- end }}
{{- range $f := .Flags }}
flag_{{ $f.ToVarName }}="{{ $f.Default | quote_escape }}"
{{- end }}
while [[ $# -gt 0 ]] ; do
	case "$(echo "$1" | cut -d= -f1)" in
		-h | --help)
			echo "Usage:"
			echo "  $0{{ range $p := $.Params }} {{ $p.Name | ToUpper }}{{ end }} [flags]"
			{{- if gt (len .Params) 0 }}
			echo
			{{- end }}
			{{- range $p := .Params }}
			echo "  {{ $p.Name | ToUpper }}: {{ $p.Help }}"
			{{- end }}
			echo
			echo "Flags:"
			echo "{{ FormatFlagHelp "help" "h" "print this help message" $.MaxFlagNameLength }}"
			{{- range $f := .Flags }}
			echo "{{ FormatFlagHelp $f.Name $f.Short $f.Help $.MaxFlagNameLength }}"
			{{- end }}
			exit 1
		;;
		{{- range $f := .Flags }}
		{{ if ne $f.Short "" }}-{{ $f.Short }} | {{ end }}--{{ $f.Name }})
			if [[ $# -eq 1 ]] || [[ "$2" == -* ]] ; then
				if [[ "$1" == *=* ]] ; then
					flag_{{ $f.ToVarName }}="$(echo "$1" | cut -d= -f2-)"
				else
					flag_{{ $f.ToVarName }}=true
				fi
			else
				shift
				flag_{{ $f.ToVarName }}="$1"
			fi
		;;
		{{- end }}
		-*)
			printf 'Unknown flag "%s"' "$1" ; echo
			exit 1
		;;
		*)
		{{- if .HasParams }}
			{{- range $i, $p := .Params }}
			{{- if eq $i 0 }}
			if [ $_arg_parse_params_set -eq {{ $i }} ] ; then
			{{- else }}
			elif [ $_arg_parse_params_set -eq {{ $i }} ] ; then
			{{- end }}
				param_{{ $p.Name }}="$1"
				((_arg_parse_params_set++))
			{{- if eq $i (sub (len $.Params) 1) }}
			else
			{{- end }}
			{{- end }}
				((_arg_parse_params_set++))
				echo "$0: error: accepts {{ len .Params }} args(s), received $_arg_parse_params_set"
				exit 1
			fi
		{{- else }}
			echo "$0: error: accepts 0 args, received 1 or more"
			exit 1
		{{- end }}
		;;
	esac
	shift
done
{{- if .HasParams }}
if [[ $_arg_parse_params_set -lt {{ len .Params }} ]] ; then
	echo "$0: error: accepts {{ len .Params }} arg(s), received $_arg_parse_params_set"
	exit 1
fi
unset _arg_parse_params_set
{{- end }}
# argparse:stop ABOVE CODE IS AUTO-GENERATED - DO NOT TOUCH`))

func codegenArgparse(params []param, flags []flag) string {
	var s strings.Builder
	longestFlagName := len("help")
	for _, f := range flags {
		if len(f.Name) > longestFlagName {
			longestFlagName = len(f.Name)
		}
	}
	err := argParseTemplate.Execute(&s, templateData{
		Params:            params,
		Flags:             flags,
		MaxFlagNameLength: longestFlagName,
	})
	if err != nil {
		panic(err)
	}
	return s.String()
}

func mapSlice[A, B any, S ~[]A](s S, mf func(A) B) []B {
	out := make([]B, len(s))
	for i := range s {
		out[i] = mf(s[i])
	}
	return out
}

func reduceSlice[E, R any, S ~[]E](in S, rf func(R, E, int) R, initial R) R {
	r := initial
	for i := range in {
		r = rf(r, in[i], i)
	}
	return r
}
