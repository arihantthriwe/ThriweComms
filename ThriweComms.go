package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"context"
	"flag"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/random"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go"
	"github.com/carlmjohnson/requests"
)

// ThriweCommsAPI defines the interface for the SendMail and SendSms functions.
type ThriweCommsAPI interface {
	SendMail(c echo.Context, recipient, emailMessageBody, requestId *string) (string, error)
	SendSms(c echo.Context, countryCode, mobileNumber, smsMessageBody, requestId *string) (string, error)
}

type thriweCommsAPI struct {
	client       *sqs.Client
	projectCode  *string
	sqsQueueName *string
}

// NewThriweCommsAPI creates a new object for ThriweCommsAPI just pass-in the aws.config, projectCode()
func NewThriweCommsAPI(client *sqs.Client, projectCode, sqsQueueName *string) ThriweCommsAPI {
	return &thriweCommsAPI{client: client, projectCode: projectCode, sqsQueueName: sqsQueueName}
}

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
func requestId() *string { return flag.String("rId", random.String(32), "request id") }
type TrackerRequestType struct{
	MessageBody string `json:"messageBody"`
	RequestId string `json:"requestId"`
	Status string	`json:"status"`
	CommsId string `json:"commsId"`
	ProjectCode string `json:"projectCode"`
}
type TrackerResponseType struct{
	ObjectId string `json:"objectId"`
	CreatedAt time.Time `json:"createdAt"`
}
func main() {
	var c echo.Context
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	sqsClient := sqs.NewFromConfig(cfg)
	emailMessageBody := flag.String("emb", `<div style="font-family:Helvetica,Arial,sans-serif;min-width:1000px;overflow:auto;line-height:2"><div style="margin:50px auto;width:70%;padding:20px 0"><div style="border-bottom:1px solid #eee"><a href="" style="font-size:1.4em;color:#00466a;text-decoration:none;font-weight:600">Your Brand</a></div><p style="font-size:1.1em">Hi,</p><p>Thank you for choosing Your Brand. Use the following OTP to complete your Sign Up procedures. OTP is valid for 5 minutes</p><h2 style="background:#00466a;margin:0 auto;width:max-content;padding:0 10px;color:#fff;border-radius:4px">123456</h2><p style="font-size:.9em">Regards,<br>Your Brand</p><hr style="border:none;border-top:1px solid #eee"><div style="float:right;padding:8px 0;color:#aaa;font-size:.8em;line-height:1;font-weight:300"><p>Your Brand Inc</p><p>1600 Amphitheatre Parkway</p><p>California</p></div></div></div>`, "email message body")
	smsMessageBody := flag.String("smb", `'Hi, \n123456 is your OTP to verify your mobile number. OTP Code is valid for 10 minutes. THRIWE'`, "sms message body")
	queue := flag.String("q", "sms-mail", "The name of the queue")
	projectCode := flag.String("p", "FAB-ONE", "project code")
	recipient := flag.String("r", "arihant.jain@thriwe.com", "recipient")
	countryCode := flag.String("c", "+91", "country code")
	mobileNumber := flag.String("m", "8630771592", "mobile number")
	thriweCommsAPI := NewThriweCommsAPI(sqsClient, projectCode, queue)
	thriweCommsAPI.SendMail(c, recipient, emailMessageBody, requestId())
	thriweCommsAPI.SendSms(c, countryCode, mobileNumber, smsMessageBody, requestId())
}
func (t *thriweCommsAPI) SendMail(c echo.Context, recipient, emailMessageBody, requestId *string) (string, error) {
	flag.Parse()
	if *t.sqsQueueName == "" {
		return "", fmt.Errorf("you must supply the name of a queue (-q QUEUE)")
	}
	if *emailMessageBody == "" {
		return "", fmt.Errorf("you must supply the message body (-mb MESSAGE-BODY)")
	}
	if *requestId == "" {
		return "", fmt.Errorf("you must supply the request id (-rID REQUEST-ID)")
	}

	// Get URL of queue
	gQInput := &sqs.GetQueueUrlInput{
		QueueName: t.sqsQueueName,
	}
	body := TrackerRequestType{MessageBody: *emailMessageBody, RequestId: *requestId, Status: "INITIATED", CommsId: "1", ProjectCode: *t.projectCode}
	var responseTracker TrackerResponseType
	errTrackerRequest := requests.
		URL("https://dev-fab-api-gateway.thriwe.com/parse/classes/tracker").
		Header("X-Parse-Application-Id", "DEV_APPLICATION_ID").
		Header("X-Parse-Master-Key", "DEV_MASTER_KEY").
		BodyJSON(&body).
		ToJSON(&responseTracker).
		Fetch(context.Background())
	if errTrackerRequest != nil {
		log.Println(errTrackerRequest)
		return "", fmt.Errorf("got an error creating initializer tracker for ThriweComms request")
	}
	result, err := GetQueueURL(context.TODO(), t.client, gQInput)
	if err != nil {
		log.Print("Got an error getting the queue URL:")
		return "", err
	}

	queueURL := result.QueueUrl
	queueBody := `{"commsId":"1","projectCode":"`+*t.projectCode+`","recipient":"`+*recipient+`","requestId":"`+*requestId+`","messageBody":"`+*emailMessageBody+`","trackerObjectId":"`+responseTracker.ObjectId+`"}`
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
			}},
		MessageBody: aws.String(queueBody),
		QueueUrl:    queueURL,
	}

	resp, err := SendMsg(context.TODO(), t.client, sMInput)
	if err != nil {
		var oe *smithy.OperationError
		if errors.As(err, &oe) {
			log.Printf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
			if err != nil {
				return "", fmt.Errorf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
			}
			return "", fmt.Errorf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
		}
		return "", err
	}

	log.Println("Sent message with ID: " + *resp.MessageId)
	return *resp.MessageId, nil
}
func (t *thriweCommsAPI) SendSms(c echo.Context, countryCode, mobileNumber, smsMessageBody, requestId *string) (string, error) {
	flag.Parse()
	if *t.sqsQueueName == "" {
		return "", fmt.Errorf("you must supply the name of a queue (-q QUEUE)")
	}
	if *smsMessageBody == "" {
		return "", fmt.Errorf("you must supply the message body (-mb MESSAGE-BODY)")
	}
	if *requestId == "" {
		return "", fmt.Errorf("you must supply the request id (-rID REQUEST-ID)")
	}

	// Get URL of queue
	gQInput := &sqs.GetQueueUrlInput{
		QueueName: t.sqsQueueName,
	}
	body := TrackerRequestType{MessageBody: *smsMessageBody, RequestId: *requestId, Status: "INITIATED", CommsId: "2", ProjectCode: *t.projectCode}
	var responseTracker TrackerResponseType
	errTrackerRequest := requests.
		URL("https://dev-fab-api-gateway.thriwe.com/parse/classes/tracker").
		Header("X-Parse-Application-Id", "DEV_APPLICATION_ID").
		Header("X-Parse-Master-Key", "DEV_MASTER_KEY").
		BodyJSON(&body).
		ToJSON(&responseTracker).
		Fetch(context.Background())
	if errTrackerRequest != nil {
		log.Println(errTrackerRequest)
		return "", fmt.Errorf("got an error creating initializer tracker for ThriweComms request")
	}
	result, err := GetQueueURL(context.TODO(), t.client, gQInput)
	if err != nil {
		log.Print("Got an error getting the queue URL:")
		return "", err
	}

	queueURL := result.QueueUrl
	queueBody := `{"commsId":"2","projectCode":"`+*t.projectCode+`","countryCode":"`+*countryCode+`","mobileNumber":"`+*mobileNumber+`","requestId":"`+*requestId+`","messageBody":"`+*smsMessageBody+`","trackerObjectId":"`+responseTracker.ObjectId+`"}`
	sMInput := &sqs.SendMessageInput{
		DelaySeconds: 1,
		MessageAttributes: map[string]types.MessageAttributeValue{
			"Title": {
				DataType:    aws.String("String"),
				StringValue: aws.String("SMS"),
			},
			"Author": {
				DataType:    aws.String("String"),
				StringValue: aws.String("ThriweComms library"),
			}},
		MessageBody: aws.String(queueBody),
		QueueUrl:    queueURL,
	}

	resp, err := SendMsg(context.TODO(), t.client, sMInput)
	if err != nil {
		var oe *smithy.OperationError
		if errors.As(err, &oe) {
			log.Printf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
			if err != nil {
				return "", fmt.Errorf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
			}
			return "", fmt.Errorf("failed to call service: %s, operation: %s, error: %v", oe.Service(), oe.Operation(), oe.Unwrap())
		}
		return "", err
	}

	log.Println("Sent message with ID: " + *resp.MessageId)
	return *resp.MessageId, nil
}
