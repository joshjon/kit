package paginate

import (
	"strconv"

	"github.com/joshjon/kit/id"
)

func Int64CursorParser() CursorParserFunc[int64] {
	return func(rawCursor string) (*int64, error) {
		num, err := strconv.ParseInt(rawCursor, 10, 64)
		if err != nil {
			return nil, err
		}
		return &num, nil
	}
}

func IDCursorParser[ID id.ID, PT id.SubtypePtr[ID]]() CursorParserFunc[ID] {
	return func(rawCursor string) (*ID, error) {
		entityID, err := id.Parse[ID, PT](rawCursor)
		if err != nil {
			return nil, err
		}
		return &entityID, nil
	}
}
