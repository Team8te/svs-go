package endpoint

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
