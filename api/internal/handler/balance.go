package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
)

var errInvalidMobile = errors.New("mobile format is not valid")

func (h *Handler) Balance() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := mux.Vars(r)["user"]
		if user == "" {
			w.WriteHeader(http.StatusNotFound)
			// w.Write([]byte(service.ErrCampaginNotFound.Error()))

			return
		}

		u, err := newMobile(user)
		if err != nil {
			h.logger.Errorf("handler: Balance() >> %w", err)
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		account, err := h.service.GetBalance(r.Context(), u)
		if err != nil {
			h.logger.Errorf("handler: Balance() >> %w", err)

		}

		b, err := json.Marshal(account)
		if err != nil {
			h.logger.Errorf("handler: GetRedeemers() >> %w", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func newMobile(v string) (string, error) {
	if v == "" {
		return "", errInvalidMobile
	}

	v = sanitizeMobile(v)

	err := validateMobile(v)
	if err != nil {
		return "", err
	}

	return v, nil
}

func sanitizeMobile(v string) string {
	re := regexp.MustCompile(`[\D]`)
	v = re.ReplaceAllString(v, "")
	v = strings.TrimLeft(v, "0")

	re = regexp.MustCompile("^98")
	v = re.ReplaceAllString(v, "")
	v = strings.TrimLeft(v, "0")

	if v == "" {
		return ""
	}

	return "0" + v
}

func validateMobile(v string) error {
	re := regexp.MustCompile(`^0?9\d{9}$`)
	if re.MatchString(v) {
		return nil
	}

	return errInvalidMobile
}
