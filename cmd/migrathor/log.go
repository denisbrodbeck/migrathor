package main

import (
	"fmt"
	"io"
	"log"

	"github.com/lib/pq"
)

func logCloser(c io.Closer, l *log.Logger) {
	if err := c.Close(); err != nil {
		l.Printf("failed to close handle: %+v", err)
	}
}

func formatPqError(err error) string {
	if e, ok := err.(*pq.Error); ok {
		msg := fmt.Sprintf("Severity   : %s\n", e.Severity)
		msg += fmt.Sprintf("Error Code : %s (%s)\n", e.Code, e.Code.Name())
		msg += fmt.Sprintf("Message    : %s\n", e.Message)
		if e.Detail != "" {
			msg += fmt.Sprintf("Detail     : %s\n", e.Detail)
		}
		if e.Hint != "" {
			msg += fmt.Sprintf("Hint       : %s\n", e.Hint)
		}
		if e.Position != "" {
			msg += fmt.Sprintf("Position   : %s\n", e.Position)
		}
		return msg
	}
	return err.Error()
}
