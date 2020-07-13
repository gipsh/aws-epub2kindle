# aws-epub2kindle
serverless convert and send epubs to your kindle 

![alt text](https://github.com/gipsh/aws-epub2kindle/blob/master/docs/arq.png?raw=true)


## What is this ? 

This is a sample service to upload, convert and send epubs books directly to your kindle

Its an example of how to convert and old project to the new serverless/sam paradigm. 

## How to deploy 

I used serverless for defining infrastructure and deploy 

First you need to build the layer for calibre

```bash
cd resources/calibre-layer
./build.sh
sls deploy
```

Then you create the bucket 

```bash
cd resources/upload
sls deploy
```

Now upload compile and upload the lambdas

```bash
cd services
make
sls deploy
```

The final part, you need to upload the frontend, so take the created endpoint for the `purl` lambdas and 
edit `resources/frontend/client/dist/index.html` and replace the `lambdaUrl` 

Then deploy the frontend to a static bucket  with: 

```bash
sls deploy client
```

Now you can use your browser to open the deployed endpoint and use the service 











