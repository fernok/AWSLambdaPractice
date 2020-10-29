package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/disintegration/imaging"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Response defines the message and whether the function was successful
type Response struct {
	Message string `json:"message"`
	Ok      bool   `json:"ok"`
}

// Handler is the main handler function for AWS lambda
func Handler(event events.S3Event) (Response, error) {
	// srcBucket is the name of the bucket in which a event occurred
	// the handler is triggered by a PNG object creation event in a S3 bucket
	srcBucket := event.Records[0].S3.Bucket.Name
	// itemName is the name of the item created by the event
	itemName := event.Records[0].S3.Object.Key

	// putBucketName is the target S3 bucket to which the function stores the result
	// putBucketName is given as an environment variable
	putBucketName := os.Getenv("PUT_BUCKET_NAME")
	// modificationType is the image modification type, according to which the function modifies the image
	/*
		** types of modification **
		1. grayscale	: changes the image to grayscale
		2. invert		: negates the colors of the image
		3. horizontal	: flips the image horizontally
		4. vertical		: flips the image vertically
	*/
	// this function implements only four of the functions provided by the imaging package
	// for more information, visit https://godoc.org/github.com/disintegration/imaging
	// modificationType is given as an environment variable
	modificationType := os.Getenv("MODIFICATION_TYPE")

	// create a new session for S3
	sess := session.Must(session.NewSession())

	// create a buffer for storing objects fetched from S3
	buff := &aws.WriteAtBuffer{}
	// create a downloader object for managing downloads from S3
	downloader := s3manager.NewDownloader(sess)
	// download the image "itemName" from bucket "srcBucket"
	// is stored into buffer buff
	_, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(srcBucket),
		Key:    aws.String(itemName),
	})
	if err != nil {
		return Response{
			Message: fmt.Sprint("Failed! An Error Occurred."),
			Ok:      false,
		}, err
	}

	// read the bytes of the buffer buff and stores it to data
	data := bytes.NewReader(buff.Bytes())
	// decode the data and transfers it into a image
	img, _, _ := image.Decode(data)

	// according to modificationType, modify the img
	switch modificationType {
	case "grayscale":
		img = imaging.Grayscale(img)
	case "invert":
		img = imaging.Invert(img)
	case "horizontal":
		img = imaging.FlipH(img)
	case "vertical":
		img = imaging.FlipV(img)
	default:
		img = img
	}

	// create a buffer for storing the image
	newBuff := new(bytes.Buffer)
	// endcode the image file into bytes
	err = png.Encode(newBuff, img)
	if err != nil {
		return Response{
			Message: fmt.Sprint("Failed! An Error Occurred."),
			Ok:      false,
		}, err
	}
	// read the bytes of the buffer newBuff and store it into sendData
	// sendData is the data that will be stored into S3 Bucket putBucketName
	sendData := bytes.NewReader(newBuff.Bytes())

	// create a uploader object for managing uploads to S3
	uploader := s3manager.NewUploader(sess)
	// upload the data "sendData" into bucket "putBucketName"
	// the item is stored as "modificationtype-itemName"
	// if modification is "grayscale" and image name is "image.png",
	// the result is "grayscale-image.png"
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(putBucketName),
		Key:    aws.String(modificationType + "-" + itemName),
		Body:   sendData,
	})
	if err != nil {
		return Response{
			Message: fmt.Sprint("Failed! An Error Occurred."),
			Ok:      false,
		}, err
	}

	return Response{
		Message: fmt.Sprintf("Successful! Check %s S3 Bucket.", putBucketName),
		Ok:      true,
	}, nil
}

func main() {
	lambda.Start(Handler)
}
