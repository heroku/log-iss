package logiss

import (
	"strconv"
	"time"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/heroku/log-iss/Godeps/_workspace/src/golang.org/x/crypto/bcrypt"
)

// FIXME: Add condition to not put user when the username already exists
func NewUserItem(table, user, pwd, note string) dynamodb.PutItemInput {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	return dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item: map[string]dynamodb.AttributeValue{
			"UserName": dynamodb.AttributeValue{
				S: aws.String(user),
			},
			"Password": dynamodb.AttributeValue{
				S: aws.String(string(hash)),
			},
			"Note": dynamodb.AttributeValue{
				S: aws.String(note),
			},
			"Version": dynamodb.AttributeValue{
				N: aws.String("0"), // start with a version of 0
			},
			"Created": dynamodb.AttributeValue{
				N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
			},
		},
	}
}
