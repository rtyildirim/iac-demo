#!/bin/bash

cd api-lambda 
echo "Building the api lambda binary"
GOOS=linux GOARCH=amd64 go build -o main main.go
echo "Compressing the handler into a ZIP file"
zip lambda-api-function.zip main
echo "Cleaning up"
rm main
cd ..

echo "CDK Deploy..."
cdk deploy