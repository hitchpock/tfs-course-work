package robot

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"
)

const (
	rfc3339Local = "2006-01-02T15:04:05Z"
	beutify      = "Mon, 02 Jan 2006 15:04:05"
)

type NullTime sql.NullTime

func (t *NullTime) UnmarshalJSON(b []byte) error {
	if b == nil {
		t.Valid = false
	}

	t.Valid = true

	return json.Unmarshal(b, &t.Time)
}

func (t *NullTime) MarshalJSON() ([]byte, error) {
	if !t.Valid {
		return []byte(`""`), nil
	}

	sTime := t.Time.Format(rfc3339Local)

	return json.Marshal(sTime)
}

func (t *NullTime) Scan(value interface{}) error {
	t.Time, t.Valid = value.(time.Time)
	return nil
}

func (t NullTime) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}

	return t.Time, nil
}

func (t *NullTime) ViewHTML() string {
	if t.Valid {
		return t.Time.Format(beutify)
	}

	return "-"
}
