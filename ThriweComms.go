package ThriweComms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"context"
	"flag"

	"github.com/labstack/echo/v4"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// SQSSendMessageAPI defines the interface for the GetQueueUrl and SendMessage functions.
// We use this interface to test the functions using a mocked service.
type SQSSendMessageAPI interface {
	GetQueueUrl(ctx context.Context,
		params *sqs.GetQueueUrlInput,
		optFns ...func(*sqs.Options)) (*sqs.GetQueueUrlOutput, error)

	SendMessage(ctx context.Context,
		params *sqs.SendMessageInput,
		optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// GetQueueURL gets the URL of an Amazon SQS queue.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a GetQueueUrlOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to GetQueueUrl.
func GetQueueURL(c context.Context, api SQSSendMessageAPI, input *sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error) {
	return api.GetQueueUrl(c, input)
}

// SendMsg sends a message to an Amazon SQS queue.
// Inputs:
//     c is the context of the method call, which includes the AWS Region.
//     api is the interface that defines the method call.
//     input defines the input arguments to the service call.
// Output:
//     If success, a SendMessageOutput object containing the result of the service call and nil.
//     Otherwise, nil and an error from the call to SendMessage.
func SendMsg(c context.Context, api SQSSendMessageAPI, input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return api.SendMessage(c, input)
}

func Echo(Host string, Token string) string {
	req, err := http.NewRequest("GET", "https://"+Host+".i.tgcloud.io:9000/echo", nil)
	if err != nil {
		return err.Error()
	}
	req.Header.Set("Authorization", "Bearer "+Token)
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err.Error()
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err.Error()
	}
	sb := string(body)
	response.Body.Close()
	var jsonMap map[string]interface{}
	json.Unmarshal([]byte(sb), &jsonMap)
	mess := jsonMap["message"]
	return fmt.Sprintf("%v", mess)
}

func SendMail(c echo.Context, cfg aws.Config, userData, projectId, templateCode, requestId string) (string, error) {
	queue := flag.String("q", "sms-mail", "The name of the queue")
	flag.Parse()
	messageBody := `,{
		"userData": {` + userData + `},
		"projectId": ` + projectId + `,
		"templateCode": ` + templateCode + `,
		"requestId": ` + requestId + `
	}`
	if *queue == "" {
		return "", fmt.Errorf("you must supply the name of a queue (-q QUEUE)")
	}

	client := sqs.NewFromConfig(cfg)

	// Get URL of queue
	gQInput := &sqs.GetQueueUrlInput{
		QueueName: queue,
	}

	result, err := GetQueueURL(context.TODO(), client, gQInput)
	if err != nil {
		log.Print("Got an error getting the queue URL:")
		return "", err
	}

	queueURL := result.QueueUrl

	sMInput := &sqs.SendMessageInput{
		DelaySeconds: 1,
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Title": {
				DataType:    aws.String("String"),
				StringValue: aws.String("Mail"),
			},
			"Author": {
				DataType:    aws.String("String"),
				StringValue: aws.String("ThriweComms library"),
			},
			"RequestId": {
				DataType:    aws.String("Number"),
				StringValue: aws.String(requestId),
			},
		},
		MessageBody: aws.String(messageBody),
		QueueUrl:    queueURL,
	}

	resp, err := SendMsg(context.TODO(), client, sMInput)
	if err != nil {
		log.Println("Got an error sending the message:")
		return "", err
	}

	log.Println("Sent message with ID: " + *resp.MessageId)
	return *resp.MessageId, nil
}
