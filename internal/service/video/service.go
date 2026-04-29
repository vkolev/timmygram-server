package video

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"timmygram/internal/model"
	"timmygram/internal/repository"
)

var (
	ErrInvalidFileType = errors.New("file must be a video")
	ErrFileTooLarge    = errors.New("file exceeds 200 MB limit")
)

type VideoService struct {
	videoRepo   repository.VideoRepository
	storagePath string
	maxDuration int
	outputRatio string
}

func NewVideoService(repo repository.VideoRepository, storagePath string, maxDuration int, outputRatio string) *VideoService {
	return &VideoService{
		videoRepo:   repo,
		storagePath: storagePath,
		maxDuration: maxDuration,
		outputRatio: outputRatio,
	}
}

func (s *VideoService) Upload(userID int, title string, fileHeader *multipart.FileHeader) (*model.Video, error) {
	// Size check before opening.
	if fileHeader.Size > 200*1024*1024 {
		return nil, ErrFileTooLarge
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// MIME check via magic bytes.
	buf := make([]byte, 512)
	n, err := src.Read(buf)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	contentType := http.DetectContentType(buf[:n])
	if !strings.HasPrefix(contentType, "video/") && contentType != "application/octet-stream" {
		return nil, ErrInvalidFileType
	}
	// Rewind after reading magic bytes.
	if seeker, ok := src.(io.Seeker); ok {
		if _, err := seeker.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	uuid, err := generateUUID()
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = ".mp4"
	}

	rawDir := filepath.Join(s.storagePath, "raw")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return nil, err
	}
	rawPath := filepath.Join(rawDir, uuid+ext)

	dst, err := os.Create(rawPath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return nil, err
	}

	v := &model.Video{
		UserID:      userID,
		Title:       title,
		Filename:    uuid,
		OutputRatio: s.outputRatio,
	}
	id, err := s.videoRepo.Create(v)
	if err != nil {
		os.Remove(rawPath)
		return nil, err
	}
	v.ID = int(id)

	go s.transcode(v.ID, rawPath, uuid)

	return v, nil
}

func (s *VideoService) transcode(videoID int, rawPath, uuid string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("transcode panic for video %d: %v", videoID, r)
		}
	}()

	duration, width, height, err := probe(rawPath)
	if err != nil {
		log.Printf("ffprobe failed for video %d: %v", videoID, err)
		return
	}

	if duration > s.maxDuration {
		log.Printf("video %d exceeds max duration (%ds > %ds), removing", videoID, duration, s.maxDuration)
		s.videoRepo.Delete(videoID)
		os.Remove(rawPath)
		return
	}

	transcodedDir := filepath.Join(s.storagePath, "transcoded")
	if err := os.MkdirAll(transcodedDir, 0o755); err != nil {
		log.Printf("mkdir transcoded failed: %v", err)
		return
	}
	outPath := filepath.Join(transcodedDir, uuid+".mp4")

	if err := transcode(rawPath, outPath); err != nil {
		log.Printf("ffmpeg transcode failed for video %d: %v", videoID, err)
		return
	}

	thumbDir := filepath.Join(s.storagePath, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0o755); err != nil {
		log.Printf("mkdir thumbnails failed: %v", err)
		return
	}
	thumbPath := filepath.Join(thumbDir, uuid+".jpg")

	if err := generateThumbnail(rawPath, thumbPath); err != nil {
		log.Printf("thumbnail generation failed for video %d: %v", videoID, err)
		// Non-fatal — proceed without thumbnail.
	}

	aspectRatio := computeAspectRatio(width, height)
	thumbFilename := ""
	if _, err := os.Stat(thumbPath); err == nil {
		thumbFilename = uuid + ".jpg"
	}

	if err := s.videoRepo.UpdateTranscoded(videoID, duration, aspectRatio, thumbFilename); err != nil {
		log.Printf("UpdateTranscoded failed for video %d: %v", videoID, err)
	}
}

func (s *VideoService) GetUserVideos(userID int) ([]*model.Video, error) {
	return s.videoRepo.FindByUserID(userID)
}

func (s *VideoService) GetVideo(id, userID int) (*model.Video, error) {
	v, err := s.videoRepo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if v.UserID != userID {
		return nil, model.ErrVideoForbidden
	}
	return v, nil
}

func (s *VideoService) UpdateTitle(id, userID int, title string) error {
	v, err := s.GetVideo(id, userID)
	if err != nil {
		return err
	}

	return s.videoRepo.UpdateTitle(v.ID, strings.TrimSpace(title))
}

func (s *VideoService) LikeVideo(id, userID int, deviceID string) (int, error) {
	v, err := s.GetVideo(id, userID)
	if err != nil {
		return 0, err
	}

	return s.videoRepo.LikeVideo(v.ID, strings.TrimSpace(deviceID))
}

func (s *VideoService) CountLikes(id, userID int) (int, error) {
	v, err := s.GetVideo(id, userID)
	if err != nil {
		return 0, err
	}

	return s.videoRepo.CountLikes(v.ID)
}

func (s *VideoService) DeleteVideo(id, userID int) error {
	v, err := s.GetVideo(id, userID)
	if err != nil {
		return err
	}

	if err := s.videoRepo.Delete(v.ID); err != nil {
		return err
	}

	_ = os.Remove(filepath.Join(s.storagePath, "raw", v.Filename+".mp4"))
	_ = os.Remove(filepath.Join(s.storagePath, "transcoded", v.Filename+".mp4"))
	if v.Thumbnail != "" {
		_ = os.Remove(filepath.Join(s.storagePath, "thumbnails", v.Thumbnail))
	}

	return nil
}

// ── FFmpeg helpers ─────────────────────────────────────────────────────────────

type ffprobeOutput struct {
	Format  struct{ Duration string } `json:"format"`
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func probe(path string) (durationSecs, width, height int, err error) {
	out, err := exec.Command(
		"ffprobe", "-v", "quiet",
		"-print_format", "json",
		"-show_format", "-show_streams",
		path,
	).Output()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe: %w", err)
	}

	var data ffprobeOutput
	if err := json.Unmarshal(out, &data); err != nil {
		return 0, 0, 0, fmt.Errorf("ffprobe parse: %w", err)
	}

	dur, _ := strconv.ParseFloat(data.Format.Duration, 64)
	durationSecs = int(dur)

	for _, s := range data.Streams {
		if s.Width > 0 && s.Height > 0 {
			width, height = s.Width, s.Height
			break
		}
	}
	return durationSecs, width, height, nil
}

func transcode(input, output string) error {
	cmd := exec.Command(
		"ffmpeg", "-y", "-i", input,
		"-vf", "scale=720:1280:force_original_aspect_ratio=decrease,pad=720:1280:(ow-iw)/2:(oh-ih)/2:color=black",
		"-c:v", "libx264", "-profile:v", "baseline", "-level", "3.0",
		"-c:a", "aac", "-movflags", "+faststart",
		output,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w\n%s", err, string(out))
	}
	return nil
}

func generateThumbnail(input, output string) error {
	cmd := exec.Command(
		"ffmpeg", "-y", "-i", input,
		"-ss", "2", "-vframes", "1",
		output,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg thumbnail: %w\n%s", err, string(out))
	}
	return nil
}

// ── Utilities ──────────────────────────────────────────────────────────────────

func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func computeAspectRatio(width, height int) string {
	if width == 0 || height == 0 {
		return ""
	}
	g := gcd(width, height)
	return fmt.Sprintf("%d:%d", width/g, height/g)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
