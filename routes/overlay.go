package routes

import (
	"net/http"
	"text/template"

	"github.com/AirNomadSmitty/TPDK/cache"

	"github.com/AirNomadSmitty/TPDK/mappers"
	"github.com/AirNomadSmitty/TPDK/opto"
)

type OverlayController struct {
	PlayerMapper *mappers.PlayerMapper
	Cache        *cache.Cache
}
type exposuresOverlayData struct {
	Exposures map[string][]*opto.Exposure
}

type lineupsOverlayData struct {
	Lineups []*opto.Lineup
}

func NewOverlayController(playerMapper *mappers.PlayerMapper, cache *cache.Cache) *OverlayController {
	return &OverlayController{playerMapper, cache}
}

func (cont *OverlayController) GetExposures(res http.ResponseWriter, req *http.Request) {
	data := &exposuresOverlayData{cont.Cache.Exposures}

	t, err := template.ParseFiles("views/overlay/exposure.html")
	if err != nil {
		panic(err.Error())
	}
	t.Execute(res, data)
}

func (cont *OverlayController) GetLineups(res http.ResponseWriter, req *http.Request) {
	data := &lineupsOverlayData{cont.Cache.Lineups}

	t, err := template.ParseFiles("views/overlay/lineups.html")
	if err != nil {
		panic(err.Error())
	}
	t.Execute(res, data)

}
