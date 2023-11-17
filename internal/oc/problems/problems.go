package problems

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"ocontest/internal/db"
	"ocontest/internal/jwt"
	"ocontest/internal/otp"
	"ocontest/pkg"
	"ocontest/pkg/aes"
	"ocontest/pkg/configs"
	"ocontest/pkg/smtp"
	"ocontest/pkg/structs"
)

type ProblemsHandler interface {
	CreateProblem(ctx context.Context, req structs.RequestCreateProblem) (structs.ResponseCreateProblem, int)
	GetProblem(ctx context.Context)
}

type ProblemsHandlerImp struct {
}

func NewAuthHandler(
	authRepo db.AuthRepo, jwtHandler jwt.TokenGenerator,
	smtpSender smtp.Sender, config *configs.OContestConf,
	aesHandler aes.AESHandler, otpStorage otp.OTPStorage) AuthHandler {
	return &AuthHandlerImp{
		authRepo:   authRepo,
		jwtHandler: jwtHandler,
		smtpSender: smtpSender,
		configs:    config,
		aesHandler: aesHandler,
		otpStorage: otpStorage,
	}
}

func (p *AuthHandlerImp) RegisterUser(ctx context.Context, reqData structs.RegisterUserRequest) (ans structs.RegisterUserResponse, status int) {
	logger := pkg.Log.WithField("method", "RegisterUser")

	encryptedPassword, err := p.aesHandler.Encrypt(reqData.Password)
	if err != nil {
		logger.Error("error on encrypting password", err)
		status = 503
		ans.Message = "something went wrong, please try again later."
		return
	}

	var user structs.User
	user, err = p.authRepo.GetByUsername(ctx, reqData.Username)
	if err != nil {
		user = structs.User{
			Username:          reqData.Username,
			EncryptedPassword: encryptedPassword,
			Email:             reqData.Email,
			Verified:          false,
		}

		userID, newErr := p.authRepo.InsertUser(ctx, user)
		if newErr != nil {
			logger.Errorf("couldn't insert user in database, error on get: %v, error on insert: %v", err, newErr)
			status = 503
			ans.Message = "something went wrong, please try again later."
			return
		}
		user.ID = userID
	}

	otpCode, err := p.otpStorage.GenRegisterOTP(fmt.Sprintf("%d", user.ID))
	if err != nil {
		logger.Error("error on generating otp", err)
		status = 503
		ans.Message = "something went wrong, please try again later."
		return
	}

	validateEmailMessage := p.genEmailMessage(user, otpCode, Register)
	err = p.smtpSender.SendEmail(reqData.Email, "Welcome to OContest", validateEmailMessage)
	if err != nil {
		logger.Error("error on sending email", err)
		status = 503
		err = pkg.ErrInternalServerError
		return
	}

	ans = structs.RegisterUserResponse{
		Ok:      true,
		UserID:  user.ID,
		Message: "Sent Verification email",
	}
	return
}

func (p *AuthHandlerImp) VerifyEmail(ctx context.Context, userID int64, token string) int {

	logger := pkg.Log.WithField("method", "VerifyEmail")
	userIDStr := fmt.Sprintf("%d", userID)
	if err := p.otpStorage.CheckRegisterOTP(userIDStr, token); err != nil {
		if errors.Is(err, pkg.ErrForbidden) {
			return http.StatusForbidden
		}
		logger.Error("error on check register otp", err)
		return http.StatusInternalServerError
	}

	if err := p.authRepo.VerifyUser(ctx, userID); err != nil {
		logger.Error("error on verifying user", err)

		return http.StatusInternalServerError
	}
	return http.StatusOK
}

