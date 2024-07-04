package vad

// 音频缓冲区，里面的 cache 可以自动变长
type IAudioCache interface {
	// 放入音频数据
	Put(data []byte)
	// 取出音频数据
	GetAll() []byte
	// 取指定大小数据
	GetSize(size int) []byte
	// 清空缓冲区
	Clear()
	// 缓冲区大小
	Size() int
}

// 确保 AudioCache 实现了 IAudioCache 接口
var _ IAudioCache = &AudioCache{}

type AudioCache struct {
	cache []byte
}

func (a *AudioCache) Put(data []byte) {
	// Put implementation
	a.cache = append(a.cache, data...)
}

func (a *AudioCache) GetAll() []byte {
	// Get implementation
	if len(a.cache) == 0 {
		return nil
	}
	data := a.cache
	a.cache = nil
	return data
}
func (a *AudioCache) GetSize(size int) []byte {
	// GetSize implementation
	if len(a.cache) == 0 {
		return nil
	}
	fixNum := 0
	if len(a.cache) < size {
		fixNum = size - len(a.cache)
		size = len(a.cache)
	}
	data := a.cache[:size]
	if fixNum > 0 {
		data = append(data, make([]byte, fixNum)...)
	}
	a.cache = a.cache[size:]
	return data
}

func (a *AudioCache) Clear() {
	// Clear implementation
	a.cache = nil
}

func (a *AudioCache) Size() int {
	// Size implementation
	return len(a.cache)
}

func NewAudioCache() *AudioCache {
	// NewAudioCache implementation
	return &AudioCache{
		cache: make([]byte, 0),
	}
}
