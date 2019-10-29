package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/AirNomadSmitty/TPDK/cache"
	"github.com/AirNomadSmitty/TPDK/mappers"
	"github.com/AirNomadSmitty/TPDK/opto"
	"github.com/AirNomadSmitty/TPDK/routes"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

func main() {
	db, err := sql.Open("mysql", "root:@/optomizr")
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}

	// parseFileSD("4NEvNYG.csv", 5, db)
	r := mux.NewRouter()
	userMapper := mappers.MakeUserMapper(db)
	groupMapper := mappers.MakeGroupMapper(db)
	playerMapper := mappers.MakePlayerMapper(db)
	thresholdMapper := mappers.MakeThresholdMapper(db)
	projectionChangeMapper := mappers.NewProjectionChangeMapper(db)
	cache := &cache.Cache{}

	index := routes.MakeIndexController(groupMapper, playerMapper, thresholdMapper, cache)
	login := routes.MakeLoginController(userMapper)
	lastCrunch := routes.NewLastCrunchController(cache)
	projections := routes.NewProjectionsController(playerMapper, projectionChangeMapper)
	players := routes.NewPlayersController(playerMapper)
	overlay := routes.NewOverlayController(playerMapper, cache)

	r.HandleFunc("/", lastCrunch.Get).Methods("GET")
	// r.HandleFunc("/showdown", NewAuthenticatedWrapper(showdown.Get).ServeHTTP).Methods("GET")
	r.HandleFunc("/login", login.Get).Methods("GET")
	r.HandleFunc("/login", login.Post).Methods("POST")
	r.HandleFunc("/crunch", index.Get).Methods("GET")
	r.HandleFunc("/projections/{slateID:[0-9]+}/{playerID:[0-9]+}", projections.Post).Methods("POST")
	r.HandleFunc("/players", players.Get).Methods("GET")
	r.HandleFunc("/overlay/exposures", overlay.GetExposures).Methods("GET")
	r.HandleFunc("/overlay/lineups", overlay.GetLineups).Methods("GET")
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.Handle("/", r)
	http.ListenAndServe(":8080", nil)
}

func parseFileSD(filename string, slateID int64, db *sql.DB) []*opto.SDPlayer {
	csvFile, _ := os.Open(filename)
	reader := csv.NewReader(bufio.NewReader(csvFile))

	var players []*opto.SDPlayer
	i := 0
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		/*
			0 = name
			1 = projection
			2 = position
			3 = team
			4 = salary
			5 = val
			6 = inGroup
			7 = score
			8 = opp
			9 = correlation
		*/
		salary, _ := strconv.Atoi(line[4])
		projection, _ := strconv.ParseFloat(line[1], 64)
		score, _ := strconv.ParseFloat(line[7], 64)
		correlation, _ := strconv.ParseFloat(line[9], 64)

		player := &opto.Player{Name: line[0], Position: line[2], Salary: salary, Team: line[3], Projection: projection, Score: score, Opp: line[8]}
		sdPlayer := &opto.SDPlayer{Player: player, Correlation: correlation}
		players = append(players, sdPlayer)
		i++
	}

	var playerID int64
	for _, player := range players {
		err := db.QueryRow("SELECT player_id FROM players WHERE name=?", player.Player.Name).Scan(&playerID)
		if err == sql.ErrNoRows {
			result, err := db.Exec("INSERT INTO players (name, team, position) VALUES (?, ?, ?)", player.Player.Name, player.Player.Team, player.Player.Position)
			playerID, err = result.LastInsertId()
			if err != nil {
				panic(err)
			}
		}
		player.Player.ID = playerID
		fmt.Println(slateID)
		_, err = db.Exec("INSERT INTO players_to_slates (player_id, slate_id, salary, projection, actual_score, opp, correlation) VALUES (?, ?, ?, ?, ?, ?, ?)", player.Player.ID, slateID, player.Player.Salary, player.Player.Projection, player.Player.Score, player.Player.Opp, player.Correlation)
		if err != nil {
			panic(err)
		}
	}
	return players
}

func parseFile(filename string, slateID int64, db *sql.DB) (map[string][]*opto.Player, int, map[string]*opto.Group) {
	csvFile, _ := os.Open(filename)
	reader := csv.NewReader(bufio.NewReader(csvFile))

	positionPlayers := make(map[string][]*opto.Player)
	groups := make(map[string]*opto.Group)
	i := 0
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}
		/*
			0 = name
			1 = projection
			2 = position
			3 = team
			4 = salary
			5 = val
			6 = inGroup
			7 = score
			8 = opp
			9 = correlation
		*/
		position := line[2]
		salary, _ := strconv.Atoi(line[4])
		projection, _ := strconv.ParseFloat(line[1], 64)
		score, _ := strconv.ParseFloat(line[7], 64)
		groupable := line[6] == "1"
		if projection < 5 {
			continue
		}
		player := &opto.Player{Name: line[0], Position: line[2], Salary: salary, Team: line[3], Projection: projection, Score: score, Opp: line[8]}
		positionPlayers[position] = append(positionPlayers[position], player)
		if groupable {
			if groups[line[3]] == nil {
				groups[line[3]] = &opto.Group{}
			}
			if player.Position == "QB" {
				groups[line[3]].Key = player
			} else {
				groups[line[3]].List = append(groups[line[3]].List, player)
			}

		}
		i++
	}

	var playerID int64
	for _, players := range positionPlayers {
		for _, player := range players {
			err := db.QueryRow("SELECT player_id FROM players WHERE name=?", player.Name).Scan(&playerID)
			if err == sql.ErrNoRows {
				result, err := db.Exec("INSERT INTO players (name, team, position) VALUES (?, ?, ?)", player.Name, player.Team, player.Position)
				playerID, err = result.LastInsertId()
				if err != nil {
					panic(err)
				}
			}
			player.ID = playerID
			_, err = db.Exec("INSERT INTO players_to_slates (player_id, slate_id, salary, projection, actual_score, opp) VALUES (?, ?, ?, ?, ?, ?)", player.ID, slateID, player.Salary, player.Projection, player.Score, player.Opp)
			if err != nil {
				panic(err)
			}
		}
	}
	var groupID int64
	for _, group := range groups {
		result, _ := db.Exec("INSERT INTO groups (title, slate_id, key_player_id) VALUES (?,?,?)", group.Key.Name, slateID, group.Key.ID)
		groupID, _ = result.LastInsertId()
		result, err := db.Exec("INSERT INTO users_to_groups (user_id, group_id) VALUES (2, ?)", groupID)
		if err != nil {
			panic(err)
		}
		for _, player := range group.List {
			db.Exec("INSERT INTO players_to_groups (player_id, group_id) VALUES (?,?)", player.ID, groupID)
		}
	}
	return positionPlayers, i, groups
}
