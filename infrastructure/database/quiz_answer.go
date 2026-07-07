//go:generate mockgen -source=$GOFILE -package=mock_$GOPACKAGE -destination=../../mock/$GOPACKAGE/$GOFILE
package database

import (
	"context"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"
	pkgerrors "github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	genModel "github.com/Code0716/stock-price-repository/infrastructure/database/gen_model"
	genQuery "github.com/Code0716/stock-price-repository/infrastructure/database/gen_query"
	"github.com/Code0716/stock-price-repository/models"
	"github.com/Code0716/stock-price-repository/repositories"
)

const mysqlErrNumDuplicateEntry uint16 = 1062

type QuizAnswerRepositoryImpl struct {
	query *genQuery.Query
}

func NewQuizAnswerRepositoryImpl(db *gorm.DB) repositories.QuizAnswerRepository {
	return &QuizAnswerRepositoryImpl{
		query: genQuery.Use(db),
	}
}

func (qi *QuizAnswerRepositoryImpl) Create(ctx context.Context, answer *models.QuizAnswer) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	if err := tx.QuizAnswer.WithContext(ctx).Create(qi.convertToDBModel(answer)); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrNumDuplicateEntry {
			return repositories.ErrQuizAnswerAlreadyExists
		}
		return pkgerrors.Wrap(err, "QuizAnswerRepositoryImpl.Create error")
	}
	return nil
}

func (qi *QuizAnswerRepositoryImpl) ListByQuizDate(ctx context.Context, quizDate time.Time) ([]*models.QuizAnswer, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	rows, err := tx.QuizAnswer.WithContext(ctx).
		Where(tx.QuizAnswer.QuizDate.Eq(dateOnlyOf(quizDate))).
		Find()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "QuizAnswerRepositoryImpl.ListByQuizDate error")
	}

	answers := make([]*models.QuizAnswer, 0, len(rows))
	for _, r := range rows {
		answers = append(answers, qi.convertToDomainModel(r))
	}
	return answers, nil
}

func (qi *QuizAnswerRepositoryImpl) ListUngraded(ctx context.Context) ([]*models.QuizAnswer, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	rows, err := tx.QuizAnswer.WithContext(ctx).
		Where(tx.QuizAnswer.Outcome.IsNull()).
		Find()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "QuizAnswerRepositoryImpl.ListUngraded error")
	}

	answers := make([]*models.QuizAnswer, 0, len(rows))
	for _, r := range rows {
		answers = append(answers, qi.convertToDomainModel(r))
	}
	return answers, nil
}

func (qi *QuizAnswerRepositoryImpl) UpdateGrading(ctx context.Context, answers []*models.QuizAnswer) error {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	for _, a := range answers {
		if _, err := tx.QuizAnswer.WithContext(ctx).
			Where(tx.QuizAnswer.ID.Eq(a.ID)).
			Updates(qi.convertToDBModel(a)); err != nil {
			return pkgerrors.Wrap(err, "QuizAnswerRepositoryImpl.UpdateGrading error")
		}
	}
	return nil
}

func (qi *QuizAnswerRepositoryImpl) ListAllGraded(ctx context.Context) ([]*models.QuizAnswer, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	rows, err := tx.QuizAnswer.WithContext(ctx).
		Where(tx.QuizAnswer.Outcome.IsNotNull()).
		Order(tx.QuizAnswer.QuizDate).
		Find()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "QuizAnswerRepositoryImpl.ListAllGraded error")
	}

	answers := make([]*models.QuizAnswer, 0, len(rows))
	for _, r := range rows {
		answers = append(answers, qi.convertToDomainModel(r))
	}
	return answers, nil
}

func (qi *QuizAnswerRepositoryImpl) ListAll(ctx context.Context) ([]*models.QuizAnswer, error) {
	tx, ok := GetTxQuery(ctx)
	if !ok {
		tx = qi.query
	}

	rows, err := tx.QuizAnswer.WithContext(ctx).
		Order(tx.QuizAnswer.AnsweredAt).
		Find()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "QuizAnswerRepositoryImpl.ListAll error")
	}

	answers := make([]*models.QuizAnswer, 0, len(rows))
	for _, r := range rows {
		answers = append(answers, qi.convertToDomainModel(r))
	}
	return answers, nil
}

func (qi *QuizAnswerRepositoryImpl) convertToDomainModel(m *genModel.QuizAnswer) *models.QuizAnswer {
	answer := &models.QuizAnswer{
		ID:           m.ID,
		QuizDate:     m.QuizDate,
		StockBrandID: m.StockBrandID,
		TickerSymbol: m.TickerSymbol,
		Prediction:   models.QuizPrediction(m.Prediction),
		AnsweredAt:   m.AnsweredAt,
		GradedAt:     m.GradedAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
	if m.NextClosePrice != nil {
		v := decimal.NewFromFloat(*m.NextClosePrice)
		answer.NextClosePrice = &v
	}
	if m.ActualReturn != nil {
		v := decimal.NewFromFloat(*m.ActualReturn)
		answer.ActualReturn = &v
	}
	if m.Outcome != nil {
		v := models.QuizOutcome(*m.Outcome)
		answer.Outcome = &v
	}
	if m.Score != nil {
		v := int(*m.Score)
		answer.Score = &v
	}
	return answer
}

func (qi *QuizAnswerRepositoryImpl) convertToDBModel(a *models.QuizAnswer) *genModel.QuizAnswer {
	m := &genModel.QuizAnswer{
		ID:           a.ID,
		QuizDate:     dateOnlyOf(a.QuizDate),
		StockBrandID: a.StockBrandID,
		TickerSymbol: a.TickerSymbol,
		Prediction:   string(a.Prediction),
		AnsweredAt:   a.AnsweredAt,
		GradedAt:     a.GradedAt,
	}
	if a.NextClosePrice != nil {
		v, _ := a.NextClosePrice.Round(4).Float64()
		m.NextClosePrice = &v
	}
	if a.ActualReturn != nil {
		v, _ := a.ActualReturn.Round(6).Float64()
		m.ActualReturn = &v
	}
	if a.Outcome != nil {
		v := string(*a.Outcome)
		m.Outcome = &v
	}
	if a.Score != nil {
		v := int32(*a.Score)
		m.Score = &v
	}
	return m
}
