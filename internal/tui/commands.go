package tui

import (
	"log"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dom1torii/yet-another-server-picker/internal/api"
	"github.com/dom1torii/yet-another-server-picker/internal/config"
	"github.com/dom1torii/yet-another-server-picker/internal/fs"
	"github.com/dom1torii/yet-another-server-picker/internal/ips"
	"github.com/dom1torii/yet-another-server-picker/internal/platform/firewall"
	"github.com/dom1torii/yet-another-server-picker/internal/presets"
)

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		m.getRelays(),
		loadPresets(),
		m.updateStatus(),
		tea.SetWindowTitle("Yet Another Server Picker"),
	)
}

func (m *model) getRelays() tea.Cmd {
	return func() tea.Msg {
		resp, err := api.FetchRelays(m.cfg)
		if err != nil {
			log.Println("Error fetching relays: ", err)
			return relaysMsg{err: err}
		}

		var pops []api.Pop
		for _, pop := range resp.Pops {
			pops = append(pops, pop)
		}
		return relaysMsg{pops: pops, hiddenIps: resp.HiddenIps}
	}
}

const maxConcurrentPings = 20

func getPingToIp(gen int, index int, ip string, isBlocked bool) tea.Cmd {
	return func() tea.Msg {
		if isBlocked || ip == "" {
			return pingMsg{gen: gen, index: index, duration: -1}
		}
		duration := ips.GetPing(ip)
		return pingMsg{gen: gen, index: index, duration: duration}
	}
}

// starts pinging the next relay in line, if there is one left
func (m *model) pingNextRelay() tea.Cmd {
	if m.relays.pingNext >= len(m.relays.relays) {
		return nil
	}
	i := m.relays.pingNext
	m.relays.pingNext++

	var ip string
	if len(m.relays.relays[i].Relays) > 0 {
		ip = m.relays.relays[i].Relays[0].Ipv4
	}
	return getPingToIp(m.relays.pingGen, i, ip, m.start.blockedMap[ip])
}

// restarts pinging from scratch. every finished ping starts the next one,
// so at most maxConcurrentPings run at once. pings from an older round
// are ignored by their generation
func (m *model) refreshRelays() tea.Cmd {
	m.relays.pingGen++
	m.relays.pings = make(map[int]time.Duration)
	m.relays.pingNext = 0

	var cmds []tea.Cmd
	for range min(maxConcurrentPings, len(m.relays.relays)) {
		cmds = append(cmds, m.pingNextRelay())
	}
	return tea.Batch(cmds...)
}

func blockIps(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		err := firewall.BlockIps(cfg, nil)
		if err != nil {
			log.Println("Error blocking ips: ", err)
		}
		return firewallMsg{err: err}
	}
}

func unBlockIps() tea.Cmd {
	return func() tea.Msg {
		firewall.UnBlockIps(nil)
		return firewallMsg{}
	}
}

func isFileEmpty(path string) tea.Cmd {
	return func() tea.Msg {
		empty := fs.IsFileEmpty(path)
		return isFileEmptyMsg(empty)
	}
}

func loadPresets() tea.Cmd {
	return func() tea.Msg {
		keys := make([]string, 0, len(presets.Presets))
		for k := range presets.Presets {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return presetsMsg(keys)
	}
}

func (m *model) updateStatus() tea.Cmd {
	return func() tea.Msg {
		ipsCount := fs.GetFileLineCount(m.cfg.Ips.Path)
		blockedMap := firewall.GetBlockedIps()
		return statusMsg{
			ipsCount:     ipsCount,
			blockedCount: len(blockedMap),
			blockedMap:   blockedMap,
		}
	}
}

func writeIps(ipList []string, cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		ips.WriteIpsToFile(ipList, cfg)
		return firewallMsg{}
	}
}
