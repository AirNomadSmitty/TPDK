package routes

import (
	"net/http"
	"strconv"

	"github.com/AirNomadSmitty/TPDK/mappers"
	"github.com/AirNomadSmitty/TPDK/opto"
	"github.com/AirNomadSmitty/TPDK/utils"
	"github.com/gorilla/mux"
)

type ProjectionsController struct {
	PlayerMapper           *mappers.PlayerMapper
	ProjectionChangeMapper *mappers.ProjectionChangeMapper
}
type ProjectionsData struct {
	Lineups     []*opto.Lineup
	Exposures   map[string][]*opto.Exposure
	TotalReward int
	CashCount   int
}

func NewProjectionsController(playerMapper *mappers.PlayerMapper, projectionChangeMapper *mappers.ProjectionChangeMapper) *ProjectionsController {
	return &ProjectionsController{playerMapper, projectionChangeMapper}
}

func (cont *ProjectionsController) Post(res http.ResponseWriter, req *http.Request) {
	param := req.URL.Query().Get("testParam")
	if param != "testTokenHere" {
		res.Write([]byte("Unauthorized"))
		return
	}

	vars := mux.Vars(req)
	playerID, err := strconv.Atoi(vars["playerID"])
	slateID, err := strconv.Atoi(vars["slateID"])
	changePercent, err := strconv.ParseFloat(req.FormValue("change"), 64)
	if err != nil {
		panic(err)
	}

	player, err := cont.PlayerMapper.GetFromSlateIDAndPlayerID(slateID, playerID)
	changeRaw := player.Projection * changePercent
	player.Projection = player.Projection + changeRaw
	cont.PlayerMapper.SaveToSlate(player, slateID)
	projectionChange := &mappers.ProjectionChange{PlayerID: playerID, SlateID: slateID, ChangePercent: changePercent, ChangeRaw: changeRaw, Username: req.FormValue("user")}
	cont.ProjectionChangeMapper.Save(projectionChange)
	ret := utils.JsonFormat(player)
	res.Write([]byte(ret))
}
