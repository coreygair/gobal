package util

import (
	"errors"
	"strconv"
)

func ValidatePort(port string) (bool, error) {
	portAsInt, err := strconv.Atoi(port)

	if err != nil {
		return false, err
	}

	return ValidatePortInt(portAsInt)
}

func ValidatePortInt(port int) (bool, error) {
	if port < 0 || port > 65535 {
		return false, errors.New("Port number out of range")
	}

	return true, nil
}
