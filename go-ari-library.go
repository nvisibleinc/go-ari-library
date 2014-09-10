package nv


import (
  "encoding/json"
  "time"
)

type NV_Event struct {
    ServerID    string
    Timestamp   time.Time
    ARI_Event   string
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