package opto

import (
	"math/rand"
	"sort"
	"time"

	"github.com/jinzhu/copier"

	"github.com/draffensperger/golp"
)

const SDsalaryCap = 50000
const SDlineupSize = 6

type SDPlayer struct {
	Player      *Player
	Correlation float64
	IsCaptain   bool
}

type SDOpto struct {
	Settings              *Settings
	baseLineupConstraints []*Constraint
	variablePlayerMap     map[int]*SDPlayer
}

type SDExposure struct {
	Player  *SDPlayer
	Percent int
}

func NewSDOpto(settings *Settings) *SDOpto {
	return &SDOpto{Settings: settings}
}

type SDLineup struct {
	Score         float64
	ActualScore   float64
	RandomScore   float64
	Salary        int
	PlayerIndexes []int
	CPT           *SDPlayer
	Flex          []*SDPlayer
	Reward        int
}

func (lineup *SDLineup) MakeUniqueConstraintFromLineup() []golp.Entry {
	var uniqueConstraint []golp.Entry
	for _, index := range lineup.PlayerIndexes {
		uniqueConstraint = append(uniqueConstraint, golp.Entry{index, 1})
	}

	return uniqueConstraint
}

func (lineup *SDLineup) ForceAddPlayer(player *SDPlayer) {
	lineup.PlayerIndexes = append(lineup.PlayerIndexes, player.Player.ConstraintID)
	if player.IsCaptain {
		lineup.CPT = player
	} else {
		lineup.Flex = append(lineup.Flex, player)
	}
}

func (opto *SDOpto) Run(players []*SDPlayer) ([]*SDLineup, []*SDExposure) {
	var lineups []*SDLineup
	var lp *golp.LP
	var uniquenessConstraints [][]golp.Entry
	for i := 0; i < 52; i++ {
		lp = opto.getLpForIteration(players, uniquenessConstraints)

		lp.Solve()
		vars := lp.Variables()
		lineup := opto.makeLineupFromVars(vars)
		// Copy the lineup with this iteration's randomized projections for later
		lineupCopy := SDLineup{}
		copier.Copy(&lineupCopy, &lineup)
		lineups = append(lineups, &lineupCopy)
		uniquenessConstraints = append(uniquenessConstraints, lineup.MakeUniqueConstraintFromLineup())
	}

	sort.Slice(lineups, func(i, j int) bool {
		return lineups[i].ActualScore > lineups[j].ActualScore
	})

	exposures := opto.getExposures(lineups)
	return lineups, exposures
}

func (opto *SDOpto) getLpForIteration(players []*SDPlayer, uniquenessConstraints [][]golp.Entry) *golp.LP {
	playerCount := len(players)
	lp := opto.initializeLp(players, playerCount)
	source := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(source)
	keyRand := rand.Float64()*2*opto.Settings.Randomness - opto.Settings.Randomness
	scoreObjective := make([]float64, playerCount)

	for _, player := range players {
		player.Player.CurrentProjection = player.Player.Projection + player.Player.Projection*(keyRand*(player.Correlation+player.Correlation*rand.NormFloat64()))
		scoreObjective[player.Player.ConstraintID] = player.Player.CurrentProjection
	}

	for _, uniqueConstraint := range uniquenessConstraints {
		lp.AddConstraintSparse(uniqueConstraint, golp.LE, float64(SDlineupSize-opto.Settings.Uniques))
	}

	lp.SetObjFn(scoreObjective)
	lp.SetMaximize()

	return lp
}

