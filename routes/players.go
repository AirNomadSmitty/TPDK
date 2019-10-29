package routes

import (
	"net/http"

	"github.com/AirNomadSmitty/TPDK/mappers"
	"github.com/AirNomadSmitty/TPDK/opto"
	"github.com/AirNomadSmitty/TPDK/utils"
)

type PlayersController struct {
	PlayerMapper *mappers.PlayerMapper
}
type PlayersData struct {
	Lineups     []*opto.Lineup
	Exposures   map[string][]*opto.Exposure
	TotalReward int
	CashCount   int
}

func NewPlayersController(playerMapper *mappers.PlayerMapper) *PlayersController {
	return &PlayersController{playerMapper}
}

func (cont *PlayersController) Get(res http.ResponseWriter, req *http.Request) {
	players, _ := cont.PlayerMapper.GetAllForJson()

	ret := utils.JsonFormat(players)
	res.Write([]byte(ret))
}
