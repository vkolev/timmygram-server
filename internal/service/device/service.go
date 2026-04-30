package device

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
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

var ErrInvalidPIN = errors.New("invalid or expired PIN")

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

func (s *DeviceService) GetOrCreatePairingPIN(userID int) (*model.DevicePairingPIN, error) {
	if err := s.deviceRepo.DeleteExpiredPairingPINs(); err != nil {
		return nil, err
	}

	existingPIN, err := s.deviceRepo.FindActivePairingPINByUserID(userID)
	if err == nil {
		return existingPIN, nil
	}
	if !errors.Is(err, model.ErrDevicePairingPINNotFound) {
		return nil, err
	}

	token, err := s.GenerateDeviceToken(userID)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(5 * time.Minute)

	for attempts := 0; attempts < 10; attempts++ {
		pin, err := generateSixDigitPIN()
		if err != nil {
			return nil, err
		}

		pairingPIN := &model.DevicePairingPIN{
			UserID:    userID,
			PIN:       pin,
			Token:     token,
			ExpiresAt: expiresAt,
		}

		if err := s.deviceRepo.CreatePairingPIN(pairingPIN); err == nil {
			return pairingPIN, nil
		}
	}

	return nil, errors.New("failed to create unique pairing pin")
}

func (s *DeviceService) ExchangePIN(pin string) (string, error) {

	if err := s.deviceRepo.DeleteExpiredPairingPINs(); err != nil {
		return "", err
	}

	pairingPIN, err := s.deviceRepo.FindActivePairingPIN(strings.TrimSpace(pin))
	if err != nil {
		if errors.Is(err, model.ErrDevicePairingPINNotFound) {
			return "", ErrInvalidPIN
		}

		return "", err
	}

	if err := s.deviceRepo.DeletePairingPIN(pairingPIN.ID); err != nil {
		return "", err
	}

	return pairingPIN.Token, nil
}

func generateSixDigitPIN() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
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

func (s *DeviceService) BlockDevice(id, userID int) error {
	return s.deviceRepo.Block(id, userID)
}

func (s *DeviceService) UnblockDevice(id, userID int) error {
	return s.deviceRepo.Unblock(id, userID)
}

func (s *DeviceService) DeleteDevice(id, userID int) error {
	return s.deviceRepo.Delete(id, userID)
}

func (s *DeviceService) GetRandomFeed(userID int, limit int) ([]*model.Video, error) {
	return s.videoRepo.FindRandomByUserID(userID, limit)
}

func (s *DeviceService) GetFeed(userID int, page, pageSize int) ([]*model.Video, bool, error) {
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
