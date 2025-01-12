package video

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type VideoInfo struct {
	Duration float64
	Format   string
	Size     int64
}

type Processor interface {
	GetVideoInfo(ctx context.Context, filepath string) (*VideoInfo, error)
	Trim(ctx context.Context, inputPath, outputPath string, start, end float64) error
	Merge(ctx context.Context, inputPaths []string, outputPath string) error
}

type FFmpegProcessor struct {
	ffmpegPath  string
	ffprobePath string
}

func NewFFmpegProcessor() *FFmpegProcessor {
	return &FFmpegProcessor{
		ffmpegPath:  "ffmpeg",
		ffprobePath: "ffprobe",
	}
}

func (p *FFmpegProcessor) GetVideoInfo(ctx context.Context, filepath string) (*VideoInfo, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		filepath,
	}

	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get video info: %w", err)
	}

	var result struct {
		Format struct {
			Duration string `json:"duration"`
			Size     string `json:"size"`
			Format   string `json:"format_name"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	duration, err := strconv.ParseFloat(result.Format.Duration, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid duration format: %w", err)
	}

	size, err := strconv.ParseInt(result.Format.Size, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid size format: %w", err)
	}

	return &VideoInfo{
		Duration: duration,
		Format:   result.Format.Format,
		Size:     size,
	}, nil
}

func (p *FFmpegProcessor) Trim(ctx context.Context, inputPath, outputPath string, start, end float64) error {
	args := []string{
		"-i", inputPath,
		"-ss", fmt.Sprintf("%.3f", start),
		"-t", fmt.Sprintf("%.3f", end-start),
		"-c:v", "libx264",
		"-c:a", "aac",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to trim video: %w, output: %s", err, string(output))
	}

	return nil
}

func (p *FFmpegProcessor) Merge(ctx context.Context, inputPaths []string, outputPath string) error {

	tmpDir := filepath.Dir(outputPath)
	listPath := filepath.Join(tmpDir, "filelist.txt")

	var fileContent string
	for _, path := range inputPaths {

		absPath, err := filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}

		escapedPath := strings.ReplaceAll(absPath, "'", "'\\''")
		fileContent += fmt.Sprintf("file '%s'\n", escapedPath)
	}

	if err := os.WriteFile(listPath, []byte(fileContent), 0644); err != nil {
		return fmt.Errorf("failed to write file list: %w", err)
	}
	defer os.Remove(listPath)

	args := []string{
		"-f", "concat",
		"-safe", "0",
		"-i", listPath,
		"-c:v", "libx264",
		"-c:a", "aac",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to merge videos: %w, output: %s", err, string(output))
	}

	return nil
}
