package errors

import "errors"

var Is = errors.Is

const (
	MsgNotFound      = "not found"
	MsgWrongPassword = "wrong password"

	MsgUserNotFound         = "user not found"
	MsgUserPasswordNotFound = "not found user password"
	MsgUserEmailNotFound    = "user with the given email is not found"

	MsgProductNotFound = "product not found"
	MsgPageNotFound    = "page not found"
	MsgSettingNotFound = "setting not found"
)

var (
	ErrNotFound      = errors.New(MsgNotFound)
	ErrWrongPassword = errors.New(MsgWrongPassword)

	ErrUserNotFound         = errors.New(MsgUserNotFound)
	ErrUserPasswordNotFound = errors.New(MsgUserPasswordNotFound)
	ErrUserEmailNotFound    = errors.New(MsgUserEmailNotFound)

	ErrProductNotFound = errors.New(MsgProductNotFound)
	ErrPageNotFound    = errors.New(MsgPageNotFound)
	ErrSettingNotFound = errors.New(MsgSettingNotFound)
)
