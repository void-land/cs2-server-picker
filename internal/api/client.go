package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

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

func FetchRelays(cfg *config.Config) (Response, error) {
	url := "https://api.steampowered.com/ISteamApps/GetSDRConfig/v1?appid=730"
	log.Println("Fetching API...")

	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln("Error fetching: ", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Fatalln("Error closing body: ", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Error fetching: ", err)
	}

	log.Println("Success fetching API. Status:", resp.Status)

	var response Response
	if err := json.Unmarshal(body, &response); err != nil {
		return Response{}, err
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
