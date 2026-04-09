package services

import (
	"context"
	"github.com/intigritypos/integritypos/internal/domain"
)

type TurnoService struct {
	SessionRepo domain.SessionRepository
}

func NewTurnoService(sessionRepo domain.SessionRepository) *TurnoService {
	return &TurnoService{SessionRepo: sessionRepo}
}

func (s *TurnoService) OpenSession(ctx context.Context, session domain.CashSession) error {
	return s.SessionRepo.Save(ctx, session)
}

func (s *TurnoService) CloseSession(ctx context.Context, sessionID string, realCash domain.Money) error {
	session, err := s.SessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}
	expected := session.ExpectedCash
	difference := realCash - expected
	return s.SessionRepo.CloseSession(ctx, sessionID, realCash, difference)
}

func (s *TurnoService) AddCashMovement(ctx context.Context, movement domain.CashMovement) error {
	return s.SessionRepo.AddMovement(ctx, movement)
}

