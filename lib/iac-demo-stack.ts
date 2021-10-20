import * as cdk from '@aws-cdk/core';
import * as lambda from '@aws-cdk/aws-lambda';
import { LambdaIntegration, MethodLoggingLevel, RestApi } from "@aws-cdk/aws-apigateway"
import { PolicyStatement } from '@aws-cdk/aws-iam';
import * as dynamodb from '@aws-cdk/aws-dynamodb';


export class IacDemoStack extends cdk.Stack {
  constructor(scope: cdk.Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);


    // Create a dynamodb table for blogs
    const blogTable = new dynamodb.Table(this, "BlogTable", {
      tableName: "iacDemoBlogTable",
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      partitionKey: { name: 'blogId', type: dynamodb.AttributeType.STRING },
      pointInTimeRecovery: false,
    });


    // Create lambda function to handle API requests.
    // Binary app code is in zip file
    const lambdaFunction = new lambda.Function(this, "IacDemoFunction", {
      runtime: lambda.Runtime.GO_1_X,
      handler: "main",
      code: lambda.Code.fromAsset("./api-lambda/lambda-api-function.zip"),
      memorySize: 128,
      timeout: cdk.Duration.seconds(10),
      environment: {
        'BLOG_TABLE_NAME': blogTable.tableName,
      }
    });

    // Grant the lambda role read/write permissions to our table
    blogTable.grantReadWriteData(lambdaFunction);

    // Create new rest api on Api Gateway
    const restApi = new RestApi(this, "IacDemoRestApi", {
      description: "Rest API Demo Using CDK",
      defaultCorsPreflightOptions: {
        allowHeaders: ["*"],
        allowMethods: ['OPTIONS', 'GET', 'POST', 'PUT', 'PATCH', 'DELETE'],
        allowCredentials: true,
        allowOrigins: ["*"],
      },
      deployOptions: {
        stageName: "beta",
        metricsEnabled: true,
        loggingLevel: MethodLoggingLevel.INFO,
        dataTraceEnabled: true,
      },
    })


    // Create API endpoints
    const blogs = restApi.root.addResource('blogs', {});
    const getBlogsMethod = blogs.addMethod("GET", new LambdaIntegration(lambdaFunction, {}), {
      apiKeyRequired: false,
    });
    const postBlogMethod = blogs.addMethod("POST", new LambdaIntegration(lambdaFunction, {}), {
      apiKeyRequired: false,
    });
    const blog = blogs.addResource('{blogId}', {});
    const getBlog = blog.addMethod("GET", new LambdaIntegration(lambdaFunction, {}), {
      apiKeyRequired: false,
    });
  
    // Allow lambda function to create log groups and write logs on CloudWatch
    const logPermission = new PolicyStatement();
    logPermission.addResources('arn:aws:logs:*:*:*');
    logPermission.addActions('logs:CreateLogGroup');
    logPermission.addActions('logs:CreateLogStream');
    logPermission.addActions('logs:PutLogEvents');
    lambdaFunction.addToRolePolicy(logPermission);

    // Output our API url
    new cdk.CfnOutput(this, 'apiUrl', { value: restApi.url });

  }
}