func (opto *SDOpto) initializeLp(players []*SDPlayer, playerCount int) *golp.LP {
	lp := golp.NewLP(0, playerCount)

	if len(opto.baseLineupConstraints) == 0 {
		i := 0
		flexConstraint := &Constraint{nil, golp.EQ, SDlineupSize - 1, true, nil}
		captainConstraint := &Constraint{nil, golp.EQ, 1, true, nil}
		budgetConstraint := &Constraint{nil, golp.LE, salaryCap, false, nil}
		samePlayerConstraints := make(map[int64]*Constraint)
		vsDefenseConstraints := make(map[string]*Constraint)
		bothTeamsConstraints := make(map[string]*Constraint)

		opto.variablePlayerMap = make(map[int]*SDPlayer, playerCount)
		for _, player := range players {
			player.Player.ConstraintID = i
			opto.variablePlayerMap[i] = player
			budgetConstraint.Floats = append(budgetConstraint.Floats, float64(player.Player.Salary))
			if player.IsCaptain {
				captainConstraint.Entries = append(captainConstraint.Entries, golp.Entry{i, 1})
			} else {
				flexConstraint.Entries = append(flexConstraint.Entries, golp.Entry{i, 1})
			}
			if samePlayerConstraints[player.Player.ID] == nil {
				constraint := &Constraint{nil, golp.LE, 1, true, nil}
				samePlayerConstraints[player.Player.ID] = constraint
			}
			samePlayerConstraints[player.Player.ID].Entries = append(samePlayerConstraints[player.Player.ID].Entries, golp.Entry{player.Player.ConstraintID, 1})

			/** VS DEFENSE **/

			if player.Player.Position == "DST" {
				if vsDefenseConstraints[player.Player.Team] == nil {
					vsDefenseConstraints[player.Player.Team] = &Constraint{nil, golp.LE, SDlineupSize + 1, true, nil}
				}
				vsDefenseConstraints[player.Player.Team].Entries = append(vsDefenseConstraints[player.Player.Team].Entries, golp.Entry{player.Player.ConstraintID, SDlineupSize})
			} else {
				if vsDefenseConstraints[player.Player.Opp] == nil {
					vsDefenseConstraints[player.Player.Opp] = &Constraint{nil, golp.LE, SDlineupSize + 1, true, nil}
				}
				vsDefenseConstraints[player.Player.Opp].Entries = append(vsDefenseConstraints[player.Player.Opp].Entries, golp.Entry{player.Player.ConstraintID, 1})
			}

			/** both teams **/

			if bothTeamsConstraints[player.Player.Team] == nil {
				bothTeamsConstraints[player.Player.Team] = &Constraint{nil, golp.GE, 1, true, nil}
			}
			bothTeamsConstraints[player.Player.Team].Entries = append(bothTeamsConstraints[player.Player.Team].Entries, golp.Entry{player.Player.ConstraintID, 1})

			i++
		}
		opto.baseLineupConstraints = append(opto.baseLineupConstraints, budgetConstraint)
		opto.baseLineupConstraints = append(opto.baseLineupConstraints, flexConstraint)
		opto.baseLineupConstraints = append(opto.baseLineupConstraints, captainConstraint)
		for _, constraint := range samePlayerConstraints {
			opto.baseLineupConstraints = append(opto.baseLineupConstraints, constraint)
		}
		for _, vsDconstraint := range vsDefenseConstraints {
			opto.baseLineupConstraints = append(opto.baseLineupConstraints, vsDconstraint)
		}
		for _, bothTeamsConstraint := range bothTeamsConstraints {
			opto.baseLineupConstraints = append(opto.baseLineupConstraints, bothTeamsConstraint)
		}

	}

	for _, constraint := range opto.baseLineupConstraints {
		if constraint.isSparse {
			lp.AddConstraintSparse(constraint.Entries, constraint.Type, constraint.RightHand)
		} else {
			lp.AddConstraint(constraint.Floats, constraint.Type, constraint.RightHand)
		}
	}
	for i := 0; i < playerCount; i++ {
		lp.SetBinary(i, true)
	}
	return lp
}

func (opto *SDOpto) getExposures(lineups []*SDLineup) []*SDExposure {
	exposures := make(map[int]int)
	for _, lineup := range lineups {
		for _, id := range lineup.PlayerIndexes {
			exposures[id]++
		}
	}

	var exposureSlice []*SDExposure
	totalLines := len(lineups)
	for id, count := range exposures {
		exposureSlice = append(exposureSlice, &SDExposure{opto.variablePlayerMap[id], count * 100 / totalLines})
	}
	sort.Slice(exposureSlice, func(i, j int) bool {
		return exposureSlice[i].Percent > exposureSlice[j].Percent
	})

	return exposureSlice
}

func (opto *SDOpto) makeLineupFromVars(vars []float64) *SDLineup {
	lineup := &SDLineup{}
	for index, val := range vars {
		if val == 1 {
			player := opto.variablePlayerMap[index]
			lineup.Salary += player.Player.Salary
			lineup.ActualScore += player.Player.Score
			lineup.Score += player.Player.Projection
			lineup.RandomScore += player.Player.CurrentProjection
			lineup.ForceAddPlayer(player)
		}
	}
	return lineup
}
