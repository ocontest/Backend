package problems

import (
	"bytes"
	"context"
	"errors"
	"net/http"

	"github.com/ocontest/backend/internal/db/repos"
	"github.com/ocontest/backend/pkg"
	"github.com/ocontest/backend/pkg/structs"

	"github.com/sirupsen/logrus"
)

type ProblemsHandler interface {
	CreateProblem(ctx context.Context, req structs.RequestCreateProblem) (structs.ResponseCreateProblem, int)
	GetProblem(ctx context.Context, problemID int64) (structs.ResponseGetProblem, int)
	ListProblem(ctx context.Context, req structs.RequestListProblems) (structs.ResponseListProblems, int)
	DeleteProblem(ctx context.Context, problemId int64) int
	AddTestcase(ctx context.Context, problemID int64, data []byte) int
	GetTestcase(ctx context.Context, problemID int64) ([]structs.ResponseGetTestcase, int)
	UpdateProblem(ctx context.Context, req structs.RequestUpdateProblem) int
}

type ProblemsHandlerImp struct {
	problemMetadataRepo     repos.ProblemsMetadataRepo
	problemsDescriptionRepo repos.ProblemDescriptionsRepo
	testcaseRepo            repos.TestCaseRepo
}

func NewProblemsHandler(
	problemsRepo repos.ProblemsMetadataRepo, problemsDescriptionRepo repos.ProblemDescriptionsRepo,
	testcaseRepo repos.TestCaseRepo,
) ProblemsHandler {
	return &ProblemsHandlerImp{
		problemMetadataRepo:     problemsRepo,
		problemsDescriptionRepo: problemsDescriptionRepo,
		testcaseRepo:            testcaseRepo,
	}
}

func (p ProblemsHandlerImp) CreateProblem(ctx context.Context, req structs.RequestCreateProblem) (ans structs.ResponseCreateProblem, status int) {
	logger := pkg.Log.WithField("method", "create_problem")
	docID, err := p.problemsDescriptionRepo.Insert(req.Description, nil)
	if err != nil {
		logger.Error("error on inserting problem description: ", err)
		status = http.StatusInternalServerError
		return
	}
	problem := structs.Problem{
		Title:      req.Title,
		DocumentID: docID,
		CreatedBy:  ctx.Value("user_id").(int64),
		IsPrivate:  req.IsPrivate,
		Hardness:   req.Hardness,
	}
	ans.ProblemID, err = p.problemMetadataRepo.InsertProblem(ctx, problem)
	if err != nil {
		logger.Error("error on inserting problem metadata: ", err)
		status = http.StatusInternalServerError
		return
	}
	status = http.StatusOK
	return
}

func (p ProblemsHandlerImp) GetProblem(ctx context.Context, problemID int64) (structs.ResponseGetProblem, int) {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "GetProblem",
		"module": "Problems",
	})

	problem, err := p.problemMetadataRepo.GetProblem(ctx, problemID)
	if err != nil {
		logger.Error("error on getting problem from problem metadata repos: ", err)
		status := http.StatusInternalServerError
		if errors.Is(err, pkg.ErrNotFound) {
			status = http.StatusNotFound
		}
		return structs.ResponseGetProblem{}, status
	}

	doc, err := p.problemsDescriptionRepo.Get(problem.DocumentID)
	if err != nil {
		logger.Error("error on getting problem from problem decription repos: ", err)
		return structs.ResponseGetProblem{}, http.StatusInternalServerError
	}

	return structs.ResponseGetProblem{
		ProblemID:   problemID,
		Title:       problem.Title,
		SolveCount:  problem.SolvedCount,
		Hardness:    problem.Hardness,
		Description: doc.Description,
		IsOwned:     problem.CreatedBy == ctx.Value("user_id").(int64),
	}, http.StatusOK
}

