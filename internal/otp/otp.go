package otp

import (
	"fmt"
	"github.com/ocontest/backend/pkg"
	"github.com/ocontest/backend/pkg/kvstorages"
	"math/rand"
)

type OTPHandler interface {
	GenOTP(userID, typ string) (string, error)
	CheckOTP(userID, typ, val string) error
}

func NewOTPHandler(storage kvstorages.KVStorage) OTPHandler {
	return &OTPHandlerImp{
		storage: storage,
	}
}

// TODO: use redis or memcache instead of map
type OTPHandlerImp struct {
	storage kvstorages.KVStorage
}

func (o *OTPHandlerImp) GenOTP(userID, typ string) (string, error) {
	k := fmt.Sprintf("%s/%s", typ, userID)
	v := fmt.Sprintf("%06d", rand.Intn(1000000))
	return v, o.storage.Save(k, v)
}

func (o *OTPHandlerImp) CheckOTP(userID, typ, val string) error {
	k := fmt.Sprintf("%s/%s", typ, userID)
	ans, err := o.storage.Get(k)
	if err != nil {
		return err
	}
	if ans != val {
		return pkg.ErrForbidden
	}
	return nil
}
