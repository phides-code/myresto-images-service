package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator"
)

type ResponseStructure struct {
	Data         any     `json:"data"`
	ErrorMessage *string `json:"errorMessage"` // can be string or nil
}

var validate *validator.Validate = validator.New()

var headers = map[string]string{
	"Access-Control-Allow-Origin":  OriginURL,
	"Access-Control-Allow-Headers": "Content-Type, X-CF-Token, x-admin-key",
}

type ImageData struct {
	Image   string `json:"image"`
	FileExt string `json:"fileExt"`
}

func router(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("router() received " + req.HTTPMethod + " request")

	if !localMode {
		awsCfToken := os.Getenv("AWS_CF_TOKEN")

		if awsCfToken == "" {
			return serverError(errors.New("Error reading environment variable"))
		}

		providedCfToken := req.Headers["X-CF-Token"]

		if providedCfToken != awsCfToken {
			return clientError(http.StatusUnauthorized)
		}
	}

	switch req.HTTPMethod {
	case "POST":
		return handleAdminOnly(req, processPost)
	case "DELETE":
		return handleAdminOnly(req, processDelete)
	case "OPTIONS":
		return processOptions()
	default:
		log.Println("router() error parsing HTTP method")
		return clientError(http.StatusMethodNotAllowed)
	}
}

func processOptions() (events.APIGatewayProxyResponse, error) {
	additionalHeaders := map[string]string{
		"Access-Control-Allow-Methods": "OPTIONS, POST, DELETE",
		"Access-Control-Max-Age":       "3600",
	}
	mergedHeaders := mergeHeaders(headers, additionalHeaders)

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    mergedHeaders,
	}, nil
}
func processPost(
	req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("running processPost()")

	var imageData ImageData
	err := json.Unmarshal([]byte(req.Body), &imageData)
	if err != nil {
		log.Printf("Can't unmarshal body: %v", err)
		return clientError(http.StatusBadRequest)
	}

	err = validate.Struct(&imageData)
	if err != nil {
		log.Printf("Invalid body: %v", err)
		return clientError(http.StatusBadRequest)
	}

	// Decode the base64-encoded image data
	imageBytes, err := base64.StdEncoding.DecodeString(imageData.Image)
	if err != nil {
		log.Println("Error decoding base64 image:", err)
		return clientError(http.StatusBadRequest)
	}

	contentType := http.DetectContentType(imageBytes)

	// Check if the uploaded file is an image
	if !strings.HasPrefix(contentType, "image/") {
		log.Println("Uploaded file is not an image")
		return clientError(http.StatusBadRequest)
	}

	fileExt := imageData.FileExt
	image := bytes.NewReader(imageBytes)

	fileName, err := myS3.UploadFile(image, fileExt, contentType)

	if err != nil {
		return serverError(err)
	}

	response := ResponseStructure{
		Data:         &fileName,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Println("processPost() error running json.Marshal")
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Body:       string(responseJson),
		Headers:    headers,
	}, nil
}

func processDelete(
	req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	id, idPresent := req.PathParameters["id"]
	if !idPresent {
		log.Println("processDelete() error reading req.PathParameters[\"id\"]")
		return clientError(http.StatusBadRequest)
	}
	log.Println("running processDelete on id: " + id)

	_, err := myS3.DeleteObject(id)
	if err != nil {
		log.Printf("Couldn't delete object from bucket: %v", err)
		return serverError(err)
	}

	response := ResponseStructure{
		Data:         &id,
		ErrorMessage: nil,
	}

	responseJson, err := json.Marshal(response)
	if err != nil {
		log.Println("processDelete() error running json.Marshal")
		return serverError(err)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Body:       string(responseJson),
		Headers:    headers,
	}, nil
}
