package opto

import (
	"math/rand"
	"sort"
	"time"

	"github.com/jinzhu/copier"

	"github.com/draffensperger/golp"
)

const salaryCap = 50000
const lineupSize = 9
const ALL = "ALL"
const QB = "QB"
const RB = "RB"
const WR = "WR"
const TE = "TE"
const FLEX = "FLEX"
const DST = "DST"

var positionMaxes = map[string]float64{
	QB:   1,
	RB:   3,
	WR:   4,
	TE:   2,
	DST:  1,
	FLEX: 7,
}
var positionMins = map[string]float64{
	QB:   1,
	RB:   2,
	WR:   3,
	TE:   1,
	DST:  1,
	FLEX: 7,
}

type Opto struct {
	Settings              *Settings
	baseLineupConstraints []*Constraint
	variablePlayerMap     map[int]*Player
}

func NewOpto(settings *Settings) *Opto {
	return &Opto{Settings: settings}
}

type Settings struct {
	Uniques    int
	GroupSize  int
	Randomness float64
}

type Player struct {
	Name              string
	Position          string
	Salary            int
	Team              string
	Projection        float64
	ConstraintID      int
	Score             float64
	CurrentProjection float64
	ID                int64
	Opp               string
}

type Lineup struct {
	Score         float64
	ActualScore   float64
	Salary        int
	PlayerIndexes []int
	WRs           []*Player
	RBs           []*Player
	TE            *Player
	QB            *Player
	Flex          *Player
	DST           *Player
	Reward        int
}

type Group struct {
	Name string
	ID   int64
	Key  *Player
	List []*Player
	Size int
}

type Exposure struct {
	Player  *Player
	Percent int
}

type Constraint struct {
	Entries   []golp.Entry
	Type      golp.ConstraintType
	RightHand float64
	isSparse  bool
	Floats    []float64
}

func (lineup *Lineup) ForceAddPlayer(player *Player) {
	lineup.PlayerIndexes = append(lineup.PlayerIndexes, player.ConstraintID)
	switch player.Position {
	case QB:
		lineup.QB = player
		break
	case RB:
		if len(lineup.RBs) == 2 {
			lineup.Flex = player
		} else {
			lineup.RBs = append(lineup.RBs, player)
		}
		break
	case WR:
		if len(lineup.WRs) == 3 {
			lineup.Flex = player
		} else {
			lineup.WRs = append(lineup.WRs, player)
		}
		break
	case TE:
		if lineup.TE != nil {
			lineup.Flex = player
		} else {
			lineup.TE = player
		}
		break
	case DST:
		lineup.DST = player
	}
}

func (opto *Opto) Run(playerCount int, playersByPosition map[string][]*Player, groups []*Group) ([]*Lineup, map[string][]*Exposure) {
	var lineups []*Lineup
	var lp *golp.LP
	var uniquenessConstraints [][]golp.Entry
	for i := 0; i < 150; i++ {
		lp = opto.getLpForIteration(playersByPosition, playerCount, groups, uniquenessConstraints)

		lp.Solve()
		vars := lp.Variables()
		lineup := opto.makeLineupFromVars(vars)

		// Copy the lineup with this iteration's randomized projections for later
		lineupCopy := Lineup{}
		copier.Copy(&lineupCopy, &lineup)
		lineups = append(lineups, &lineupCopy)
		uniquenessConstraints = append(uniquenessConstraints, makeUniqueConstraintFromLineup(lineup))
	}

	sort.Slice(lineups, func(i, j int) bool {
		return lineups[i].ActualScore > lineups[j].ActualScore
	})

	exposures := opto.getExposures(lineups)
	return lineups, exposures
}

func (opto *Opto) getLpForIteration(playersByPosition map[string][]*Player, playerCount int, groups []*Group, uniquenessConstraints [][]golp.Entry) *golp.LP {
	lp := opto.initializeLp(playersByPosition, playerCount, groups, opto.Settings.GroupSize)
	source := rand.NewSource(time.Now().UnixNano())
	rand := rand.New(source)

	scoreObjective := make([]float64, playerCount)

	for _, positionPlayers := range playersByPosition {
		for _, player := range positionPlayers {
			player.CurrentProjection = player.Projection + player.Projection*(rand.Float64()*2*opto.Settings.Randomness-opto.Settings.Randomness)
			scoreObjective[player.ConstraintID] = player.CurrentProjection
		}
	}

	for _, uniqueConstraint := range uniquenessConstraints {
		lp.AddConstraintSparse(uniqueConstraint, golp.LE, float64(lineupSize-opto.Settings.Uniques))
	}

	lp.SetObjFn(scoreObjective)
	lp.SetMaximize()

	return lp
}

