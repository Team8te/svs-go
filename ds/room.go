package ds

// RoomID ...
type RoomID string

func (r *RoomID) ToString() string {
	if r != nil {
		return string(*r)
	}
	return ""
}

// Room ...
type Room struct {
	ID       RoomID
	Name     string
	StreamID int64
}
