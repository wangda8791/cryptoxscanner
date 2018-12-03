package pkg

import (
	"gitlab.com/crankykernel/cryptoxscanner/log"
	"strconv"
)

var BuildNumber string

func BuildNumberAsInt() int64 {
	buildNumber, err := strconv.ParseInt(BuildNumber, 10, 64)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"BuildNumber": BuildNumber,
		}).Errorf("Failed to convert BuildNumber to int64")
		return 0
	}
	return buildNumber
}