package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/google/uuid"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

type blogItem struct {
	BlogId    string `json:"blogId"`
	Author    string `json:"author"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

type errorResponse struct {
	Message string `json:"message"`
	Detail  string `json:"detail"`
}

var awsRegion string
var blogTableName string

func main() {
	// Read table names and AWS region from Environment
	awsRegion = os.Getenv("AWS_REGION")
	if awsRegion == "" {
		log.Fatal("Missing AWS Region")
	}
	blogTableName = os.Getenv("BLOG_TABLE_NAME")
	if blogTableName == "" {
		log.Fatal("missing BLOG_TABLE_NAME")
	}

	// Start lambda handler
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	// Handle requests based on path
	path := strings.ToLower(req.Path)
	switch path {
	case "/blogs":
		return blogsHandler(req)
	default:
		if strings.Index(path, "/blogs/") == 0 {
			return blogHandler(req)
		}
	}
	return unhandledPath(req)
}

func blogsHandler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		return getBlogs(req)
	case "POST":
		return createBlog(req)
	default:
		return unhandledMethod(req)
	}
}

func blogHandler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		return getBlog(req)
	default:
		return unhandledMethod(req)
	}
}

func getBlogs(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	resp, err := svc.Scan(&dynamodb.ScanInput{
		TableName: aws.String(blogTableName),
		Select:    aws.String(dynamodb.SelectAllAttributes),
	})
	if err != nil {
		result := errorResponse{
			Message: "Internal server error",
			Detail:  err.Error(),
		}
		return apiResponse(http.StatusInternalServerError, result)
	}

	res := []blogItem{}

	for _, item := range resp.Items {
		blog := blogItem{}
		err = dynamodbattribute.UnmarshalMap(item, &blog)
		if err == nil {
			res = append(res, blog)
		}
	}
	return apiResponse(http.StatusOK, res)
}

func createBlog(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {

	var newBlog blogItem

	err := json.Unmarshal([]byte(req.Body), &newBlog)
	if err != nil || newBlog.Author == "" || newBlog.Title == "" || newBlog.Body == "" {
		result := errorResponse{
			Message: "Invalid request",
			Detail:  "Request body is invalid. Please see the documentation.",
		}
		return apiResponse(http.StatusBadRequest, result)
	}

	newBlog.BlogId = uuid.NewString()
	newBlog.CreatedAt = time.Now().Format(time.RFC3339)

	err = storeBlog(newBlog)
	if err != nil {
		result := errorResponse{
			Message: "Unable to store new blog",
			Detail:  err.Error(),
		}
		return apiResponse(http.StatusInternalServerError, result)
	}

	return apiResponse(http.StatusOK, newBlog)
}

func getBlog(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	blogId := strings.Replace(req.Path, "/blogs/", "", 1)

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String(blogTableName),
		Key: map[string]*dynamodb.AttributeValue{
			"blogId": {
				S: aws.String(blogId),
			},
		},
	})

	if err != nil {
		result := errorResponse{
			Message: "Unable to get blog",
			Detail:  err.Error(),
		}
		return apiResponse(http.StatusInternalServerError, result)
	}

	if result.Item == nil {
		result := errorResponse{
			Message: "Not found",
			Detail:  req.Path + " does not exist",
		}
		return apiResponse(http.StatusNotFound, result)
	}

	blog := blogItem{}
	err = dynamodbattribute.UnmarshalMap(result.Item, &blog)
	if err != nil {
		result := errorResponse{
			Message: "Unable to get blog",
			Detail:  err.Error(),
		}
		return apiResponse(http.StatusInternalServerError, result)
	}

	return apiResponse(http.StatusOK, blog)
}

func storeBlog(newBlog blogItem) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	svc := dynamodb.New(sess)

	nu, err := dynamodbattribute.MarshalMap(newBlog)
	if err != nil {
		return err
	}

	input := &dynamodb.PutItemInput{
		Item:      nu,
		TableName: aws.String(blogTableName),
	}

	_, err = svc.PutItem(input)

	return err
}

func unhandledMethod(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	result := errorResponse{
		Message: fmt.Sprintf("%s method is not supported for %s path", req.HTTPMethod, req.Path),
		Detail:  "Try again",
	}
	return apiResponse(http.StatusNotFound, result)
}

func unhandledPath(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	result := errorResponse{
		Message: fmt.Sprintf("Invalid path %s", req.Path),
		Detail:  "Try valid paths",
	}
	return apiResponse(http.StatusNotFound, result)
}

func apiResponse(status int, body interface{}) (*events.APIGatewayProxyResponse, error) {
	resp := events.APIGatewayProxyResponse{Headers: map[string]string{"Content-Type": "application/json"}}
	resp.StatusCode = status
	stringBody, _ := json.Marshal(body)
	resp.Body = string(stringBody)
	return &resp, nil
}
