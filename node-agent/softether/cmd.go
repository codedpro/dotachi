package softether

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

const cmdTimeout = 30 * time.Second

// Client wraps vpncmd invocations against a SoftEther server.
type Client struct {
	VpncmdPath string
	ServerHost string
}

// ServerCmd runs a vpncmd command against the server (no hub context).
//
//	vpncmd <host> /SERVER /CMD <args...>
func (c *Client) ServerCmd(args ...string) (string, error) {
	cmdArgs := []string{c.ServerHost, "/SERVER", "/CMD"}
	cmdArgs = append(cmdArgs, args...)
	return runCmd(c.VpncmdPath, cmdArgs...)
}

// HubCmd runs a vpncmd command in the context of a specific hub.
//
//	vpncmd <host> /SERVER /HUB:<hubName> /CMD <args...>
func (c *Client) HubCmd(hubName string, args ...string) (string, error) {
	cmdArgs := []string{c.ServerHost, "/SERVER", "/HUB:" + hubName, "/CMD"}
	cmdArgs = append(cmdArgs, args...)
	return runCmd(c.VpncmdPath, cmdArgs...)
}

// runCmd executes vpncmd with a 30-second timeout, captures stdout+stderr,
// and returns an error if the exit code is non-zero.
func runCmd(vpncmdPath string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	log.Printf("[vpncmd] %s %s", vpncmdPath, strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, vpncmdPath, args...)
	out, err := cmd.CombinedOutput()
	output := string(out)

	if ctx.Err() == context.DeadlineExceeded {
		log.Printf("[vpncmd] TIMEOUT after %s", cmdTimeout)
		return output, fmt.Errorf("vpncmd timed out after %s", cmdTimeout)
	}
	if err != nil {
		log.Printf("[vpncmd] ERROR: %v\nOutput:\n%s", err, output)
		return output, fmt.Errorf("vpncmd failed: %w\n%s", err, output)
	}

	log.Printf("[vpncmd] OK (%d bytes output)", len(output))
	return output, nil
}
