package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mateo/ms-gestor-de-info/apps/api/internal/api"
)

func main() {
	mux := http.NewServeMux()
	humaAPI := api.NewHumaAPI(mux, api.Options{PublishDocs: true}, api.Dependencies{})
	raw, err := humaAPI.OpenAPI().YAML()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generando OpenAPI YAML: %v\n", err)
		os.Exit(1)
	}
	_, _ = os.Stdout.Write(raw)
}

