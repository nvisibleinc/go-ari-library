package nv


import (
  "encoding/json"
  "time"
)

type NV_Event struct {
  ServerID    string    `json:"server_id"`
  Timestamp   time.Time `json:"timestamp"`
  Type        string    `json:"type"`
  ARI_Event   string    `json:"ari_event"`
}

func Init(in chan []byte, out chan *NV_Event) {
  go func(in chan []byte, out chan *NV_Event) {
    for instring := range in {
      var e *NV_Event
      json.Unmarshal(instring, e)
      out <- e
    }
  }(in, out)
}