func (opto *Opto) initializeLp(playersByPosition map[string][]*Player, playerCount int, groups []*Group, groupSize int) *golp.LP {
	lp := golp.NewLP(0, playerCount)

	if len(opto.baseLineupConstraints) == 0 {
		i := 0
		var (
			budgetConstraint []float64
			flexConstraint   []golp.Entry
		)
		stackGroups := make(map[string]*Group)
		oppGroups := make(map[string]*Group)
		teamLimitConstraints := make(map[string]*Constraint)
		opto.variablePlayerMap = make(map[int]*Player, playerCount)
		for pos, positionPlayers := range playersByPosition {
			positionConstraint := &Constraint{}
			for _, player := range positionPlayers {
				player.ConstraintID = i
				opto.variablePlayerMap[i] = player
				budgetConstraint = append(budgetConstraint, float64(player.Salary))
				positionConstraint.Entries = append(positionConstraint.Entries, golp.Entry{i, 1})

				if teamLimitConstraints[player.Team] == nil {
					teamLimitConstraints[player.Team] = &Constraint{}
					stackGroups[player.Team] = &Group{Size: groupSize}
				}
				if oppGroups[player.Opp] == nil {
					oppGroups[player.Opp] = &Group{Size: 1}
				}
				if oppGroups[player.Team] == nil {
					oppGroups[player.Team] = &Group{Size: 1}
				}
				if pos == WR || pos == TE {
					flexConstraint = append(flexConstraint, golp.Entry{i, 1})
					teamLimitConstraints[player.Team].Entries = append(teamLimitConstraints[player.Team].Entries, golp.Entry{i, 1})
					stackGroups[player.Team].List = append(stackGroups[player.Team].List, player)
					oppGroups[player.Opp].List = append(oppGroups[player.Opp].List, player)
				} else if pos == RB {
					flexConstraint = append(flexConstraint, golp.Entry{i, 1})
					teamLimitConstraints[player.Team].Entries = append(teamLimitConstraints[player.Team].Entries, golp.Entry{i, 1})
				} else if pos == QB {
					teamLimitConstraints[player.Team].Entries = append(teamLimitConstraints[player.Team].Entries, golp.Entry{i, 0 - float64(groupSize)})
					stackGroups[player.Team].Key = player
					oppGroups[player.Team].Key = player
				}
				i++
			}
			if positionMaxes[pos] == positionMins[pos] {
				positionConstraint.Type = golp.EQ
				positionConstraint.RightHand = positionMaxes[pos]
				positionConstraint.isSparse = true
				opto.baseLineupConstraints = append(opto.baseLineupConstraints, positionConstraint)
			} else {
				positionConstraint.Type = golp.LE
				positionConstraint.RightHand = positionMaxes[pos]
				positionConstraint.isSparse = true
				minPositionConstraint := &Constraint{positionConstraint.Entries, golp.GE, positionMins[pos], true, nil}
				opto.baseLineupConstraints = append(opto.baseLineupConstraints, positionConstraint)
				opto.baseLineupConstraints = append(opto.baseLineupConstraints, minPositionConstraint)
			}
		}
		budgetConstraintObj := &Constraint{nil, golp.LE, salaryCap, false, budgetConstraint}
		// minBudgetConstraintOBj := &Constraint{nil, golp.GE, 49500, false, budgetConstraint}
		opto.baseLineupConstraints = append(opto.baseLineupConstraints, budgetConstraintObj)
		// opto.baseLineupConstraints = append(opto.baseLineupConstraints, minBudgetConstraintOBj)

		flexConstraintObj := &Constraint{flexConstraint, golp.EQ, positionMaxes[FLEX], true, nil}
		opto.baseLineupConstraints = append(opto.baseLineupConstraints, flexConstraintObj)

		if len(groups) == 0 {
			for _, group := range stackGroups {
				groups = append(groups, group)
			}
			for _, group := range oppGroups {
				groups = append(groups, group)
			}
		}
		for _, group := range groups {
			var groupConstraint []golp.Entry
			groupConstraint = append(groupConstraint, golp.Entry{group.Key.ConstraintID, float64(0 - group.Size)})
			for _, player := range group.List {
				groupConstraint = append(groupConstraint, golp.Entry{player.ConstraintID, 1})
			}
			// Normal group constraint
			groupConstraintObj := &Constraint{groupConstraint, golp.GE, 0, true, nil}
			opto.baseLineupConstraints = append(opto.baseLineupConstraints, groupConstraintObj)
		}

		// Limit 1 per team if not using QB
		for _, teamLimit := range teamLimitConstraints {
			teamLimit.RightHand = 1
			teamLimit.Type = golp.LE
			teamLimit.isSparse = true
			opto.baseLineupConstraints = append(opto.baseLineupConstraints, teamLimit)
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

func (opto *Opto) getExposures(lineups []*Lineup) map[string][]*Exposure {
	exposuresById := make(map[int]int)
	for _, lineup := range lineups {
		for _, id := range lineup.PlayerIndexes {
			exposuresById[id]++
		}
	}

	exposuresByPosition := make(map[string][]*Exposure)
	totalLines := len(lineups)
	for id, count := range exposuresById {
		player := opto.variablePlayerMap[id]
		exposuresByPosition[ALL] = append(exposuresByPosition[ALL], &Exposure{player, count * 100 / totalLines})
		exposuresByPosition[player.Position] = append(exposuresByPosition[player.Position], &Exposure{player, count * 100 / totalLines})
	}

	for _, exposureSlice := range exposuresByPosition {
		sort.Slice(exposureSlice, func(i, j int) bool {
			return exposureSlice[i].Percent > exposureSlice[j].Percent
		})

	}

	return exposuresByPosition
}

func makeUniqueConstraintFromLineup(lineup *Lineup) []golp.Entry {
	var uniqueConstraint []golp.Entry
	for _, index := range lineup.PlayerIndexes {
		uniqueConstraint = append(uniqueConstraint, golp.Entry{index, 1})
	}

	return uniqueConstraint
}

func (opto *Opto) makeLineupFromVars(vars []float64) *Lineup {
	lineup := &Lineup{}
	for index, val := range vars {
		if val == 1 {
			player := opto.variablePlayerMap[index]

			lineup.Salary += player.Salary
			lineup.ActualScore += player.Score
			lineup.Score += player.Projection
			lineup.ForceAddPlayer(player)
		}
	}
	return lineup
}
