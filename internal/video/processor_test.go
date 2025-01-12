package video

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func createTestVideo(t *testing.T, path string, duration int) {

	args := []string{
		"-f", "lavfi",
		"-i", fmt.Sprintf("testsrc=duration=%d:size=320x240:rate=30", duration),
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-t", fmt.Sprintf("%d", duration),
		"-y",
		path,
	}

	cmd := exec.Command("ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to create test video: %v, output: %s", err, string(output))
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Test video was not created: %v", err)
	}
}

func TestFFmpegProcessor(t *testing.T) {

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not found in PATH")
	}

	tmpDir, err := os.MkdirTemp("", "videoapi-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testVideoPath := filepath.Join(tmpDir, "test.mp4")
	createTestVideo(t, testVideoPath, 5)

	processor := NewFFmpegProcessor()
	ctx := context.Background()

	t.Run("GetVideoInfo", func(t *testing.T) {
		info, err := processor.GetVideoInfo(ctx, testVideoPath)
		if err != nil {
			t.Fatalf("Failed to get video info: %v", err)
		}

		if info.Duration < 4.9 || info.Duration > 5.1 {
			t.Errorf("Expected duration around 5s, got %f", info.Duration)
		}

		if info.Size == 0 {
			t.Error("Expected non-zero size")
		}
	})

	t.Run("Trim", func(t *testing.T) {
		outputPath := filepath.Join(tmpDir, "trimmed.mp4")
		err := processor.Trim(ctx, testVideoPath, outputPath, 1, 3)
		if err != nil {
			t.Fatalf("Failed to trim video: %v", err)
		}

		info, err := processor.GetVideoInfo(ctx, outputPath)
		if err != nil {
			t.Fatalf("Failed to get trimmed video info: %v", err)
		}

		expectedDuration := 2.0
		if info.Duration < expectedDuration-0.2 || info.Duration > expectedDuration+0.2 {
			t.Errorf("Expected duration around %.1fs, got %.1fs", expectedDuration, info.Duration)
		}
	})

	t.Run("Merge", func(t *testing.T) {

		testVideo2Path := filepath.Join(tmpDir, "test2.mp4")
		createTestVideo(t, testVideo2Path, 3)

		outputPath := filepath.Join(tmpDir, "merged.mp4")
		err := processor.Merge(ctx, []string{testVideoPath, testVideo2Path}, outputPath)
		if err != nil {
			t.Fatalf("Failed to merge videos: %v", err)
		}

		info, err := processor.GetVideoInfo(ctx, outputPath)
		if err != nil {
			t.Fatalf("Failed to get merged video info: %v", err)
		}

		expectedDuration := 8.0
		if info.Duration < expectedDuration-0.2 || info.Duration > expectedDuration+0.2 {
			t.Errorf("Expected duration around %.1fs, got %.1fs", expectedDuration, info.Duration)
		}
	})
}
