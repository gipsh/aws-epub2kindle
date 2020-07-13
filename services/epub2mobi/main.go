// main.go
package main

import (
	"fmt"
	"context"
	"log"
	"os"
	"time"
	"os/exec"
	"strings"
        "encoding/hex"
        "encoding/json"
        "math/rand"

	"path/filepath"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
        "github.com/aws/aws-sdk-go-v2/aws"
        "github.com/aws/aws-sdk-go-v2/service/s3"
        "github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
        "github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/aws/external"
)

type Message struct {
    Default string `json:"default"`
}


type EBook struct {
        Owner string `json:owner`
        Bucket string `json:bucket`
        Key string  `json:key`
        LocalMobi string
}



func TempFileName(prefix, suffix string) string {
    randBytes := make([]byte, 16)
    rand.Seed(time.Now().UnixNano())
    rand.Read(randBytes)
    return prefix+hex.EncodeToString(randBytes)+suffix
}


func Get_MOBI_Target(epub string) string {
	name := strings.TrimSuffix(epub, filepath.Ext(epub))
	return fmt.Sprintf("%s.mobi",name)

}

func uploadFile(src string, bucket string, key string, owner string, filename string, config aws.Config) {

    file, err := os.Open(src)
    if err != nil {
       fmt.Println("error", err)
    }
    defer file.Close()

    // Get file size and read the file content into a buffer

    uploader := s3manager.NewUploader(config)

    meta := make(map[string]string)
    meta["owner"] = owner
//    meta["filename"] = filename
//    fmt.Println("Metadata: ", meta)

    numBytes, err := uploader.Upload(&s3manager.UploadInput{
	    Bucket: aws.String(bucket), // Bucket to be used
	    Key:    aws.String(key),      // Name of the file to be saved
	    Body:   file,                      // File
	    Metadata: meta,
	})

        if err != nil {
            log.Fatalf("Unable to upload item %q, %v", key, err)
        }

        fmt.Println("uploaded", file.Name(), numBytes, "bytes")

}


func getHeadObject(bucket string, key string, config aws.Config) *s3.HeadObjectResponse {
    client := s3.New(config)
    input := &s3.HeadObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    }

    request := client.HeadObjectRequest(input)

    headObjectResponse, err := request.Send(context.TODO())

    if err != nil {
        panic(err)
    }

    fmt.Printf("Downloaded HeadObject: %v\n", request)

    return headObjectResponse
}


func sendToMailer(owner string, bucket string, key string, config aws.Config, ctx context.Context) {

    snsClient := sns.New(config)

    ebook := EBook{
        Owner: owner,
	Bucket: bucket,
	Key: key,
    }

   fmt.Println("ebook struct", ebook)

    ebookStr, _ := json.Marshal(ebook)

    message := Message{
        Default: string(ebookStr),
    }
    messageBytes, _ := json.Marshal(message)
    messageStr := string(messageBytes)
    fmt.Println("json", messageStr)

    req := snsClient.PublishRequest(&sns.PublishInput{
        TopicArn: aws.String( os.Getenv("SNS_MAILER")),
        Message: aws.String(messageStr),
        MessageStructure: aws.String("json"),
    })

    res, err := req.Send(ctx)
    if err != nil {log.Fatal(err)
    }

    log.Print(res)

}


func handler(ctx context.Context, s3Event events.S3Event) {
	for _, record := range s3Event.Records {
		s3data := record.S3
		fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", record.EventSource, record.EventTime, s3data.Bucket.Name, s3data.Object.Key)

        config, _ := external.LoadDefaultAWSConfig()

        downloader := s3manager.NewDownloader(config)
	tmp_epub := TempFileName("/tmp/",".epub")
	output_mobi := Get_MOBI_Target(tmp_epub)

        fmt.Println("file is ", tmp_epub)

	// get metadata first
	headObject := getHeadObject(s3data.Bucket.Name, s3data.Object.Key, config)
        fmt.Printf("%+v", headObject)
	metadata := headObject.Metadata
	owner, found := metadata["Owner"]
	filename, found  := metadata["file"]

	if found {
		fmt.Println("found!")
	}

	fmt.Println("owner: ", owner)
//	fmt.Println("filename: ", filename)

        file, err := os.Create(tmp_epub)
        numBytes, err := downloader.Download(file,
            &s3.GetObjectInput{
                Bucket: aws.String(s3data.Bucket.Name),
                Key:    aws.String(s3data.Object.Key),
        })

        if err != nil {
            log.Fatalf("Unable to download item %q, %v", s3data.Object.Key, err)
        }

        fmt.Println("Downloaded", file.Name(), numBytes, "bytes")
	fmt.Println("output mobi is", output_mobi)

	ebook_convert := "/opt/ebook-convert"
	out, err := exec.Command(ebook_convert, tmp_epub, output_mobi).Output()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("The output is %s\n", out)

	uploadFile(output_mobi, s3data.Bucket.Name, "mobis/"+filepath.Base(output_mobi), owner, filename, config)

	fmt.Println("Sending to SNS")
	sendToMailer(owner, s3data.Bucket.Name, "mobis/"+filepath.Base(output_mobi), config, ctx)


  }
}


func main() {
	// Make the handler available for Remote Procedure Call by AWS Lambda
	lambda.Start(handler)
}
