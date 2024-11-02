#!/bin/bash

# Configurar AWS CLI para usar localstack
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_DEFAULT_REGION="us-east-1"
export ENDPOINT_URL="http://localhost:4566"

# Criar bucket S3
awslocal s3 mb s3://bucket-teste

# Criar secret no Secrets Manager
awslocal secretsmanager create-secret \
    --name "seu-secret-name" \
    --secret-string '{"DB_HOST":"localhost","DB_USER":"test","DB_PASS":"test123"}'

# Criar parâmetros no SSM
awslocal ssm put-parameter \
    --name "/seu/caminho/ssm/PARAM1" \
    --value "valor1" \
    --type "String"

awslocal ssm put-parameter \
    --name "/seu/caminho/ssm/PARAM2" \
    --value "valor2" \
    --type "String"

# Criar função Lambda
awslocal lambda create-function \
    --function-name env-function \
    --runtime provided.al2 \
    --handler bootstrap \
    --role arn:aws:iam::000000000000:role/lambda-role \
    --zip-file fileb://function.zip

# Criar pipeline
awslocal codepipeline create-pipeline \
    --pipeline-name test-pipeline \
    --pipeline '{"name":"test-pipeline","roleArn":"arn:aws:iam::000000000000:role/pipeline-role","artifactStore":{"type":"S3","location":"bucket-teste"},"stages":[{"name":"Source","actions":[{"name":"Source","actionTypeId":{"category":"Source","owner":"AWS","provider":"CodeCommit","version":"1"},"configuration":{"RepositoryName":"test-repo","BranchName":"main"},"outputArtifacts":[{"name":"SourceOutput"}],"runOrder":1}]},{"name":"Build","actions":[{"name":"BuildAction","actionTypeId":{"category":"Build","owner":"AWS","provider":"CodeBuild","version":"1"},"configuration":{"ProjectName":"test-project"},"inputArtifacts":[{"name":"SourceOutput"}],"outputArtifacts":[{"name":"BuildOutput"}],"runOrder":1}]}]}'