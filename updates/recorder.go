package updates

type Recorder interface {
	Record(upd any, toVersion int) error
}
