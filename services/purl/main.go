package main

import (
        "os"
        "bytes"
        "context"
        "encoding/json"
        "time"
        "math/rand"
        "encoding/hex"
	"regexp"
	"fmt"
	"log"

        "github.com/aws/aws-sdk-go/aws"
        "github.com/aws/aws-lambda-go/events"
        "github.com/aws/aws-lambda-go/lambda"
        "github.com/aws/aws-sdk-go/aws/session"
        "github.com/aws/aws-sdk-go/service/s3"
)

type Response events.APIGatewayProxyResponse

func removeInvalidChars(filename string) string {

    reg, err := regexp.Compile("[^a-zA-Z0-9\\s-_\\(\\)\\.]+")
    if err != nil {
        log.Fatal(err)
    }
    processedString := reg.ReplaceAllString(filename, "")

    fmt.Printf("A string of %s \nbecomes %s \n", filename, processedString)

    return processedString

}

func TempFileName(prefix, suffix string) string {
    randBytes := make([]byte, 16)
    rand.Seed(time.Now().UnixNano())
    rand.Read(randBytes)
    return prefix+hex.EncodeToString(randBytes)+suffix
}

func JSON(t interface{}) ([]byte, error) {
    buffer := &bytes.Buffer{}
    encoder := json.NewEncoder(buffer)
    encoder.SetEscapeHTML(false)
    err := encoder.Encode(t)
    return buffer.Bytes(), err
}


func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
        //var buf bytes.Buffer
        svc := s3.New(session.New())

        s3key := TempFileName("epubs/tmp-",".epub")

        var bucketName = os.Getenv("UPLOAD_BUCKET")

        req, _ := svc.PutObjectRequest(&s3.PutObjectInput{
          Bucket: aws.String(bucketName),
          Key:    &s3key,
          Metadata: map[string]*string{
	   "owner": aws.String(request.QueryStringParameters["owner"]),
//          "file":  aws.String(removeInvalidChars(request.QueryStringParameters["file"])),
	  },
        })
        urlStr, err := req.Presign(15 * time.Minute)

        body, err := JSON(map[string]interface{}{
                "url": urlStr,
                "file": s3key,
		"owner": request.QueryStringParameters["owner"],
		"original_filename": removeInvalidChars(request.QueryStringParameters["file"]),
        })


        if err != nil {
                return Response{StatusCode: 404}, err
        }

        resp := Response{
                StatusCode:      200,
                IsBase64Encoded: false,
                Body:            string(body),
                Headers: map[string]string{
                        "Access-Control-Allow-Origin": "*",
                        "Access-Control-Allow-Headers": "*",
                        "Content-Type":           "application/json",
                },
        }

        return resp, nil
}

func main() {
        lambda.Start(Handler)
}
