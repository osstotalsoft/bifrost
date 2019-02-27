package filters

import (
	"net/http"
)

func AuthorizationFilter() func(request *http.Request) error {
	return func(request *http.Request) error {
		//log.Info("AuthorizationFilter")
		return nil
	}
}
