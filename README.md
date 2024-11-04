<h1 align="center">
  <br>
  realtime-vad-go
  <br>
</h1>
<h4 align="center">A simple Golang realtime speech detector powered by Silero VAD</h4>
<p align="center">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>
<br>

## Important Note

Currently, this implementation only supports VAD detection for PCM s16le encoded audio data（**16 bit, 16000hz, pcm**）.

If you need to use other encodings, you will need to convert your audio data to PCM s16le format before processing or just modify the code to support other encodings.

## Quick Start

### Code!

> you can also reference the tests in `vad_test.go`

```go
import (
    vad "github.com/mereithhh/realtime-vad-go"
)

func Demo() {
    // config
    config := &vad.DefaultVadConfig

    // when detect speech, this callback will be called
    onVad = func(pcmData []byte, durationMs int) {
      // onVad implementation
      fmt.Printf("onVad, duration: %d", durationMs)
    }

    onStartSpeaking := func() {
      fmt.Println("onStartSpeaking")
    }

    // Create a new VAD instance
    detector := vad.NewRealTimeVadDetector(config, onVad, onStartSpeaking)

    // start the detector, this function will not block , running in a goroutine
    detector.StartDetect()

    // once you get the audio data, you can feed it to the detector
    vad.PutPcmData(pcmData)

    // stop
    vad.StopDetect()
}

```

### Install the deps and model file

> if your system is not linux x86_64, you need to download the onnxruntime library from the [official website](https://onnxruntime.ai/docs/build/ep_onnxruntime.html) and copy it to your system lib folder.

```shell
sudo cp ./onnxruntime/lib/* /usr/local/lib/
sudo cp ./onnxruntime/include/* /usr/local/include/
sudo ldconfig
sudo mkdir -p /usr/local/share/vad_model
sudo cp ./model_file/* /usr/local/share/vad_model/
```

### With Dockerfile

> if your system is not linux x86_64, you need to download the onnxruntime library from the [official website](https://onnxruntime.ai/docs/build/ep_onnxruntime.html) and copy it to your project folder.

Copy `onnxruntime` and `model_file` in this project to your project folder, and then add the following lines to your Dockerfile:

```dockerfile
COPY  ./onnxruntime/lib/* /usr/local/lib/
COPY  ./onnxruntime/include/* /usr/local/include/
RUN ln -s /usr/local/lib/libonnxruntime.so.1.18.0 /usr/local/lib/libonnxruntime.so
RUN ldconfig

COPY ./model_file/* /usr/local/share/vad_model
```

## Requirements

- [Golang](https://go.dev/doc/install) >= v1.21
- A C compiler (e.g. GCC)
- ONNX Runtime
- A [Silero VAD](https://github.com/snakers4/silero-vad) model

## About

This project is inspired by [ricky0123/vad](https://github.com/ricky0123/vad), a real-time voice activity detector for web applications. Essentially, I have reimplemented this algorithm in Go.

[web version](https://www.vad.ricky0123.com/)

[Silero VAD](https://github.com/snakers4/silero-vad)

[Silero VAD Go](https://github.com/streamer45/silero-vad-go)

## Algorithm

1.  Sample rate conversion is performed on input audio so that the processed audio has a sample rate of 16000.
2.  The converted samples are batched into "frames" of size `frameSamples` samples.
3.  The Silero vad model is run on each frame and produces a number between 0 and 1 indicating the probability that the sample contains speech.
4.  If the algorithm has not detected speech lately, then it is in a state of `not speaking`. Once it encounters a frame with speech probability greater than `positiveSpeechThreshold`, it is changed into a state of `speaking`. When it encounters `redemptionFrames` frames with speech probability less than `negativeSpeechThreshold` without having encountered a frame with speech probability greater than `positiveSpeechThreshold`, the speech audio segment is considered to have ended and the algorithm returns to a state of `not speaking`. Frames with speech probability in between `negativeSpeechThreshold` and `positiveSpeechThreshold` are effectively ignored.
5.  When the algorithm detects the end of a speech audio segment (i.e. goes from the state of `speaking` to `not speaking`), it counts the number of frames with speech probability greater than `positiveSpeechThreshold` in the audio segment. If the count is less than `minSpeechFrames`, then the audio segment is considered a false positive. Otherwise, `preSpeechPadFrames` frames are prepended to the audio segment and the segment is made accessible through the higher-level API.

## Configuration

- `positiveSpeechThreshold: number` - determines the threshold over which a probability is considered to indicate the presence of speech.
- `negativeSpeechThreshold: number` - determines the threshold under which a probability is considered to indicate the absence of speech.
- `redemptionFrames: number` - number of speech-negative frames to wait before ending a speech segment.
- `frameSamples: number` - the size of a frame in samples - 1536 by default and probably should not be changed.
- `preSpeechPadFrames: number` - number of audio frames to prepend to a speech segment.
- `minSpeechFrames: number` - minimum number of speech-positive frames for a speech segment.

## License

MIT License - see [LICENSE](LICENSE) for full text
