package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/ocontest/backend/internal/db/repos"
	"net/http"

	"github.com/ocontest/backend/internal/jwt"
	"github.com/ocontest/backend/internal/otp"
	"github.com/ocontest/backend/pkg"
	"github.com/ocontest/backend/pkg/aes"
	"github.com/ocontest/backend/pkg/configs"
	"github.com/ocontest/backend/pkg/smtp"
	"github.com/ocontest/backend/pkg/structs"
	"github.com/sirupsen/logrus"
)

type AuthHandler interface {
	RegisterUser(ctx context.Context, request structs.RegisterUserRequest) (structs.RegisterUserResponse, int)
	VerifyEmail(ctx context.Context, userID int64, otp string) int
	LoginWithPassword(ctx context.Context, email, password string) (structs.AuthenticateResponse, int)
	RenewToken(ctx context.Context, userID int64, token_type string, fullRefresh bool) (structs.AuthenticateResponse, int)
	RequestLoginWithOTP(ctx context.Context, userID string) int
	LoginWithOTP(ctx context.Context, email, otpCode string) (structs.AuthenticateResponse, int)
	EditUser(ctx context.Context, request structs.RequestEditUser) int
	ParseAuthToken(ctx context.Context, token string) (int64, string, error)
	GetUser(ctx context.Context, userID int64, getPrivate bool) (structs.ReponeGetUser, int)
}

type AuthHandlerImp struct {
	authRepo   repos.UsersRepo
	jwtHandler jwt.TokenGenerator
	smtpSender smtp.Sender
	configs    *configs.OContestConf
	aesHandler aes.AESHandler
	otpStorage otp.OTPHandler
}

func NewAuthHandler(
	authRepo repos.UsersRepo, jwtHandler jwt.TokenGenerator,
	smtpSender smtp.Sender, config *configs.OContestConf,
	aesHandler aes.AESHandler, otpStorage otp.OTPHandler) AuthHandler {
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

	userIDStr := fmt.Sprintf("%d", user.ID)
	otpCode, err := p.otpStorage.Gen(ctx, userIDStr, "register")
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
	if err := p.otpStorage.Check(ctx, userIDStr, "register", token); err != nil {
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

func (p *AuthHandlerImp) LoginWithPassword(ctx context.Context, email, password string) (structs.AuthenticateResponse, int) {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "LoginWithPassword",
		"module": "auth",
	})

	userInDB, err := p.authRepo.GetByEmail(ctx, email)
	if err != nil {
		logger.Errorf("error on getting user from db. error: %v", err)
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
	encPassword, err := p.aesHandler.Encrypt(password)
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

func (p *AuthHandlerImp) RequestLoginWithOTP(ctx context.Context, email string) (status int) {
	logger := pkg.Log.WithField("method", "RequestLoginWithOTP")

	user, err := p.authRepo.GetByEmail(ctx, email)
	if err != nil {
		logger.Error("error on request otp login: ", err)
		return http.StatusInternalServerError
	}
	status = http.StatusInternalServerError
	userIDStr := fmt.Sprintf("%d", user.ID)
	otpCode, err := p.otpStorage.Gen(ctx, userIDStr, "login")

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

	status = http.StatusOK
	return
}

func (p *AuthHandlerImp) LoginWithOTP(ctx context.Context, email, otpCode string) (ans structs.AuthenticateResponse, status int) {

	logger := pkg.Log.WithField("method", "VerifyEmail")
	user, err := p.authRepo.GetByEmail(ctx, email)
	status = http.StatusInternalServerError
	if err != nil {
		logger.Error("error on getting user from db: ", err)
		return
	}
	userIDStr := fmt.Sprintf("%d", user.ID)
	if err := p.otpStorage.Check(ctx, userIDStr, "login", otpCode); err != nil {
		if errors.Is(err, pkg.ErrForbidden) || errors.Is(err, pkg.ErrNotFound) {
			status = http.StatusForbidden
			return
		}
		logger.Error("error on check register otp", err)
		return
	}

	if err := p.authRepo.VerifyUser(ctx, user.ID); err != nil {
		logger.Error("error on verifying user", err)
		return
	}

	accessToken, refreshToken, err := p.genAuthToken(user.ID)
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

	encryptedPassword, err := a.aesHandler.Encrypt(request.Password)
	if err != nil {
		logger.Error("error on encrypting password: ", err)
		return http.StatusInternalServerError
	}

	user := structs.User{
		ID:                request.UserID,
		Username:          request.Username,
		Email:             request.Email,
		EncryptedPassword: encryptedPassword,
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

func (a *AuthHandlerImp) GetUser(ctx context.Context, userID int64, getPrivate bool) (ans structs.ReponeGetUser, status int) {
	logger := pkg.Log.WithFields(logrus.Fields{
		"method": "GetUser",
		"module": "auth",
	})

	user, err := a.authRepo.GetByID(ctx, userID)
	if err != nil {
		logger.Error("error on getting user: ", err)
		status = http.StatusInternalServerError
		return
	}

	ans.Username = user.Username
	if getPrivate {
		ans.Email = user.Email
	}

	status = http.StatusOK
	return
}
