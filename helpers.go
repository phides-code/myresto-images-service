package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
)

func clientError(status int) (events.APIGatewayProxyResponse, error) {

	errorString := http.StatusText(status)

	response := ResponseStructure{
		Data:         nil,
		ErrorMessage: &errorString,
	}

	responseJson, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		Body:       string(responseJson),
		StatusCode: status,
		Headers:    headers,
	}, nil
}

func serverError(err error) (events.APIGatewayProxyResponse, error) {
	log.Println(err.Error())

	errorString := http.StatusText(http.StatusInternalServerError)

	response := ResponseStructure{
		Data:         nil,
		ErrorMessage: &errorString,
	}

	responseJson, _ := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		Body:       string(responseJson),
		StatusCode: http.StatusInternalServerError,
		Headers:    headers,
	}, nil
}

func mergeHeaders(baseHeaders, additionalHeaders map[string]string) map[string]string {
	mergedHeaders := make(map[string]string)
	for key, value := range baseHeaders {
		mergedHeaders[key] = value
	}
	for key, value := range additionalHeaders {
		mergedHeaders[key] = value
	}
	return mergedHeaders
}

func isValidAdminKey(providedAdminKey string) bool {
	adminKey := os.Getenv("ADMIN_KEY")

	if adminKey == "" {
		log.Println("isValidAdminKey() got blank adminKey")
		return false
	}

	return providedAdminKey == adminKey
}

func handleAdminOnly(
	req events.APIGatewayProxyRequest,
	handler func(events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error),
) (events.APIGatewayProxyResponse, error) {
	var providedAdminKey string

	if localMode {
		providedAdminKey = req.Headers["X-Admin-Key"]
	} else {
		providedAdminKey = req.Headers["x-admin-key"]
	}

	if !isValidAdminKey(providedAdminKey) {
		log.Println("handleAdminOnly() error: AdminKey mismatch")
		return clientError(http.StatusUnauthorized)
	}
	return handler(req)
}
