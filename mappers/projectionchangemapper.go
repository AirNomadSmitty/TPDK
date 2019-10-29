package mappers

import "database/sql"

type ProjectionChange struct {
	PlayerID      int
	SlateID       int
	ChangePercent float64
	ChangeRaw     float64
	Time          int
	Username      string
}

type ProjectionChangeMapper struct {
	db *sql.DB
}

func NewProjectionChangeMapper(db *sql.DB) *ProjectionChangeMapper {
	return &ProjectionChangeMapper{db}
}

func (mapper *ProjectionChangeMapper) Save(projectionChange *ProjectionChange) {
	_, err := mapper.db.Exec("INSERT INTO projection_changes (`player_id`, `slate_id`, `change_percent`, `change_raw`, `username`) VALUES (?, ?, ?, ?, ?)",
		projectionChange.PlayerID, projectionChange.SlateID, projectionChange.ChangePercent, projectionChange.ChangeRaw, projectionChange.Username)
	if err != nil {
		panic(err)
	}
}

func (mapper *ProjectionChangeMapper) GetRecent() {
	
}
