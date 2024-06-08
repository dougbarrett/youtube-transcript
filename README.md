# YouTube Transcript API

Heavily inspired by the excellent Python library: https://github.com/jdepoix/youtube-transcript-api

*** THIS IS PROBABLY NOT SUITABLE FOR ALL USE CASES, I THREW THIS TOGETHER IN A FEW HOURS ***

```go
package main

import (
    "log"
    "context"

    "github.com/dougbarrett/youtube-transcript"
)

func main() {
    videoID := "VIDEO_ID"
    transcript, err := youtubetranscript.GetTranscript(context.Context(), videoID)

    if err != nil {
        panic(err)
    }

    log.Printf("%v", transcript)
}
```

