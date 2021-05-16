package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Response events.APIGatewayProxyResponse

type ExchangeEventResponse struct {
	Id     string    `json:"id" validate:"required"`
	Apikey string    `json:"xapikey,omitempty" validate:"required"`
	Base   string    `json:"base" validate:"required"`
	Target string    `json:"target" validate:"required"`
	Rate   float64   `json:"rate" validate:"required"`
	Value  float64   `json:"value" validate:"required"`
	Time   time.Time `json:"time" validate:"required"`
}

func ToJSON(o interface{}) string {
	j, _ := json.Marshal(o)
	return string(j)
}

var dynamo *dynamodb.Client

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {

	id := event.PathParameters["id"]
	// xapikey := event.Headers["x-api-key"]

	getItemResponse, err := dynamo.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(os.Getenv("REQUESTSTABLE")),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return Response{}, err
	}
	if getItemResponse.Item == nil {
		return Response{StatusCode: http.StatusNotFound}, nil
	}

	parsedTime, err := time.Parse(time.RFC3339Nano, getItemResponse.Item["time"].(*types.AttributeValueMemberS).Value)
	parsedRate, err := strconv.ParseFloat(getItemResponse.Item["rate"].(*types.AttributeValueMemberN).Value, 64)
	parsedValue, err := strconv.ParseFloat(getItemResponse.Item["value"].(*types.AttributeValueMemberN).Value, 64)
	response := ExchangeEventResponse{
		Id:     getItemResponse.Item["id"].(*types.AttributeValueMemberS).Value,
		Apikey: getItemResponse.Item["xapikey"].(*types.AttributeValueMemberS).Value,
		Base:   getItemResponse.Item["base"].(*types.AttributeValueMemberS).Value,
		Target: getItemResponse.Item["target"].(*types.AttributeValueMemberS).Value,
		Rate:   parsedRate,
		Value:  parsedValue,
		Time:   parsedTime,
	}

	log.Println(ToJSON(response))
	// clear xapikey for response
	response.Apikey = ""

	var buf bytes.Buffer

	body, err := json.Marshal(response)
	if err != nil {
		log.Printf("Couldn't marshal into json: %v", response)
		return Response{}, err
	}
	json.HTMLEscape(&buf, body)

	resp := Response{
		StatusCode:      200,
		IsBase64Encoded: false,
		Body:            buf.String(),
	}

	return resp, nil
}

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	dynamo = dynamodb.NewFromConfig(cfg)
}

func main() {
	lambda.Start(Handler)
}
