package main

import (
    "os"
    "fmt"
    "strings"
    "strconv"
    "math/rand"
    "crypto/tls"
    "time"
    "encoding/json"
    "encoding/hex"
    "log"
    "gopkg.in/gomail.v2"
    "github.com/badoux/checkmail"

    "context"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
    "github.com/aws/aws-sdk-go-v2/aws/external"

)

type SMTPConfig struct {
        Port int
        Host string
        User string
        Pass string
        Sender string
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

func Send_Mail(ebook EBook, cfg SMTPConfig) {
        fmt.Println("Sending email")
        if !ValidateAddress(ebook.Owner) {
                return
        }

        d := gomail.NewDialer(cfg.Host, cfg.Port, cfg.User, cfg.Pass)
        d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

        m := gomail.NewMessage()
        m.SetHeader("From", cfg.Sender)
        m.SetHeader("To", ebook.Owner)
        m.SetHeader("Subject", "A book for you")
        m.SetBody("text/html", "Happy birthday. Here goes a small <b>gift</b>!")
        m.Attach(ebook.LocalMobi)

        if err := d.DialAndSend(m); err != nil {
                fmt.Println(err)
        }
}

func ValidateAddress(address string) bool  {

                err := checkmail.ValidateFormat(address)
                if err != nil {
                        fmt.Println("format not valid")
                        return false
                }

                components := strings.Split(address, "@")
                _, domain := components[0], components[1]
                if domain !=  "kindle.com" {
                        fmt.Println("domain is not kindle.com")
                        return false
                }
                return true


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


func handler(ctx context.Context, snsEvent events.SNSEvent) {
    var smtpconfig SMTPConfig

    smtpconfig.Sender = os.Getenv("SMTP_SENDER")
    smtpconfig.Host = os.Getenv("SMTP_HOST")
    smtpconfig.User = os.Getenv("SMTP_USER")
    smtpconfig.Pass = os.Getenv("SMTP_PASS")
    smtpconfig.Port, _ = strconv.Atoi(os.Getenv("SMTP_PORT"))

    fmt.Println("smtpconfig: ", smtpconfig)

    config, _ := external.LoadDefaultAWSConfig()

    for _, record := range snsEvent.Records {
        snsRecord := record.SNS
        fmt.Printf("[%s %s] Message = %s \n", record.EventSource, snsRecord.Timestamp, snsRecord.Message)

	fmt.Println("snsRecord", snsRecord.Message)

	var ebook EBook
        json.Unmarshal([]byte(snsRecord.Message), &ebook)

	fmt.Println(ebook)

	// now download the file
        tmp_mobi := TempFileName("/tmp/",".mobi")
	ebook.LocalMobi = tmp_mobi

	downloadFromS3(ebook.Bucket, ebook.Key, tmp_mobi, config)

	Send_Mail(ebook, smtpconfig)

     }

}

func downloadFromS3(bucket string, item string, output string, config aws.Config) {


        downloader := s3manager.NewDownloader(config)

        file, err := os.Create(output)
        numBytes, err := downloader.Download(file,
            &s3.GetObjectInput{
                Bucket: aws.String(bucket),
                Key:    aws.String(item),
        })

        if err != nil {
            log.Fatalf("Unable to download item %q, %v", item, err)
        }

        fmt.Println("Downloaded", file.Name(), numBytes, "bytes")


}


func main() {
    lambda.Start(handler)
}

