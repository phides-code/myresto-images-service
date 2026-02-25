package main

import (
	"encoding/json"
	"log"
	"net/http"

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
