package cache

import (
	"github.com/AirNomadSmitty/TPDK/opto"
)

type Cache struct {
	Lineups   []*opto.Lineup
	Exposures map[string][]*opto.Exposure
}
