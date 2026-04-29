package device

import (
	"time"
	"timmygram/internal/model"
	"timmygram/internal/repository"
	"timmygram/internal/utils/qr"

	"github.com/golang-jwt/jwt/v5"
)

type DeviceService struct {
	deviceRepo repository.DeviceRepository
	videoRepo  repository.VideoRepository
	jwtSecret  string
	serverURL  string
}

func NewDeviceService(deviceRepo repository.DeviceRepository, videoRepo repository.VideoRepository, jwtSecret, serverURL string) *DeviceService {
	return &DeviceService{
		deviceRepo: deviceRepo,
		videoRepo:  videoRepo,
		jwtSecret:  jwtSecret,
		serverURL:  serverURL,
	}
}

func (s *DeviceService) GenerateDeviceToken(userID int) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"type":    "device",
		"exp":     time.Now().Add(365 * 24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *DeviceService) GenerateQRPayload(userID int) (qr.QRPayload, error) {
	token, err := s.GenerateDeviceToken(userID)
	if err != nil {
		return qr.QRPayload{}, err
	}
	return qr.QRPayload{ServerURL: s.serverURL, Token: token}, nil
}

func (s *DeviceService) RegisterDevice(userID int, deviceID, deviceName string) error {
	return s.deviceRepo.Upsert(&model.Device{
		UserID:     userID,
		DeviceID:   deviceID,
		DeviceName: deviceName,
	})
}

func (s *DeviceService) GetDevices(userID int) ([]*model.Device, error) {
	return s.deviceRepo.FindByUserID(userID)
}

func (s *DeviceService) GetRandomFeed(userID int, deviceID string, limit int) ([]*model.Video, error) {
	return s.videoRepo.FindRandomByUserID(userID, limit)
}

func (s *DeviceService) GetFeed(userID int, deviceID string, page, pageSize int) ([]*model.Video, bool, error) {
	if page < 1 {
		page = 1
	}

	offset := (page - 1) * pageSize
	videos, err := s.videoRepo.FindPageByUserID(userID, pageSize+1, offset)
	if err != nil {
		return nil, false, err
	}

	hasNextPage := len(videos) > pageSize
	if hasNextPage {
		videos = videos[:pageSize]
	}
	return videos, hasNextPage, nil
}

func (s *DeviceService) GetRandomVideo(userId int, deviceID string) (*model.Video, error) {
	result, err := s.videoRepo.GetRandomUnwatchedVideo(userId, deviceID)
	if err != nil {
		return nil, err
	}
	return result, nil
}