func (p ProblemsHandlerImp) ListProblem(ctx context.Context, req structs.RequestListProblems) (structs.ResponseListProblems, int) {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "ListProblem",
		"module": "Problems",
	})
	problems, total_count, err := p.problemMetadataRepo.ListProblems(ctx, req.OrderedBy, req.Descending, req.Limit, req.Offset, req.GetCount)
	if err != nil {
		logger.Error("error on listing problems: ", err)
		return structs.ResponseListProblems{}, http.StatusInternalServerError
	}

	ans := make([]structs.ResponseListProblemsItem, 0)
	for _, p := range problems {
		ans = append(ans, structs.ResponseListProblemsItem{
			ProblemID:  p.ID,
			Title:      p.Title,
			SolveCount: p.SolvedCount,
			Hardness:   p.Hardness,
		})
	}
	return structs.ResponseListProblems{
		TotalCount: total_count,
		Problems:   ans,
	}, http.StatusOK
}

func (p ProblemsHandlerImp) UpdateProblem(ctx context.Context, req structs.RequestUpdateProblem) int {

	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "UpdateProblem",
		"module": "Problems",
	})

	problem, err := p.problemMetadataRepo.GetProblem(ctx, req.Id)
	if err != nil {
		logger.Error("error on getting problem from problem metadata repos: ", err)
		status := http.StatusInternalServerError
		if errors.Is(err, pkg.ErrNotFound) {
			status = http.StatusNotFound
		}
		return status
	}
	if problem.CreatedBy != ctx.Value("user_id").(int64) {
		return http.StatusForbidden
	}

	err = p.problemMetadataRepo.UpdateProblem(ctx, req.Id, req.Title, req.Hardness)
	if err != nil {
		logger.Error("error on updating problem on problem metadata repo: ", err)
		status := http.StatusInternalServerError
		if errors.Is(err, pkg.ErrNotFound) {
			status = http.StatusNotFound
		}
		return status
	}

	if req.Description != "" {
		err = p.problemsDescriptionRepo.Update(problem.DocumentID, req.Description)
		if err != nil {
			logger.Error("error on updating problem description: ", err)
			status := http.StatusInternalServerError
			return status
		}
	}

	return http.StatusAccepted
}

func (p ProblemsHandlerImp) DeleteProblem(ctx context.Context, problemID int64) int {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "DeleteProblem",
		"module": "Problems",
	})

	problem, err := p.problemMetadataRepo.GetProblem(ctx, problemID)
	if err != nil {
		logger.Error("error on getting problem from problem metadata repos: ", err)
		status := http.StatusInternalServerError
		if errors.Is(err, pkg.ErrNotFound) {
			status = http.StatusNotFound
		}
		return status
	}
	if problem.CreatedBy != ctx.Value("user_id").(int64) {
		return http.StatusForbidden
	}

	documentID, err := p.problemMetadataRepo.DeleteProblem(ctx, problemID)
	if err != nil {
		logger.Error("error on deleting problem from problem metadata repo: ", err)
		status := http.StatusInternalServerError
		if errors.Is(err, pkg.ErrNotFound) {
			status = http.StatusNotFound
		}
		return status
	}

	err = p.problemsDescriptionRepo.Delete(documentID)
	if err != nil {
		logger.Error("error on deleting problem from problem decription repo: ", err)
		return http.StatusInternalServerError
	}

	return http.StatusAccepted
}

func (p ProblemsHandlerImp) AddTestcase(ctx context.Context, problemID int64, data []byte) int {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "AddTestcase",
		"module": "Problems",
	})

	testCases, err := unzip(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		logger.Error("error on unzip file: ", err)
		return http.StatusInternalServerError
	}

	for _, t := range testCases {
		t.ProblemID = problemID
		_, err := p.testcaseRepo.Insert(ctx, t)
		if err != nil {
			logger.Error("error on insert testcase to db", err)
			return http.StatusInternalServerError
		}
	}
	return http.StatusOK
}
func (p ProblemsHandlerImp) GetTestcase(ctx context.Context, problemID int64) ([]structs.ResponseGetTestcase, int) {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "GetTestcase",
		"module": "Problems",
	})

	testCases, err := p.testcaseRepo.GetAllTestsOfProblem(ctx, problemID)

	if err != nil {
		logger.WithError(err).Error("error on get testcases from db")
		return nil, http.StatusInternalServerError
	}

	ans := make([]structs.ResponseGetTestcase, len(testCases))
	for i, t := range testCases {
		ans[i] = structs.ResponseGetTestcase{
			ID:     t.ID,
			Input:  t.Input,
			Output: t.ExpectedOutput,
		}
	}

	return ans, http.StatusOK
}
