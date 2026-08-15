package meters

import "OpenCNC_config_service/monitor_service/structures/monitoring"

type SamplesWindow struct {
	size    uint32
	samples []*monitoring.DataSample
}

func NewSampleWindow(size uint32) *SamplesWindow {
	return &SamplesWindow{
		size:    size,
		samples: make([]*monitoring.DataSample, 0, size),
	}
}

func (w *SamplesWindow) Add(sample *monitoring.DataSample) {
	w.samples = append(w.samples, sample)

	if uint32(len(w.samples)) > w.size {
		w.samples = w.samples[len(w.samples)-int(w.size):]
	}
}

func (w *SamplesWindow) Ready() bool {
	return uint32(len(w.samples)) >= w.size
}

func (w *SamplesWindow) Samples() []*monitoring.DataSample {
	return w.samples
}

func (w *SamplesWindow) Reset() {
	w.samples = w.samples[:0]
}
