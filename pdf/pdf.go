package pdf

import (
	"fmt"
	"io"
)

const Version = "1.7"

type PDF struct {
	Body []Indirect
}

func Encode(w io.Writer, p *PDF) (err error) {
	_, err = fmt.Fprintf(bw, "%%PDF-%s\n", Version)
	if err != nil {
		return err
	}

	for _, obj := range p.Body {
		err := EncodeObject(w, obj)
		if err != nil {
			return err
		}
	}

	// TODO: Cross-reference section.

	// TODO: Trailer.

	_, err = io.WriteString(w, "%%EOF")
	return err
}
