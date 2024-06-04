package endpoint

import (
	"net/http"

	"github.com/Team8te/svs-go/protocol/rtmp"
)

type streams struct {
	Publishers []stream `json:"publishers"`
	Players    []stream `json:"players"`
}

type stream struct {
	Key             string `json:"key"`
	Url             string `json:"url"`
	StreamId        uint32 `json:"stream_id"`
	VideoTotalBytes uint64 `json:"video_total_bytes"`
	VideoSpeed      uint64 `json:"video_speed"`
	AudioTotalBytes uint64 `json:"audio_total_bytes"`
	AudioSpeed      uint64 `json:"audio_speed"`
}

// http://127.0.0.1:8090/stat/livestat
func (server *Endpoint) getLiveStaticsHandler(w http.ResponseWriter, req *http.Request) *Response {
	res := &Response{
		w:      w,
		Data:   nil,
		Status: 200,
	}

	room := ""

	if err := req.ParseForm(); err == nil {
		room = req.Form.Get("room")
	}

	rtmpStream := server.handler.(*rtmp.RtmpStream)
	if rtmpStream == nil {
		res.Status = 500
		res.Data = "Get rtmp stream information error"
		return res
	}

	msgs := new(streams)

	if room == "" {
		rtmpStream.GetStreams().Range(func(key, val interface{}) bool {
			if s, ok := val.(*rtmp.Stream); ok {
				if s.GetReader() != nil {
					switch s.GetReader().(type) {
					case *rtmp.VirReader:
						v := s.GetReader().(*rtmp.VirReader)
						msg := stream{key.(string), v.Info().URL, v.ReadBWInfo.StreamId, v.ReadBWInfo.VideoDatainBytes, v.ReadBWInfo.VideoSpeedInBytesperMS,
							v.ReadBWInfo.AudioDatainBytes, v.ReadBWInfo.AudioSpeedInBytesperMS}
						msgs.Publishers = append(msgs.Publishers, msg)
					}
				}
			}
			return true
		})

		rtmpStream.GetStreams().Range(func(key, val interface{}) bool {
			ws := val.(*rtmp.Stream).GetWs()
			ws.Range(func(k, v interface{}) bool {
				if pw, ok := v.(*rtmp.PackWriterCloser); ok {
					if pw.GetWriter() != nil {
						switch pw.GetWriter().(type) {
						case *rtmp.VirWriter:
							v := pw.GetWriter().(*rtmp.VirWriter)
							msg := stream{key.(string), v.Info().URL, v.WriteBWInfo.StreamId, v.WriteBWInfo.VideoDatainBytes, v.WriteBWInfo.VideoSpeedInBytesperMS,
								v.WriteBWInfo.AudioDatainBytes, v.WriteBWInfo.AudioSpeedInBytesperMS}
							msgs.Players = append(msgs.Players, msg)
						}
					}
				}
				return true
			})
			return true
		})
	} else {
		// Warning: The room should be in the "live/stream" format!
		roomInfo, exists := (rtmpStream.GetStreams()).Load(room)
		if exists == false {
			res.Status = 404
			res.Data = "room not found or inactive"
			return res
		}

		if s, ok := roomInfo.(*rtmp.Stream); ok {
			if s.GetReader() != nil {
				switch s.GetReader().(type) {
				case *rtmp.VirReader:
					v := s.GetReader().(*rtmp.VirReader)
					msg := stream{room, v.Info().URL, v.ReadBWInfo.StreamId, v.ReadBWInfo.VideoDatainBytes, v.ReadBWInfo.VideoSpeedInBytesperMS,
						v.ReadBWInfo.AudioDatainBytes, v.ReadBWInfo.AudioSpeedInBytesperMS}
					msgs.Publishers = append(msgs.Publishers, msg)
				}
			}

			s.GetWs().Range(func(k, v interface{}) bool {
				if pw, ok := v.(*rtmp.PackWriterCloser); ok {
					if pw.GetWriter() != nil {
						switch pw.GetWriter().(type) {
						case *rtmp.VirWriter:
							v := pw.GetWriter().(*rtmp.VirWriter)
							msg := stream{room, v.Info().URL, v.WriteBWInfo.StreamId, v.WriteBWInfo.VideoDatainBytes, v.WriteBWInfo.VideoSpeedInBytesperMS,
								v.WriteBWInfo.AudioDatainBytes, v.WriteBWInfo.AudioSpeedInBytesperMS}
							msgs.Players = append(msgs.Players, msg)
						}
					}
				}
				return true
			})
		}
	}

	//resp, _ := json.Marshal(msgs)
	res.Data = msgs
	return res
}
