#!/bin/bash

set -euo pipefail
[ -n "${TRACE:-}" ] && set -x

# An attempt to demonstrate all capabilites of argparse
# param:one Some help text
# param:two argparse will look for 'one' first, then 'two'
# flag:dry-run Something about --dry-run
# flag:short,s Demoing the short flag
# argparse:start BELOW IS AUTO-GENERATED - DO NOT TOUCH (by: code.nkcmr.net/argparse)
param_one=""
param_two=""
_arg_parse_params_set=0
flag_dry_run=""
flag_short=""
while [[ $# -gt 0 ]] ; do
	case "$(echo "$1" | cut -d= -f1)" in
		-h | --help)
			echo "Usage:"
			echo "  $0 ONE TWO [flags]"
			echo
			echo "  ONE: Some help text"
			echo "  TWO: argparse will look for 'one' first, then 'two'"
			echo
			echo "Flags:"
			echo "  -h, --help      print this help message"
			echo "  --dry-run       Something about --dry-run"
			echo "  -s, --short     Demoing the short flag"
			exit 1
		;;
		--dry-run)
			if [[ $# -eq 1 ]] || [[ "$2" == -* ]] ; then
				if [[ "$1" == *=* ]] ; then
					flag_dry_run="$(echo "$1" | cut -d= -f2-)"
				else
					flag_dry_run=true
				fi
			else
				shift
				flag_dry_run="$1"
			fi
		;;
		-s | --short)
			if [[ $# -eq 1 ]] || [[ "$2" == -* ]] ; then
				if [[ "$1" == *=* ]] ; then
					flag_short="$(echo "$1" | cut -d= -f2-)"
				else
					flag_short=true
				fi
			else
				shift
				flag_short="$1"
			fi
		;;
		-*)
			printf 'Unknown flag "%s"' "$1" ; echo
			exit 1
		;;
		*)
			if [ $_arg_parse_params_set -eq 0 ] ; then
				param_one="$1"
				((_arg_parse_params_set++))
			elif [ $_arg_parse_params_set -eq 1 ] ; then
				param_two="$1"
				((_arg_parse_params_set++))
			else
				((_arg_parse_params_set++))
				echo "$0: error: accepts 2 args(s), received $_arg_parse_params_set"
				exit 1
			fi
		;;
	esac
	shift
done
if [[ $_arg_parse_params_set -lt 2 ]] ; then
	echo "$0: error: accepts 2 arg(s), received $_arg_parse_params_set"
	exit 1
fi
unset _arg_parse_params_set
# argparse:stop ABOVE CODE IS AUTO-GENERATED - DO NOT TOUCH

# the end result of parsing will be variables left that are prefixed according
# to type and suffixed with a snake_case version of the name.

echo "one:     $param_one"
echo "two:     $param_two"
echo "dry-run: $flag_dry_run"
echo "short:   $flag_short"
