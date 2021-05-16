package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/segmentio/ksuid"
)

type Response events.APIGatewayProxyResponse

type ExchangeResponse struct {
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

var secrets *secretsmanager.Client
var dynamo *dynamodb.Client

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {

	baseCurrency := event.PathParameters["base"]
	targetCurrency := event.PathParameters["target"]
	amount, err := strconv.ParseFloat(event.PathParameters["amount"], 64)
	id := ksuid.New()
	xapikey := event.Headers["x-api-key"]

	if err != nil {
		log.Printf("%s: %s couldn't be parsed into float.", id.String(), event.PathParameters["amount"])
		log.Println(err)
		return Response{StatusCode: 401, Body: err.Error()}, nil
	}

	getSecretValueResponse, err := secrets.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(os.Getenv("APIKEY"))})
	if err != nil {
		return Response{}, err
	}

	log.Printf("API Key: %s", *getSecretValueResponse.SecretString)

	// get forex rate from exchange reates api for base -> target
	// value = amount * rate
	// generate id with ksuid

	rate := 0.94
	response := ExchangeResponse{id.String(), xapikey, baseCurrency, targetCurrency, rate, amount * rate, time.Now()}
	dynamo.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(os.Getenv("REQUESTSTABLE")),
		Item: map[string]types.AttributeValue{
			"id":      &types.AttributeValueMemberS{Value: response.Id},
			"xapikey": &types.AttributeValueMemberS{Value: response.Apikey},
			"base":    &types.AttributeValueMemberS{Value: response.Base},
			"target":  &types.AttributeValueMemberS{Value: response.Target},
			"rate":    &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", response.Rate)},
			"value":   &types.AttributeValueMemberN{Value: fmt.Sprintf("%f", response.Value)},
			"time":    &types.AttributeValueMemberS{Value: response.Time.String()},
		},
	})
	if err != nil {
		return Response{}, err
	}
	log.Println(ToJSON(response))
	// clear xapikey for response
	response.Apikey = ""

	var buf bytes.Buffer

	body, err := json.Marshal(response)
	if err != nil {
		log.Printf("%s: Couldn't marshal into json: %v", id.String(), response)
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
	secrets = secretsmanager.NewFromConfig(cfg)
	dynamo = dynamodb.NewFromConfig(cfg)
}

func main() {
	lambda.Start(Handler)
}
