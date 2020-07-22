package awsclient

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
)

type AWSSession struct {
	*rds.RDS
}

func CreateSession(region string, profile string) (*AWSSession, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewSharedCredentials("", profile),
	})
	if err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("failed to setup aws session: %s", err.Error()))
	}

	if _, err := sess.Config.Credentials.Get(); err != nil {
		return nil, fmt.Errorf(fmt.Sprintf("failed to retrieve aws credentials: %s", err.Error()))
	}

	rdsSvc := rds.New(sess)

	return &AWSSession{rdsSvc}, nil
}
