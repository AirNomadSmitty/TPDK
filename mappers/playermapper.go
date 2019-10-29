package mappers

import (
	"database/sql"

	"github.com/jinzhu/copier"

	"github.com/AirNomadSmitty/TPDK/opto"
)

type BasePlayer struct {
	PlayerID int
	Name     string
	Team     string
	Position string
}

type PlayerMapper struct {
	db *sql.DB
}

func MakePlayerMapper(db *sql.DB) *PlayerMapper {
	return &PlayerMapper{db}
}

func (mapper *PlayerMapper) GetFromSlateIDByPosition(slateID int) (int, map[string][]*opto.Player, error) {
	playersByPosition := make(map[string][]*opto.Player)
	sql := `select p.name, p.position, pts.salary, p.team, pts.projection, pts.actual_score, p.player_id, pts.opp
	FROM players p
	INNER JOIN players_to_slates pts ON p.player_id = pts.player_id
	WHERE pts.slate_id = ?`

	rows, err := mapper.db.Query(sql, slateID)

	if err != nil {
		return 0, nil, err
	}
	defer rows.Close()

	playerCount := 0
	for rows.Next() {
		player := &opto.Player{}
		err = rows.Scan(&player.Name, &player.Position, &player.Salary, &player.Team, &player.Projection, &player.Score, &player.ID, &player.Opp)
		if err != nil {
			return 0, nil, err
		}
		playersByPosition[player.Position] = append(playersByPosition[player.Position], player)
		playerCount++
	}

	return playerCount, playersByPosition, nil
}

func (mapper *PlayerMapper) GetFromSDSlateID(slateID int) ([]*opto.SDPlayer, error) {
	sql := `select p.name, p.position, pts.salary, p.team, pts.projection, pts.actual_score, p.player_id, pts.opp, pts.correlation
	FROM players p
	INNER JOIN players_to_slates pts ON p.player_id = pts.player_id
	WHERE pts.slate_id = ?`

	rows, err := mapper.db.Query(sql, slateID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []*opto.SDPlayer
	for rows.Next() {
		player := &opto.Player{}
		sdPlayer := &opto.SDPlayer{Player: player}
		err = rows.Scan(&sdPlayer.Player.Name, &sdPlayer.Player.Position, &sdPlayer.Player.Salary, &sdPlayer.Player.Team, &sdPlayer.Player.Projection, &sdPlayer.Player.Score, &sdPlayer.Player.ID, &sdPlayer.Player.Opp, &sdPlayer.Correlation)
		if err != nil {
			return nil, err
		}
		sdPlayer.IsCaptain = false
		players = append(players, sdPlayer)
		capSDPlayer := opto.SDPlayer{}
		copier.Copy(&capSDPlayer, &sdPlayer)
		capSDPlayer.IsCaptain = true
		capSDPlayer.Player.Salary = int(float64(capSDPlayer.Player.Salary) * 1.5)
		capSDPlayer.Player.Projection = capSDPlayer.Player.Projection * 1.5
		capSDPlayer.Player.Score = capSDPlayer.Player.Score * 1.5
		players = append(players, &capSDPlayer)
	}

	return players, nil

}

func (mapper *PlayerMapper) GetFromSlateIDAndPlayerID(slateID int, playerID int) (*opto.Player, error) {
	sql := `select p.name, p.position, pts.salary, p.team, pts.projection, pts.actual_score, p.player_id, pts.opp
	FROM players p
	INNER JOIN players_to_slates pts ON p.player_id = pts.player_id
	WHERE pts.slate_id = ?
	AND p.player_id = ?`

	player := &opto.Player{}
	err := mapper.db.QueryRow(sql, slateID, playerID).Scan(&player.Name, &player.Position, &player.Salary, &player.Team, &player.Projection, &player.Score, &player.ID, &player.Opp)

	if err != nil {
		return nil, err
	}

	return player, nil
}

func (mapper *PlayerMapper) SaveToSlate(player *opto.Player, slateID int) {
	var err error
	if player.ID != 0 {
		_, err = mapper.db.Exec(`UPDATE players
		SET
		name = ?,
		team = ?,
		position = ?
		where player_id = ?`, player.Name, player.Team, player.Position, player.Position)
	} else {
		results, err := mapper.db.Exec(`INSERT INTO players (name, team, position) VALUES (?, ?, ?)`,
			player.Name, player.Team, player.Position)
		if err != nil {
			panic(err)
		}

		player.ID, err = results.LastInsertId()

	}
	_, err = mapper.db.Exec(`INSERT INTO players_to_slates (player_id, slate_id, salary, projection, actual_score, opp) VALUES (?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE
	player_id = ?,
	slate_id = ?,
	salary = ?,
	projection = ?,
	actual_score = ?,
	opp = ?`, player.ID, slateID, player.Salary, player.Projection, player.Score, player.Opp, player.ID, slateID, player.Salary, player.Projection, player.Score, player.Opp)

	if err != nil {
		panic(err)
	}
}

func (mapper *PlayerMapper) GetAllForJson() ([]*BasePlayer, error) {
	sql := `select player_id, name, team, position from players`

	rows, err := mapper.db.Query(sql)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		players []*BasePlayer
	)
	for rows.Next() {

		player := &BasePlayer{}
		err = rows.Scan(&player.PlayerID, &player.Name, &player.Team, &player.Position)
		if err != nil {
			return nil, err
		}
		players = append(players, player)
	}

	return players, nil

}
