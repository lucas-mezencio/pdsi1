package devicetoken

import "errors"

var (
    ErrInvalidToken = errors.New("devicetoken: invalid token")
    ErrNotFound     = errors.New("devicetoken: not found")
    ErrConflict     = errors.New("devicetoken: token already registered to another user")
)
