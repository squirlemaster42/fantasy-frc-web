package handler

import (
	"os/exec"

	"github.com/labstack/echo/v4"
)

func (h *Handler) StreamLogsHandler(c echo.Context) error {
	cmd := exec.Command(
		"journalctl",
		"-u", "fantasyfrc",
		"-f",           // follow (stream)
		"-o", "json",   // structured output
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Important headers for streaming
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Transfer-Encoding", "chunked")

	buf := make([]byte, 4096)

	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			_, writeErr := c.Response().Write(buf[:n])
			if writeErr != nil {
				break
			}
			c.Response().Flush()
		}
		if err != nil {
			break
		}
	}

	return nil
}
