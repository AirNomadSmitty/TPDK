package mappers

import (
	"database/sql"

	"github.com/AirNomadSmitty/TPDK/opto"
)

type GroupMapper struct {
	db *sql.DB
}

func MakeGroupMapper(db *sql.DB) *GroupMapper {
	return &GroupMapper{db}
}

func (mapper *GroupMapper) GetFromUserIDAndSlateID(userID int64, slateID int) ([]*opto.Group, error) {
	groups := make(map[int64]*opto.Group)
	sql := `select g.title, g.group_id, g.key_player_id, p.position, pts.salary, p.name, p.team, pts.projection, pts.actual_score, pts.opp from users_to_groups utg
	INNER JOIN groups g ON utg.group_id = g.group_id
	INNER JOIN players p ON p.player_id = g.key_player_id
	INNER JOIN players_to_slates pts ON p.player_id = pts.player_id
	WHERE utg.user_id = ? AND g.slate_id = ?`
	rows, err := mapper.db.Query(sql, userID, slateID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Initialize groups with Key Players
	for rows.Next() {
		group := &opto.Group{}
		group.Key = &opto.Player{}
		err = rows.Scan(&group.Name, &group.ID, &group.Key.ID, &group.Key.Position, &group.Key.Salary, &group.Key.Name, &group.Key.Team, &group.Key.Projection, &group.Key.Score, &group.Key.Opp)
		if err != nil {
			return nil, err
		}
		groups[group.ID] = group
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	// Add in regular group members
	sql = `select g.group_id, p.player_id, p.position, pts.salary, p.name, p.team, pts.projection, pts.actual_score, pts.opp from users_to_groups utg
	INNER JOIN groups g ON utg.group_id = g.group_id
	INNER JOIN players_to_groups ptg ON ptg.group_id = g.group_id
	INNER JOIN players p ON p.player_id = ptg.player_id
	INNER JOIN players_to_slates pts ON p.player_id = pts.player_id
	WHERE utg.user_id = ? AND g.slate_id = ?`
	rows, err = mapper.db.Query(sql, userID, slateID)
	if err != nil {
		panic(err)
		return nil, err
	}
	var groupID int64
	for rows.Next() {
		player := &opto.Player{}
		err = rows.Scan(&groupID, &player.ID, &player.Position, &player.Salary, &player.Name, &player.Team, &player.Projection, &player.Score, &player.Opp)
		if err != nil {
			return nil, err
		}
		if foundGroup, ok := groups[groupID]; ok {
			group := foundGroup
			group.List = append(group.List, player)
		} else {
			panic("No group found")
		}
	}

	var groupSlice []*opto.Group
	for _, group := range groups {
		groupSlice = append(groupSlice, group)
	}

	return groupSlice, nil
}
