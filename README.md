# Pipeline Environment Variables Handler

Lambda function written in Go that manages environment variables during AWS CodePipeline execution. It fetches secrets from AWS Secrets Manager and AWS Systems Manager Parameter Store, combining them into a `.env` file that is stored in S3 for use in subsequent pipeline stages.

## Features

- Fetches secrets from AWS Secrets Manager
- Retrieves parameters from AWS Systems Manager Parameter Store (Optional)
- Combines all environment variables into a single `.env` file
- Stores the `.env` file in S3 with project-specific prefixing
- Handles AWS CodePipeline job success/failure notifications
- Sanitizes environment variable values
- Provides detailed error reporting

## Prerequisites

- Go 1.19 or later
- AWS CLI configured with appropriate credentials
- AWS Account with access to:
  - AWS Lambda
  - AWS Secrets Manager
  - AWS Systems Manager Parameter Store
  - Amazon S3
  - AWS CodePipeline

## Required AWS Permissions

The Lambda function requires an IAM role with the following permissions:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "secretsmanager:GetSecretValue",
                "ssm:GetParametersByPath",
                "s3:PutObject",
                "codepipeline:PutJobSuccessResult",
                "codepipeline:PutJobFailureResult"
            ],
            "Resource": [
                "arn:aws:secretsmanager:*:*:secret:*",
                "arn:aws:ssm:*:*:parameter/*",
                "arn:aws:s3:::${ARTIFACT_BUCKET}/*",
                "arn:aws:codepipeline:*:*:*"
            ]
        }
    ]
}
```

## Environment Variables

The Lambda function requires the following environment variable:

- `ARTIFACT_BUCKET`: The name of the S3 bucket where the `.env` file will be stored

## Pipeline Configuration

The function expects the following parameters in the CodePipeline action configuration:

```json
{
    "SecretName": "your-secret-name",
    "SSMParametersPath": "/your/parameter/path",
    "ProjectName": "your-project-name"
}
```

- `SecretName`: (Required) The name of the secret in AWS Secrets Manager
- `SSMParametersPath`: (Optional) The path prefix for parameters in AWS Systems Manager Parameter Store
- `ProjectName`: (Required) The project name used as prefix for the S3 key

## Building and Deployment

1. Build the Lambda function:
```bash
GOOS=linux GOARCH=amd64 go build -o bootstrap main.go
zip function.zip bootstrap
```

2. Create the Lambda function:
```bash
aws lambda create-function \
  --function-name env-handler \
  --runtime provided.al2 \
  --handler bootstrap \
  --zip-file fileb://function.zip \
  --role arn:aws:iam::YOUR_ACCOUNT_ID:role/YOUR_LAMBDA_ROLE
```

3. Update the function code (after making changes):
```bash
aws lambda update-function-code \
  --function-name env-handler \
  --zip-file fileb://function.zip
```

## Local Testing with LocalStack

1. Start LocalStack:
```bash
docker-compose up -d
```

2. Configure AWS CLI for LocalStack:
```bash
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_DEFAULT_REGION="us-east-1"
export ENDPOINT_URL="http://localhost:4566"
```

3. Create required resources:
```bash
# Create S3 bucket
awslocal s3 mb s3://your-artifact-bucket

# Create secret
awslocal secretsmanager create-secret \
    --name "your-secret-name" \
    --secret-string '{"DB_HOST":"localhost","DB_USER":"test"}'

# Create SSM parameter
awslocal ssm put-parameter \
    --name "/your/path/PARAM1" \
    --value "value1" \
    --type "String"
```

4. Run tests:
```bash
go test -v ./...
```

## Error Handling

The function handles various error scenarios:
- Missing or invalid configuration
- AWS service access errors
- Invalid secret formats
- File operation errors
- S3 upload failures

All errors are reported back to CodePipeline with descriptive messages.

## Output

The function creates a `.env` file in the specified S3 bucket with the following format:
```
KEY1=value1
KEY2=value2
```

The file is stored at `s3://${ARTIFACT_BUCKET}/${ProjectName}/.env`

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
