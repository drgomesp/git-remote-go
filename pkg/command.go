package gitremote

const (
	CmdList  = "list"
	CmdPush  = "push"
	CmdFetch = "fetch"
)

var DefaultCapabilities = []string{CmdList, CmdPush, CmdFetch}
