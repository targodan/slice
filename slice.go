package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
)

var allFormats = []string{
	"raw",
	"hex",
	"dump",
	"gobytes",
	"gostring",
	"cstring",
	"base64",
	"md5",
	"sha256",
}

type formatter func(out io.Writer, in io.Reader) error

func makeCPrintSafe(b byte) (string, bool) {
	escaped, ok := map[byte]string{
		'\a': "\\a",
		'\b': "\\b",
		'\f': "\\f",
		'\n': "\\n",
		'\r': "\\r",
		'\t': "\\t",
		'\v': "\\v",
		'\\': "\\\\",
		'"':  "\\\"",
	}[b]
	if ok {
		return escaped, false
	}

	if 20 <= b && b <= 126 {
		return fmt.Sprintf("%c", b), false
	} else {
		return fmt.Sprintf("\\x%02x", b), true
	}
}

var formatters = map[string]formatter{
	"raw": func(out io.Writer, in io.Reader) error {
		_, err := io.Copy(out, in)
		return err
	},
	"hex": func(out io.Writer, in io.Reader) error {
		enc := hex.NewEncoder(out)
		_, err := io.Copy(enc, in)
		if err != nil {
			return err
		}
		out.Write([]byte{'\n'})
		return nil
	},
	"dump": func(out io.Writer, in io.Reader) error {
		enc := hex.Dumper(out)
		_, err := io.Copy(enc, in)
		return err
	},
	"gobytes": func(out io.Writer, in io.Reader) error {
		data, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}
		fmt.Printf("%#v\n", data)
		return nil
	},
	"gostring": func(out io.Writer, in io.Reader) error {
		data, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}
		fmt.Printf("%#v\n", string(data))
		return nil
	},
	"cstring_unsafe": func(out io.Writer, in io.Reader) error {
		fmt.Fprint(out, "\"")
		var err error
		var n int
		var lastOutWasHex bool
		buf := make([]byte, 512)
		for err == nil {
			n, err = in.Read(buf)

			for _, b := range buf[:n] {
				str, isHex := makeCPrintSafe(b)
				if lastOutWasHex && !isHex {
					// split string literal to avoid problems like this
					// { 0x00, 'a' } -> "\x00a" could be parsed wrong
					fmt.Fprint(out, "\" \"")
				}
				fmt.Fprint(out, str)
				lastOutWasHex = isHex
			}
		}
		if err != io.EOF {
			return err
		}
		fmt.Fprint(out, "\"\n")
		return nil
	},
	"cstring": func(out io.Writer, in io.Reader) error {
		fmt.Fprint(out, "\"")
		var err error
		var n int
		buf := make([]byte, 512)
		for err == nil {
			n, err = in.Read(buf)

			for _, b := range buf[:n] {
				fmt.Fprintf(out, "\\x%02x", b)
			}
		}
		if err != io.EOF {
			return err
		}
		fmt.Fprint(out, "\"\n")
		return nil
	},
	"base64": func(out io.Writer, in io.Reader) error {
		enc := base64.NewEncoder(base64.StdEncoding, out)
		_, err := io.Copy(enc, in)
		if err != nil {
			return err
		}
		out.Write([]byte{'\n'})
		return nil
	},
	"md5": func(out io.Writer, in io.Reader) error {
		enc := md5.New()
		_, err := io.Copy(enc, in)
		if err != nil {
			return err
		}
		hex.NewEncoder(out).Write(enc.Sum(nil))
		out.Write([]byte{'\n'})
		return nil
	},
	"sha256": func(out io.Writer, in io.Reader) error {
		enc := sha256.New()
		_, err := io.Copy(enc, in)
		if err != nil {
			return err
		}
		hex.NewEncoder(out).Write(enc.Sum(nil))
		out.Write([]byte{'\n'})
		return nil
	},
}

func main() {
	app := &cli.App{
		Name:      "slice",
		Usage:     "outputs contents of binary files",
		UsageText: "slice [options] FILE",
		Writer:    os.Stderr,
		ErrWriter: os.Stderr,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "offset",
				Aliases: []string{"o"},
				Usage:   "offset of output in bytes",
				Value:   "0",
			},
			&cli.StringFlag{
				Name:    "size",
				Aliases: []string{"length", "s", "l"},
				Usage:   "size of output in bytes",
				Value:   "-1",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "output format, available: " + strings.Join(allFormats, ", "),
				Value:   "raw",
			},
		},
		Action: func(c *cli.Context) error {
			var offset, size int64

			_, err := fmt.Sscanf(c.String("offset"), "%v", &offset)
			if err != nil {
				return fmt.Errorf("could not parse offset, reason: %w", err)
			}
			_, err = fmt.Sscanf(c.String("size"), "%v", &size)
			if err != nil {
				return fmt.Errorf("could not parse offset, reason: %w", err)
			}

			fmtter, ok := formatters[c.String("format")]
			if !ok {
				return fmt.Errorf("unsupported formatter \"%s\"", c.String("format"))
			}

			if c.NArg() != 1 {
				return fmt.Errorf("expected exactly one argument, got %d", c.NArg())
			}
			filename := c.Args().Get(0)
			file, err := os.OpenFile(filename, os.O_RDONLY, 0666)
			if err != nil {
				return fmt.Errorf("could not open file, reason: %w", err)
			}
			defer file.Close()

			_, err = file.Seek(offset, io.SeekStart)
			if err != nil {
				return fmt.Errorf("could not seek to offset 0x%X, reason: %w", offset, err)
			}

			var in io.Reader
			if size != -1 {
				in = io.LimitReader(file, size)
			} else {
				in = file
			}

			err = fmtter(os.Stdout, in)
			if err != nil {
				return fmt.Errorf("an error occured during reading of the file: %w", err)
			}
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		if exitErr, ok := err.(cli.ExitCoder); ok {
			os.Exit(exitErr.ExitCode())
		} else {
			os.Exit(-1)
		}
	}
}
