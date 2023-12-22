# argparse

generate next-level arg/param parsing for your bash scripts!

this is a simple tool that allows you to write specially crafted comments in your bash scripts that will then be used to generate a section of the script to parse arguments and flags.

so, if this is in a bash script:

```bash
#!/bin/bash

# param:one Some help text
# flag:dry-run Something about --dry-run
# flag:short,s Demoing the short flag

echo "one:     $param_one"
echo "dry-run: $flag_dry_run"
echo "short:   $flag_short"
```

will have an `argparse` section spliced in right under the configuration that will parse everything accordingly. (see examples directory for preview of generated code)

## installation

the only requirement is Go 1.21+

```
go install -v code.nkcmr.net/argparse@latest
```

## config format

the current format of the config is as follows.

### param

Parameters are required, positional arguments. This is the structure of a param config line:

```
param:<name: [a-z_][a-z0-9_]*> <help text>
```

#### examples

```
param:one Will be the first positional argument.
param:two Will be the second
param:3three INVALID: cannot start with number
```

### flag

Flags are optional, non-ordered arguments. They can be given a value upon invocation like: `--one two` or if they are passed without a value `--one` it is assumed that the value should just be `true`. This is the structure of a flag config line:

```
flag:<name: [a-z_][a-z0-9-_]*>(,<short_flag: [a-z]>)? <help text>
```

#### examples

```
flag:foo,f Generates a flag that can be "--foo <value>" or "-f"
flag:bar Generates a flag that can only be "--bar <value>" or "--bar"
flag:1foo INVALID: Will be ignored since it is invalid (cannot start with number)
flag:,f INVALID: Cannot only be short, must have long name
```

##### license

```
Copyright (c) 2023 Nicholas Comer

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

