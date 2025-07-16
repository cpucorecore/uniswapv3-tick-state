package main

import (
	"github.com/avast/retry-go/v4"
	"time"
)

var (
	infiniteAttempts = retry.Attempts(0)
	retryDelay       = retry.Delay(100 * time.Microsecond)
)
