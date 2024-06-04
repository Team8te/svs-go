package endpoint

import (
	"encoding/json"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type createRoomBody struct {
	Name string `json:name`
}

// http://127.0.0.1:8090/room/create?room=ROOM_NAME
func (e *Endpoint) createRoomHandler(w http.ResponseWriter, req *http.Request) *Response {
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}

	defer req.Body.Close()

	body := createRoomBody{}

	err := json.NewDecoder(req.Body).Decode(&body)
	if err != nil {
		res.Status = 400
		res.Data = "invalud body"
		return res
	}

	room, err := e.rs.CreateRoom(req.Context(), body.Name)
	if err != nil {
		res.Data = err.Error()
		res.Status = 400
	}
	res.Data = room.ID.ToString()
	log.Debugf("[KEY] name: %s, id: %v", room.Name, room.ID)

	return res
}
