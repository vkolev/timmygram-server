package user

import (
	"errors"
	"timmygram/internal/model"
	"timmygram/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken = errors.New("username taken")
	ErrCannotDelete  = errors.New("cannot delete the owner account")
)

type UserService struct {
	userRepo repository.UserRepository
}

func NewUserService(userRepo repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) ListUsers() ([]*model.User, error) {
	return s.userRepo.List()
}

func (s *UserService) CreateUser(username, password string) (*model.User, error) {
	if _, err := s.userRepo.FindByUsername(username); err == nil {
		return nil, ErrUsernameTaken
	} else if !errors.Is(err, model.ErrUserNotFound) {
		return nil, err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{Username: username, Password: string(hashed), IsOwner: false}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) DeleteUser(id int) error {
	user, err := s.userRepo.FindByID(id)
	if err != nil {
		return err
	}
	if user.IsOwner {
		return ErrCannotDelete
	}
	return s.userRepo.Delete(id)
}