func (p *AuthHandlerImp) LoginUser(ctx context.Context, request structs.LoginUserRequest) (structs.AuthenticateResponse, int) {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "LoginUser",
		"module": "auth",
	})

	userInDB, err := p.authRepo.GetByUsername(ctx, request.Username)
	if err != nil {
		logger.Error("error on getting user from db", err)
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "couldn't find user",
		}, http.StatusInternalServerError
	}
	if !userInDB.Verified {
		logger.Warning("unverified user login attempt", userInDB.Username)
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "user is not verified",
		}, http.StatusForbidden
	}
	encPassword, err := p.aesHandler.Encrypt(request.Password)
	if err != nil {
		logger.Error("error on encrypting password")
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "something went wrong",
		}, http.StatusInternalServerError
	}
	if encPassword != userInDB.EncryptedPassword {
		logger.Warning("wrong password")
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "wrong password",
		}, http.StatusUnauthorized
	}
	accessToken, refreshToken, err := p.genAuthToken(userInDB.ID)
	if err != nil {
		logger.Error("error on creating tokens", err)
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "something went wrong",
		}, http.StatusInternalServerError
	}
	return structs.AuthenticateResponse{
		Ok:           true,
		Message:      "success",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, http.StatusOK
}

func (p *AuthHandlerImp) RenewToken(ctx context.Context, userID int64, tokenType string, fullRefresh bool) (structs.AuthenticateResponse, int) {
	if tokenType != "refresh" {
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "invalid token type",
		}, http.StatusBadRequest
	}

	accessToken, refreshToken, err := p.genAuthToken(userID)
	if err != nil {
		return structs.AuthenticateResponse{
			Ok:      false,
			Message: "couldn't generate new token",
		}, http.StatusInternalServerError
	}
	if !fullRefresh {
		refreshToken = ""
	}
	return structs.AuthenticateResponse{
		Ok:           true,
		Message:      "success",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, http.StatusOK
}

func (p *AuthHandlerImp) RequestLoginWithOTP(ctx context.Context, userID int64) (status int) {
	logger := pkg.Log.WithField("method", "RequestLoginWithOTP")
	userIDStr := fmt.Sprintf("%d", userID)

	user, err := p.authRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error("error on request otp login: ", err)
		return http.StatusInternalServerError
	}
	status = http.StatusInternalServerError
	otpCode, err := p.otpStorage.GenLoginOTP(userIDStr)
	if err != nil {
		logger.Error("error on generating otp", err)
		return
	}
	validateEmailMessage := p.genEmailMessage(user, otpCode, Login)
	err = p.smtpSender.SendEmail(user.Email, "Your one time password", validateEmailMessage)
	if err != nil {
		logger.Error("error on sending email", err)
		status = 503
		err = pkg.ErrInternalServerError
		return
	}

	return
}

func (p *AuthHandlerImp) CheckLoginWithOTP(ctx context.Context, userID int64, otpCode string) (ans structs.AuthenticateResponse, status int) {

	logger := pkg.Log.WithField("method", "VerifyEmail")
	userIDStr := fmt.Sprintf("%d", userID)
	status = http.StatusInternalServerError
	if err := p.otpStorage.CheckLoginOTP(userIDStr, otpCode); err != nil {
		if errors.Is(err, pkg.ErrForbidden) {
			status = http.StatusForbidden
			return
		}
		logger.Error("error on check register otp", err)
		return
	}

	if err := p.authRepo.VerifyUser(ctx, userID); err != nil {
		logger.Error("error on verifying user", err)
		return
	}

	accessToken, refreshToken, err := p.genAuthToken(userID)
	if err != nil {
		ans = structs.AuthenticateResponse{
			Ok:      false,
			Message: "couldn't generate new token",
		}
		return
	}

	return structs.AuthenticateResponse{
		Ok:           true,
		Message:      "success",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, http.StatusOK

}

func (a *AuthHandlerImp) EditUser(ctx context.Context, request structs.RequestEditUser) int {

	logger := pkg.Log.WithField("method", "EditUser")
	user := structs.User{
		ID:                request.UserID,
		Username:          request.Username,
		Email:             request.Email,
		EncryptedPassword: request.Password,
	}
	if err := a.authRepo.UpdateUser(ctx, user); err != nil {
		logger.Error("error on update user in pg: ", err)
		return http.StatusInternalServerError
	}

	return http.StatusOK
}

func (a *AuthHandlerImp) ParseAuthToken(_ context.Context, token string) (int64, string, error) {
	return a.jwtHandler.ParseToken(token)
}
