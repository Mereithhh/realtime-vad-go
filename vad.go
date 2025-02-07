package vad

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/Mereithhh/silero-vad-go/speech"
)

type IVadDetector interface {
	DetectPcmAtom(pcmData []byte, channelNum int64, sampleRate int64, bitSize int64) (float32, error)
	StartDetect()
	PutPcmData(pcmData []byte)
	Close() error
}

type VadConfig struct {
	PositiveSpeechThreshold float32
	NegativeSpeechThreshold float32
	RedemptionFrames        int
	MinSpeechFrames         int
	PreSpeechPadFrames      int
	FrameSamples            int
	VadInterval             time.Duration
}

type RealTimeVadDetector struct {
	Sd                   *speech.Detector
	Config               *VadConfig
	InputAudioCache      *AudioCache
	VadAudioCache        *AudioCache
	VadNotSpeakingFrames [][]byte
	VadNotPassChunkSize  int
	isVadSpeaking        bool
	OnRecvVadAudio       func([]byte, int)
	OnStartSpeaking      func()
	done                 chan struct{}
	isClosed             int32
}

var _ IVadDetector = &RealTimeVadDetector{}

var DefaultVadConfig = VadConfig{
	PositiveSpeechThreshold: 0.65,
	NegativeSpeechThreshold: 0.35,
	RedemptionFrames:        8, // 8x32ms = 256ms
	MinSpeechFrames:         3, // 3x32ms = 96ms
	PreSpeechPadFrames:      1,
	FrameSamples:            512, // 32ms
	VadInterval:             10 * time.Millisecond,
}

func NewSdVad() (*speech.Detector, error) {
	vad, err := speech.NewDetector(speech.DetectorConfig{
		ModelPath:  "/usr/local/share/vad_model/silero_vad.onnx",
		SampleRate: 16000,
		Threshold:  0.5,
	})
	if err != nil {
		return nil, err
	}

	return vad, nil
}

func NewRealTimeVadDetector(config *VadConfig, callBackFn func(b []byte, ms int), onStartSpeaking func()) (*RealTimeVadDetector, error) {
	sd, err := NewSdVad()
	if err != nil {
		return nil, err
	}
	detector := &RealTimeVadDetector{
		Sd:                   sd,
		InputAudioCache:      NewAudioCache(),
		VadAudioCache:        NewAudioCache(),
		OnRecvVadAudio:       callBackFn,
		OnStartSpeaking:      onStartSpeaking,
		VadNotSpeakingFrames: make([][]byte, 0),
		done:                 make(chan struct{}),
		isClosed:             0,
	}

	if config != nil {
		detector.Config = config
	} else {
		detector.Config = &DefaultVadConfig
	}

	runtime.SetFinalizer(detector, func(v *RealTimeVadDetector) {
		if v.Sd != nil {
			v.Sd.Destroy()
		}
	})

	return detector, nil
}

// 探测给定的 pcm 数据中是否包含了人声的可能性, 最后送入模型的是f32le,16k,1channel = 4byte
func (v *RealTimeVadDetector) DetectPcmAtom(pcmData []byte, channelNum int64, sampleRate int64, bitNum int64) (float32, error) {
	byteSize := bitNum / 8
	samples := make([]float32, 0, len(pcmData)/int(byteSize))
	// fmt.Println("pcm data", len(pcmData), bitNum, byteSize)
	for i := 0; i < len(pcmData); i += int(byteSize) {
		if bitNum == 16 {
			sample := int16(binary.LittleEndian.Uint16(pcmData[i : i+int(byteSize)]))
			floatSample := float32(sample) / 32768.0
			// sample := float32(binary.LittleEndian.Uint16(pcmData[i : i+int(byteSize)]))
			// floatSample := float32(sample) / 32768.0
			samples = append(samples, floatSample)
		} else if bitNum == 32 {
			sample := math.Float32frombits(binary.LittleEndian.Uint32(pcmData[i : i+int(byteSize)]))
			samples = append(samples, sample)
		} else {
			return 0, fmt.Errorf("unsupported bit size")
		}
	}
	// fmt.Println("samples", len(samples))
	prob, err := v.Sd.Infer(samples)
	// fmt.Printf("segment: %v\n", segment)
	if err != nil {
		return 0, err
	}

	return prob, nil

}
func (v *RealTimeVadDetector) TryVAD() {
	frameSize := v.Config.FrameSamples * 2
	if v.InputAudioCache.Size() >= frameSize {
		data := v.InputAudioCache.GetSize(frameSize)
		vadResult, _ := v.DetectPcmAtom(data, 1, 16000, 16)
		if vadResult > v.Config.PositiveSpeechThreshold {
			v.VadNotPassChunkSize = 0
			if !v.isVadSpeaking {
				v.isVadSpeaking = true
				v.OnStartSpeaking()
			}
		} else if vadResult < v.Config.NegativeSpeechThreshold {
			v.VadNotPassChunkSize++
			if v.VadNotPassChunkSize >= v.Config.RedemptionFrames {
				if v.isVadSpeaking {
					v.isVadSpeaking = false
					allVadCache := v.VadAudioCache.GetAll()
					thisFrameSize := len(allVadCache) / (frameSize)
					if thisFrameSize > v.Config.MinSpeechFrames {
						padedBytes := padPreSpeechBytes(allVadCache, v.VadNotSpeakingFrames, v.Config.PreSpeechPadFrames)
						durationMs := len(padedBytes) / 32
						v.VadNotSpeakingFrames = make([][]byte, 0)
						v.OnRecvVadAudio(padedBytes, durationMs)
					}
				}
			}
		}
		if v.isVadSpeaking {
			v.VadAudioCache.Put(data)
		} else {
			maxFrames := v.Config.PreSpeechPadFrames
			if len(v.VadNotSpeakingFrames) >= maxFrames {
				v.VadNotSpeakingFrames = v.VadNotSpeakingFrames[1:]
			}
			v.VadNotSpeakingFrames = append(v.VadNotSpeakingFrames, data)
		}
	}
}

func (v *RealTimeVadDetector) StartFn() {
	for {
		select {
		case <-v.done:
			return
		default:
			v.TryVAD()
			time.Sleep(v.Config.VadInterval)
		}
	}
}
func (v *RealTimeVadDetector) StartDetect() {
	go v.StartFn()
}

func (v *RealTimeVadDetector) Close() error {
	if atomic.LoadInt32(&v.isClosed) == 1 {
		return nil
	}
	close(v.done)
	atomic.StoreInt32(&v.isClosed, 1)
	return nil
}

func (v *RealTimeVadDetector) PutPcmData(pcmData []byte) {
	v.InputAudioCache.Put(pcmData)
}

func padPreSpeechBytes(data []byte, toPadData [][]byte, frameSize int) []byte {
	if len(data) == 0 {
		return data
	}
	getSize := frameSize
	toPadFrameSize := len(toPadData)
	if toPadFrameSize < frameSize {
		getSize = toPadFrameSize
	}
	// 从 toPadData 里面拿出 getSize 个 frame 的数据
	dataToMerage := toPadData[:getSize]
	padData := make([]byte, 0)
	for _, frame := range dataToMerage {
		padData = append(padData, frame...)
	}
	return append(padData, data...)
}
