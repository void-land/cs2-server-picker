package tui

import (
	"time"

	"github.com/dom1torii/yet-another-server-picker/internal/api"
)

type firewallMsg struct{}

type relaysMsg struct {
	pops      []api.Pop
	hiddenIps []string
}

type presetsMsg []string

type isFileEmptyMsg bool

type statusMsg struct {
	ipsCount     int
	blockedCount int
	blockedMap   map[string]bool
}

type pingMsg struct {
	index    int
	duration time.Duration
}
