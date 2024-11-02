package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	configAws "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline"
	"github.com/aws/aws-sdk-go-v2/service/codepipeline/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type UserParameters struct {
	SecretName        string `json:"SecretName"`
	SSMParametersPath string `json:"SSMParametersPath"`
	ProjectName       string `json:"ProjectName"`
}

func putJobFailure(ctx context.Context, jobID, message string) error {
	cfg, err := configAws.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS config: %v", err)
	}

	cpClient := codepipeline.NewFromConfig(cfg)
	_, err = cpClient.PutJobFailureResult(ctx, &codepipeline.PutJobFailureResultInput{
		JobId: aws.String(jobID),
		FailureDetails: &types.FailureDetails{
			Message: aws.String(message),
			Type:    types.FailureTypeJobFailed,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to put job failure result: %v", err)
	}

	return fmt.Errorf(message)
}

func getEnvFunction(ctx context.Context, event events.CodePipelineJobEvent) error {
	jobID := event.CodePipelineJob.ID

	if (events.CodePipelineActionConfiguration{}) == event.CodePipelineJob.Data.ActionConfiguration {
		return putJobFailure(ctx, jobID, "Missing action configuration")
	}

	config := event.CodePipelineJob.Data.ActionConfiguration.Configuration
	if config.UserParameters == "" {
		return putJobFailure(ctx, jobID, "Missing user parameters")
	}

	var userParams UserParameters
	err := json.Unmarshal([]byte(config.UserParameters), &userParams)
	if err != nil {
		return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to parse user parameters: %v", err))
	}

	if userParams.SecretName == "" || userParams.ProjectName == "" {
		return putJobFailure(ctx, jobID, "SecretName and ProjectName are required parameters")
	}

	cfg, err := configAws.LoadDefaultConfig(ctx)
	if err != nil {
		return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to load AWS config: %v", err))
	}

	secretsClient := secretsmanager.NewFromConfig(cfg)
	ssmClient := ssm.NewFromConfig(cfg)
	s3Client := s3.NewFromConfig(cfg)
	cpClient := codepipeline.NewFromConfig(cfg)

	secretResult, err := secretsClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(userParams.SecretName),
	})
	if err != nil {
		return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to get secret: %v", err))
	}

	if secretResult.SecretString == nil {
		return putJobFailure(ctx, jobID, "Secret value is empty")
	}

	var secretData map[string]string
	err = json.Unmarshal([]byte(*secretResult.SecretString), &secretData)
	if err != nil {
		return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to parse secret data: %v", err))
	}

	if userParams.SSMParametersPath != "" {
		ssmResult, err := ssmClient.GetParametersByPath(ctx, &ssm.GetParametersByPathInput{
			Path:           aws.String(userParams.SSMParametersPath),
			Recursive:      aws.Bool(true),
			WithDecryption: aws.Bool(true),
		})
		if err != nil {
			return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to get SSM parameters: %v", err))
		}

		for _, param := range ssmResult.Parameters {
			paramName := strings.TrimPrefix(*param.Name, userParams.SSMParametersPath)
			paramName = strings.TrimPrefix(paramName, "/")
			if paramName != "" {
				secretData[paramName] = *param.Value
			}
		}
	}

	var envContent strings.Builder
	for key, value := range secretData {
		key = strings.TrimSpace(key)
		value = strings.Replace(value, "\n", "", -1)
		value = strings.Replace(value, "\r", "", -1)

		if key != "" {
			_, err := envContent.WriteString(fmt.Sprintf("%s=%s\n", key, value))
			if err != nil {
				return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to write env content: %v", err))
			}
		}
	}

	artifactBucket := os.Getenv("ARTIFACT_BUCKET")
	if artifactBucket == "" {
		return putJobFailure(ctx, jobID, "ARTIFACT_BUCKET environment variable is not set")
	}

	s3Key := filepath.Join(userParams.ProjectName, ".env")

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(artifactBucket),
		Key:         aws.String(s3Key),
		Body:        strings.NewReader(envContent.String()),
		ContentType: aws.String("text/plain"),
	})
	if err != nil {
		return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to upload to S3: %v", err))
	}

	_, err = cpClient.PutJobSuccessResult(ctx, &codepipeline.PutJobSuccessResultInput{
		JobId: aws.String(jobID),
	})
	if err != nil {
		return putJobFailure(ctx, jobID, fmt.Sprintf("Failed to put job success result: %v", err))
	}

	return nil
}

func main() {
	lambda.Start(getEnvFunction)
}
