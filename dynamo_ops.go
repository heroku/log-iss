package logiss

import (
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/dynamodb"
)

// FIXME: Add condition to not put user when the username already exists
func NewUserInput(table, user, pwd, note string) dynamodb.PutItemInput {
	return dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item: map[string]dynamodb.AttributeValue{
			"UserName": dynamodb.AttributeValue{
				S: aws.String(user),
			},
			"Password": dynamodb.AttributeValue{
				S: aws.String(pwd),
			},
			"Note": dynamodb.AttributeValue{
				S: aws.String(note),
			},
			"Created": dynamodb.AttributeValue{
				N: aws.String(time.Now().Format(time.RFC3339)),
			},
		},
	}
}
