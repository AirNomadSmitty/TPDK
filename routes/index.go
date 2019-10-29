package routes

import (
	"net/http"
	"strconv"
	"text/template"

	"github.com/AirNomadSmitty/TPDK/cache"
	"github.com/AirNomadSmitty/TPDK/mappers"
	"github.com/AirNomadSmitty/TPDK/opto"
)

type IndexController struct {
	GroupMapper     *mappers.GroupMapper
	PlayerMapper    *mappers.PlayerMapper
	ThresholdMapper *mappers.ThresholdMapper
	Cache           *cache.Cache
}

func MakeIndexController(groupMapper *mappers.GroupMapper, playerMapper *mappers.PlayerMapper, thresholdMapper *mappers.ThresholdMapper, cache *cache.Cache) *IndexController {
	return &IndexController{groupMapper, playerMapper, thresholdMapper, cache}
}

type indexData struct {
	Lineups     []*opto.Lineup
	Exposures   map[string][]*opto.Exposure
	TotalReward int
	CashCount   int
}

type IterationResult struct {
	Settings  *opto.Settings
	AvgCash   int
	AvgTotal  int
	Median    int
	Highest   float64
	AvgFourth float64
}

func (cont *IndexController) Get(res http.ResponseWriter, req *http.Request) {
	param := req.URL.Query().Get("testParam")
	if param != "testTokenHere" {
		res.Write([]byte("Unauthorized"))
		return
	}

	data := &indexData{}
	slateID := 2
	// groups, err := cont.GroupMapper.GetFromUserIDAndSlateID(auth.UserID, slateID)
	var (
		groups []*opto.Group
	)

	playerCount, playersByPosition, err := cont.PlayerMapper.GetFromSlateIDByPosition(slateID)
	if err != nil {
		panic(err.Error())
	}

	settings := generateSettingsFromRequest(req)

	optomizer := opto.NewOpto(settings)
	data.Lineups, data.Exposures = optomizer.Run(playerCount, playersByPosition, groups)
	data.TotalReward, data.CashCount = cont.rewardAndTotalLineups(data.Lineups, slateID)
	data.TotalReward = data.TotalReward - 3000
	cont.Cache.Lineups = data.Lineups
	cont.Cache.Exposures = data.Exposures

	t, err := template.ParseFiles("views/index.html")
	if err != nil {
		panic(err.Error())
	}
	t.Execute(res, data)
}

func generateSettingsFromRequest(req *http.Request) *opto.Settings {
	settings := &opto.Settings{3, 2, .2}
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

func (cont *IndexController) rewardAndTotalLineups(lineups []*opto.Lineup, slateID int) (int, int) {
	thresholds, err := cont.ThresholdMapper.GetFromSlateID(slateID)

	if err != nil {
		panic(err.Error())
	}

	total := 0
	cashCount := 0
	for _, lineup := range lineups {
		reward := findRewardForLineup(lineup, thresholds)
		total += reward
		if reward > 0 {
			cashCount++
		}
		lineup.Reward = reward
	}

	return total, cashCount
}

func findRewardForLineup(lineup *opto.Lineup, thresholds []*mappers.Threshold) int {
	for _, threshold := range thresholds {
		if threshold.Score <= lineup.ActualScore {
			return threshold.Reward
		}
	}

	return 0
}
