package routes

import (
	"net/http"
	"text/template"

	"github.com/AirNomadSmitty/TPDK/cache"
	"github.com/AirNomadSmitty/TPDK/opto"
)

type LastCrunchController struct {
	Cache *cache.Cache
}
type lastCrunchData struct {
	Lineups     []*opto.Lineup
	Exposures   map[string][]*opto.Exposure
	TotalReward int
	CashCount   int
}

func NewLastCrunchController(cache *cache.Cache) *LastCrunchController {
	return &LastCrunchController{cache}
}

func (cont *LastCrunchController) Get(res http.ResponseWriter, req *http.Request) {
	data := &lastCrunchData{Lineups: cont.Cache.Lineups, Exposures: cont.Cache.Exposures}

	t, err := template.ParseFiles("views/index.html")
	if err != nil {
		panic(err.Error())
	}
	t.Execute(res, data)

}
