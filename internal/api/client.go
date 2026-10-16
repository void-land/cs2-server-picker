package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/dom1torii/yet-another-server-picker/internal/config"
	"github.com/dom1torii/yet-another-server-picker/internal/presets"
)

type Response struct {
	Success bool           `json:"success"`
	Pops    map[string]Pop `json:"pops"`
	// relay ips of pops hidden from the list, still need to be blocked
	HiddenIps []string `json:"-"`
}

type Pop struct {
	Key    string
	Desc   string  `json:"desc"`
	Relays []Relay `json:"relays"`
}

type Relay struct {
	Ipv4      string `json:"ipv4"`
	PortRange [2]int `json:"port_range"`
}

// plain http on purpose: https to this endpoint hangs on some networks
const sdrConfigUrl = "http://api.steampowered.com/ISteamApps/GetSDRConfig/v1?appid=730"

func newClient(cfg *config.Config) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	// sudo usually strips proxy env vars, so allow setting one in the config
	if cfg.Network.Proxy != "" {
		proxyUrl, err := url.Parse(cfg.Network.Proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy %q: %w", cfg.Network.Proxy, err)
		}
		transport.Proxy = http.ProxyURL(proxyUrl)
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}

	return &http.Client{
		Timeout:   15 * time.Second,
		Transport: transport,
	}, nil
}

func FetchRelays(cfg *config.Config) (Response, error) {
	log.Println("Fetching API...")

	client, err := newClient(cfg)
	if err != nil {
		return Response{}, err
	}

	resp, err := client.Get(sdrConfigUrl)
	if err != nil {
		return Response{}, fmt.Errorf("fetching relays: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Println("Error closing body: ", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return Response{}, fmt.Errorf("fetching relays: unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, fmt.Errorf("reading relays: %w", err)
	}

	log.Println("Success fetching API. Status:", resp.Status)

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return Response{}, fmt.Errorf("parsing relays: %w", err)
	}
	if !response.Success || len(response.Pops) == 0 {
		return Response{}, fmt.Errorf("api returned no relays")
	}

	response.Pops, response.HiddenIps = filterPops(response.Pops, cfg)

	return response, nil
}

// skip useless pops that have no relays
// also skip perfect world pops if they are disabled, but return their ips
// so they still get blocked
func filterPops(pops map[string]Pop, cfg *config.Config) (map[string]Pop, []string) {
	filteredPops := make(map[string]Pop)
	var hiddenIps []string
	pwPops := presets.Presets["cn-pw"].Pops
	for key, pop := range pops {
		pop.Key = key
		_, isPW := pwPops[key]
		if len(pop.Relays) > 0 {
			if cfg.Relays.ShowPW || !isPW {
				filteredPops[key] = pop
			} else {
				for _, relay := range pop.Relays {
					if relay.Ipv4 != "" {
						hiddenIps = append(hiddenIps, relay.Ipv4)
					}
				}
			}
		}
	}
	return filteredPops, hiddenIps
}
