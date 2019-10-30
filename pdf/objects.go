package pdf

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Object interface {
	encode(s *encodeState) error
}

type encodeState struct {
	w   *bufio.Writer
	err error

	nextName int
	names    map[string]int
}

func (s *encodeState) Close() error {
	if s.err != nil {
		return s.err
	}

	s.err = s.w.Flush()
	return s.err
}

func (s *encodeState) objName(name string) int {
	if s.names == nil {
		s.names = make(map[string]int)
	}

	if n, ok := s.names[name]; ok {
		return n
	}

	n := s.nextName
	s.nextName++
	s.names[name] = n
	return n
}

func (s *encodeState) Write(buf []byte) (int, error) {
	if s.err != nil {
		return 0, s.err
	}

	n, err := s.w.Write(buf)
	s.err = err
	return n, err
}

func (s *encodeState) WriteByte(c byte) error {
	if s.err != nil {
		return s.err
	}

	s.err = s.w.WriteByte(c)
	return s.err
}

func (s *encodeState) WriteRune(r rune) (int, error) {
	if s.err != nil {
		return 0, s.err
	}

	n, err := s.w.WriteRune(r)
	s.err = err
	return n, err
}

func (s *encodeState) WriteString(str string) (int, error) {
	if s.err != nil {
		return 0, s.err
	}

	n, err := s.w.WriteString(str)
	s.err = err
	return n, err
}

func EncodeObject(w io.Writer, obj Object) (err error) {
	if obj == nil {
		_, err := io.WriteString(w, "null")
		return err
	}

	ew := encodeState{w: bufio.NewWriter(w)}
	defer func() {
		cerr := ew.Close()
		if err == nil {
			err = cerr
		}
	}()

	err = obj.encode(&ew)
	if err != nil {
		return err
	}
	return ew.err
}

type Boolean bool

func (b Boolean) encode(s *encodeState) error {
	_, err := fmt.Fprint(s, bool(b))
	return err
}

type Integer int

func (i Integer) encode(s *encodeState) error {
	_, err := fmt.Fprint(s, int(i))
	return err
}

type Real float64

func (r Real) encode(s *encodeState) error {
	_, err := fmt.Fprintf(s, "%#f", float64(r))
	return err
}

type LiteralString string

func (str LiteralString) encode(s *encodeState) error {
	r := strings.NewReplacer(
		"(", `\(`,
		")", `\)`,
		`\`, `\\`,
	)

	s.WriteByte('(')
	r.WriteString(s, string(str))
	s.WriteByte(')')
	return nil
}

type HexString []byte

func (str HexString) encode(s *encodeState) error {
	_, err := fmt.Fprintf(s, "<%X>", []byte(str))
	return err
}

type Name string

func (n Name) encode(s *encodeState) error {
	s.WriteByte('/')
	for _, r := range n {
		if (r < '!') || (r > '~') {
			_, err := fmt.Fprintf(s, "#%X", r)
			if err != nil {
				return err
			}
			continue
		}

		if r == '#' {
			_, err := s.WriteString("#23")
			if err != nil {
				return err
			}
			continue
		}

		_, err := s.WriteRune(r)
		if err != nil {
			return err
		}
	}

	return nil
}

type Array []Object

func (a Array) encode(s *encodeState) error {
	spacing := ""

	s.WriteByte('[')
	for _, obj := range a {
		_, err := s.WriteString(spacing)
		if err != nil {
			return err
		}
		err = EncodeObject(s, obj)
		if err != nil {
			return err
		}

		spacing = " "
	}
	s.WriteByte(']')

	return nil
}

type Dict map[Name]Object

func (d Dict) encode(s *encodeState) error {
	s.WriteString("<<")
	for k, v := range d {
		s.WriteByte('\n')
		err := EncodeObject(s, k)
		if err != nil {
			return err
		}
		s.WriteByte(' ')
		err = EncodeObject(s, v)
		if err != nil {
			return err
		}
	}
	s.WriteString("\n>>")

	return nil
}

type Stream struct {
	Length int
	Data   io.Reader
}

func (stream Stream) encode(s *encodeState) error {
	err := EncodeObject(s, Dict{
		"Length": Integer(stream.Length),
	})
	if err != nil {
		return err
	}
	s.WriteString("\nstream\n")
	_, err = io.CopyN(s, stream.Data, int64(stream.Length))
	if err != nil {
		return err
	}
	s.WriteString("\nendstream\n")

	return nil
}

type Indirect struct {
	Name   string
	Object Object
}

func (i Indirect) encode(s *encodeState) error {
	n := s.objName(i.Name)
	err := Integer(n).encode(s)
	if err != nil {
		return err
	}

	s.WriteString(" 0 obj\n")
	err = i.Object.encode(s)
	if err != nil {
		return err
	}
	s.WriteString("\nendobj\n")
	return nil
}

type Reference string

func (r Reference) encode(s *encodeState) error {
	n := s.objName(string(r))
	err := Integer(n).encode(s)
	if err != nil {
		return err
	}
	s.WriteString(" 0 R")
	return nil
}
