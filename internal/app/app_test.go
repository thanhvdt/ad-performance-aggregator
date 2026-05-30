package app

import "testing"

func TestBuildConfig(t *testing.T) {
	const (
		mib = 1 << 20
		kib = 1 << 10
	)
	cases := []struct {
		name                                   string
		workers, chunkKiB, memLimitMiB         int
		wantWorkers, wantChunkSize, wantMemLim int
	}{
		{"clamp zero workers and chunk", 0, 0, 0, 1, 512 * kib, 0},
		{"explicit mem limit", 4, 64, 128, 4, 64 * kib, 128 * mib},
		{"mem off", 4, 64, 0, 4, 64 * kib, 0},
		{"auto mem hits floor", 1, 512, -1, 1, 512 * kib, 32 * mib},            // 6*1*512KiB ≈ 3MiB → floored
		{"auto mem scales up", 20, 512, -1, 20, 512 * kib, 6 * 20 * 512 * kib}, // ≈60MiB > floor
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := buildConfig("in.csv", "out", 10, tc.workers, tc.chunkKiB, tc.memLimitMiB)
			if cfg.workers != tc.wantWorkers {
				t.Errorf("workers = %d, want %d", cfg.workers, tc.wantWorkers)
			}
			if cfg.chunkSize != tc.wantChunkSize {
				t.Errorf("chunkSize = %d, want %d", cfg.chunkSize, tc.wantChunkSize)
			}
			if cfg.memLimit != tc.wantMemLim {
				t.Errorf("memLimit = %d, want %d", cfg.memLimit, tc.wantMemLim)
			}
		})
	}
}
