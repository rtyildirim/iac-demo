#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from '@aws-cdk/core';
import { IacDemoStack } from '../lib/iac-demo-stack';

const app = new cdk.App();

new IacDemoStack(app, 'IacDemoStack', {
    
});
