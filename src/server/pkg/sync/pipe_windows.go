// +build windows

package sync

import (
	"fmt"
	"io"
)

func (p *Puller) makePipe(path string, f func(io.Writer) error) error {
	return fmt.Errorf("Lazy file sync through pipes is not supported on Windows")
}
