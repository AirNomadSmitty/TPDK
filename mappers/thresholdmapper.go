package mappers

import "database/sql"

type Threshold struct {
	Score  float64
	Reward int
}
type ThresholdMapper struct {
	db *sql.DB
}

func MakeThresholdMapper(db *sql.DB) *ThresholdMapper {
	return &ThresholdMapper{db}
}

func (mapper *ThresholdMapper) GetFromSlateID(slateID int) ([]*Threshold, error) {
	var thresholds []*Threshold
	sql := `select threshold, reward from slate_rewards
	Where slate_id = ?
	ORDER BY threshold DESC`

	rows, err := mapper.db.Query(sql, slateID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		threshold := &Threshold{}
		err = rows.Scan(&threshold.Score, &threshold.Reward)
		if err != nil {
			return nil, err
		}
		thresholds = append(thresholds, threshold)
	}

	return thresholds, nil
}
