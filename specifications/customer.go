package specifications

import (
	"net/http"
	"testing"
)

type CreateCustomerAction interface {
	CreateCustomer(args ...string) (int, string, error)
}

func SuccessfullyCreateCustomer(t *testing.T, driver CreateCustomerAction) {
	status, got, err := driver.CreateCustomer(`tag: "lyan"`, `name: "Lyan"`, `password: "123456"`)
	assertNoError(t, err)
	assertHTTPStatus(t, status, http.StatusCreated)
	want := map[string]any{
		"tag":  "lyan",
		"name": "Lyan",
	}
	data := extractData(t, got)
	assertCanBeTheSame(t, data, want, "didn't get expected customer")
	if _, ok := data["id"]; !ok {
		t.Error("doesn't contain id in data")
	}
}
