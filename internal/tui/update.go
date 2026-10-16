package tui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case statusMsg:
		m.start.ipsCount = msg.ipsCount
		m.start.blockedCount = msg.blockedCount
		m.start.blockedMap = msg.blockedMap
		return m, m.refreshRelays()

	case presetsMsg:
		m.presets.presetKeys = msg
		return m, nil

	case isFileEmptyMsg:
		if msg {
			m.state = stateStart
			return m, tea.Sequence(
				writeIps(m.ipsToWrite(), m.cfg),
				m.updateStatus(),
			)
		} else {
			m.state = stateConfirm
			return m, nil
		}

	case firewallMsg:
		m.start.err = msg.err
		return m, m.updateStatus()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if m.relays.startRow > m.getRows() {
			m.relays.startRow = 0
		}

	case relaysMsg:
		if msg.err != nil {
			m.relays.loadErr = msg.err
			return m, nil
		}
		m.relays.loadErr = nil

		pops := msg.pops
		sort.SliceStable(pops, func(i, j int) bool {
			keyI := getStringToSort(pops[i].Desc)
			keyJ := getStringToSort(pops[j].Desc)
			if keyI != keyJ {
				return keyI < keyJ
			}
			return strings.ToLower(pops[i].Desc) < strings.ToLower(pops[j].Desc)
		})
		m.relays.relays = pops
		m.relays.hiddenIps = msg.hiddenIps
		return m, m.refreshRelays()

	case pingMsg:
		// drop results from an older ping round
		if msg.gen != m.relays.pingGen {
			return m, nil
		}
		m.relays.pings[msg.index] = msg.duration
		return m, m.pingNextRelay()

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			m.Quitting = true
			return m, tea.Quit
		}

		switch m.state {
		case stateStart:
			return m.updateStartSelection(msg)
		case stateRelays:
			return m.updateRelaySelection(msg)
		case stateConfirm:
			return m.updateConfirmSelection(msg)
		case statePresets:
			return m.updatePresetSelection(msg)
		}
	}

	return m, nil
}
