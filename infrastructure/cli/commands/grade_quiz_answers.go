package commands

import (
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	"github.com/Code0716/stock-price-repository/usecase"
)

// grade_quiz_answers_v1
type GradeQuizAnswersV1Command struct {
	gradeQuizAnswersInteractor usecase.GradeQuizAnswersInteractor
}

func NewGradeQuizAnswersV1Command(gradeQuizAnswersInteractor usecase.GradeQuizAnswersInteractor) *GradeQuizAnswersV1Command {
	return &GradeQuizAnswersV1Command{gradeQuizAnswersInteractor}
}

func (c *GradeQuizAnswersV1Command) Command() *Command {
	return &Command{
		Name:   "grade_quiz_answers_v1",
		Usage:  "当日の日足取得後、翌営業日の終値が確定した未採点のクイズ回答を採点する。",
		Action: c.Action,
	}
}

func (c *GradeQuizAnswersV1Command) Action(ctx *cli.Context) error {
	err := c.gradeQuizAnswersInteractor.GradeQuizAnswers(ctx.Context)
	if err != nil {
		return errors.Wrap(err, "Action error")
	}
	return nil
}
