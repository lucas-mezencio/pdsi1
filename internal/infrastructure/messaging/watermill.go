package messaging

import (
	"github.com/ThreeDotsLabs/watermill"
)

func DefaultLogger() watermill.LoggerAdapter {
	return &watermill.NopLogger{}
}
