package updates

import (
	"encoding/json"
	"os"
	"path"
	"strconv"
)

type FileUpdateRecorder struct {
	Base string
}

func (up *FileUpdateRecorder) Record(upd any, toVersion int) error {
	err := os.MkdirAll(up.Base, os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.Create(path.Join(up.Base, strconv.FormatInt(int64(toVersion), 10)+".json"))
	if err != nil {
		return err
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(upd)

	return err
}
