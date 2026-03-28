package cli

import (
	"io"

	"github.com/maximerivest/mcp2cli/internal/elicitation"
	mcpclient "github.com/maximerivest/mcp2cli/internal/mcp/client"
)

func terminalConnectOptions(in io.Reader, errOut io.Writer) mcpclient.ConnectOptions {
	handler := elicitation.NewHandler(in, errOut)
	return mcpclient.ConnectOptions{ElicitationHandler: handler.Handle}
}
