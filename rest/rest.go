package rest

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/textproto"
	"os"

	"github.com/haplesspanda/haplessbot/constants"
)

var client *http.Client

func init() {
	client = &http.Client{}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func DoJsonRequest(request *http.Request) []byte {
	return DoRequest(request, "application/json")
}

func DoRequest(request *http.Request, contentType string) []byte {
	appendHeaders(request, contentType)
	response, err := client.Do(request)
	check(err)

	dumpedResponse, err := httputil.DumpResponse(response, true)
	check(err)

	log.Printf("HTTP response: %s", dumpedResponse)
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	check(err)

	return body
}

func appendHeaders(request *http.Request, contentType string) {
	authHeader := fmt.Sprintf("Bot %s", constants.TokenId)

	request.Header.Set("Authorization", authHeader)
	request.Header.Set("Content-Type", contentType)
}

type BinaryAttachment struct {
	ContentType string
	Name        string
	Filename    string
}

func MultiPartForm(jsonData []byte, attachment BinaryAttachment) (*bytes.Buffer, string) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	var jsonWriter io.Writer
	jsonHeader := make(textproto.MIMEHeader)
	jsonHeader.Set("Content-Disposition", "form-data; name=\"payload_json\"")
	jsonHeader.Set("Content-Type", "application/json")
	jsonWriter, err := writer.CreatePart(jsonHeader)
	if err != nil {
		panic(err)
	}

	_, err = jsonWriter.Write(jsonData)
	if err != nil {
		panic(err)
	}

	var binaryWriter io.Writer
	binaryHeader := make(textproto.MIMEHeader)
	binaryHeader.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"file0\"; filename=\"%s\"", attachment.Name))
	binaryHeader.Set("Content-Type", attachment.ContentType)
	binaryWriter, err = writer.CreatePart(binaryHeader)
	if err != nil {
		panic(err)
	}

	file, err := os.Open(attachment.Filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	io.Copy(binaryWriter, file)
	if err != nil {
		panic(err)
	}

	err = writer.Close()
	if err != nil {
		panic(err)
	}
	return &b, writer.Boundary()
}
