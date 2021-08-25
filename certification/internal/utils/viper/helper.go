package viper

import (
	"fmt"

	"github.com/redhat-openshift-ecosystem/openshift-preflight/certification/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func GetString(key string) (string, error) {
	val := viper.GetString(key)
	if len(val) == 0 {
		log.Error(fmt.Sprintf("unable to fetch %s from viper. The value is empty.", key))
		return "", errors.ErrNoValueFoundInViper
	}
	return val, nil
}
