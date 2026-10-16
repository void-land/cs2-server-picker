package tui

import (
	"time"

	"github.com/dom1torii/yet-another-server-picker/internal/api"
)

type firewallMsg struct {
	err error
}

type relaysMsg struct {
	pops      []api.Pop
	hiddenIps []string
	err       error
}

type presetsMsg []string

type isFileEmptyMsg bool

type statusMsg struct {
	ipsCount     int
	blockedCount int
	blockedMap   map[string]bool
}

type pingMsg struct {
	gen      int
	index    int
	duration time.Duration
}
