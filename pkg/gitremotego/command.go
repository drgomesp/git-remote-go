package gitremotego

const (
	CmdList  = "list"
	CmdPush  = "push"
	CmdFetch = "fetch"
)

var DefaultCapabilities = []string{CmdPush, CmdFetch}
