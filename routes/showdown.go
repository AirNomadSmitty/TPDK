package routes

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/AirNomadSmitty/TPDK/mappers"
	"github.com/AirNomadSmitty/TPDK/opto"
	"github.com/AirNomadSmitty/TPDK/utils"
)

type ShowdownController struct {
	PlayerMapper    *mappers.PlayerMapper
	ThresholdMapper *mappers.ThresholdMapper
}

func MakeShowdownController(playerMapper *mappers.PlayerMapper, thresholdMapper *mappers.ThresholdMapper) *ShowdownController {
	return &ShowdownController{playerMapper, thresholdMapper}
}

type showdownData struct {
	Lineups     []*opto.SDLineup
	Exposures   []*opto.SDExposure
	TotalReward int
	CashCount   int
}

func (cont *ShowdownController) Get(res http.ResponseWriter, req *http.Request, auth *utils.Auth) {
	if !auth.IsLoggedIn() {
		http.Redirect(res, req, "/login", http.StatusSeeOther)
		return
	}
	data := &showdownData{}
	slateID, err := strconv.Atoi(req.URL.Query().Get("slate"))
	if err != nil {
		panic(err.Error())
	}
	players, err := cont.PlayerMapper.GetFromSDSlateID(slateID)

	settings := cont.generateSettingsFromRequest(req)

	optomizer := opto.NewSDOpto(settings)
	data.Lineups, data.Exposures = optomizer.Run(players)
	data.TotalReward, data.CashCount = cont.rewardAndTotalLineups(data.Lineups, slateID)
	data.TotalReward = data.TotalReward - 416

	t, err := template.ParseFiles("views/showdown.html")
	if err != nil {
		panic(err.Error())
	}
	t.Execute(res, data)
}

func (cont *ShowdownController) rewardAndTotalLineups(lineups []*opto.SDLineup, slateID int) (int, int) {
	thresholds, err := cont.ThresholdMapper.GetFromSlateID(slateID)

	if err != nil {
		panic(err.Error())
	}

	total := 0
	cashCount := 0
	for _, lineup := range lineups {
		reward := findRewardForSDLineup(lineup, thresholds)
		total += reward
		if reward > 0 {
			cashCount++
		}
		lineup.Reward = reward
	}

	return total, cashCount
}

func findRewardForSDLineup(lineup *opto.SDLineup, thresholds []*mappers.Threshold) int {
	for _, threshold := range thresholds {
		if threshold.Score <= lineup.ActualScore {
			return threshold.Reward
		}
	}

	return 0
}

func (cont *ShowdownController) generateSettingsFromRequest(req *http.Request) *opto.Settings {
	settings := &opto.Settings{Uniques: 2, Randomness: 1}
	randomness := req.URL.Query().Get("randomness")
	var err error
	if randomness != "" {
		settings.Randomness, err = strconv.ParseFloat(randomness, 64)
	}
	uniques := req.URL.Query().Get("uniques")
	if uniques != "" {
		settings.Uniques, err = strconv.Atoi(uniques)
	}
	groupsize := req.URL.Query().Get("groupsize")
	if groupsize != "" {
		settings.GroupSize, err = strconv.Atoi(groupsize)
	}

	if err != nil {
		panic(err)
	}
	return settings
}
