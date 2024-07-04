package vad_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	vad "github.com/mereithhh/realtime-vad-go"
)

func TestVad(t *testing.T) {
	config := &vad.DefaultVadConfig
	config.RedemptionFrames = 5
	onVad := func(pcmData []byte, durationMs int) {
		// onVad implementation
		t.Logf("onVad, duration: %d", durationMs)
	}
	vad, err := vad.NewRealTimeVadDetector(config, onVad)
	if err != nil {
		t.Fatal(err)
		return
	}
	vad.StartDetect()

	pcmData := loadPcm()
	// send pcm data
	for i := 0; i < 10; i++ {
		vad.PutPcmData(pcmData)
	}
	time.Sleep(10 * time.Second)
	vad.StopDetect()
}

func loadPcm() []byte {
	data, _ := os.ReadFile("./test.wav")
	pcmData, _ := wavToPCM(data)
	return pcmData
}

func wavToPCM(wavData []byte) ([]byte, error) {
	wavHeaderSize := 44
	// 检查输入数据的长度是否小于 WAV 文件头大小
	if len(wavData) < wavHeaderSize {
		return nil, fmt.Errorf("input data is too short to be a valid WAV file")
	}

	// 跳过 WAV 文件头部分，获取 PCM 数据
	pcmData := wavData[wavHeaderSize:]

	return pcmData, nil
}
