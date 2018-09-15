package store

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
)

// PCloudClient represents the PCloud client instance to interact with PCLoud API.
type PCloudClient struct {
	Client *http.Client
	Token  string
}

// authResponse; internal representation of the auth response. Will be used to Unmarshal the response
type authResponse struct {
	Auth string
}

// uploadFileResponse; internal representation of the upload file response.
type uploadFileResponse struct {
	Fileids []int
}

// generateLinkResponse; internal representation of the public link response generation.
type generateLinkResponse struct {
	Link string
}

// authenticate; sends the HTTP request to authenticate with PCloud using provided credentials.
func authenticate(c *http.Client, username string, password string) (string, error) {
	resp, err := c.Get(buildPCLoudURL("userinfo", url.Values{
		"getauth":  {"1"},
		"logout":   {"1"},
		"username": {username},
		"password": {password},
	}))

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		dump, _ := httputil.DumpResponse(resp, true)

		return "", fmt.Errorf("Server responded with a non 200 (OK) status code. Response dump: \n\n%s", string(dump))
	}

	// Converting the JSON response to bytes.
	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	// We are going to use this struct to Unmarshal the JSON response from PCloud.
	jsonResponse := authResponse{}

	if err := json.Unmarshal(data, &jsonResponse); err != nil {
		return "", err
	}

	if jsonResponse.Auth == "" {
		return "", fmt.Errorf("Failed to parse the authentication. Response was: \n\n%s", string(data))
	}

	return jsonResponse.Auth, err
}

// uploadFile; Uploads files to the PCloud API, returning the fileID.
func uploadFile(pcloud *PCloudClient, filename string, r io.Reader) (int, error) {
	URL := buildPCLoudURL("uploadfile", url.Values{
		"auth": {pcloud.Token},
		// We are always going to upload in the root.
		"path":     {"/"},
		"filename": {filename},
	})

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile(filename, filename)

	if err != nil {
		return 0, err
	}

	if _, err := io.Copy(fw, r); err != nil {
		return 0, err
	}

	if err := w.Close(); err != nil {
		return 0, err
	}

	req, err := http.NewRequest("POST", URL, &b)

	if err != nil {
		return 0, err
	}

	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := pcloud.Client.Do(req)

	if err != nil {
		return 0, err
	}

	if resp.StatusCode != http.StatusOK {
		dump, _ := httputil.DumpResponse(resp, true)

		return 0, fmt.Errorf("Server responded with a non 200 (OK) status code. Response dump: \n\n%s", string(dump))
	}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return 0, err
	}

	jsonResp := uploadFileResponse{}

	if err := json.Unmarshal(data, &jsonResp); err != nil {
		return 0, err
	}

	if len(jsonResp.Fileids) != 1 {
		dump, _ := httputil.DumpResponse(resp, true)
		return 0, fmt.Errorf("Something went wrong. We could not fill get the fileids from the response. Response was: \n\n%s", string(dump))
	}

	return jsonResp.Fileids[0], nil
}

// generatePublicLink; generates a public link to a file it uploaded.
func generatePublicLink(pcloud *PCloudClient, fileID int) (string, error) {
	URL := buildPCLoudURL("getfilepublink", url.Values{
		"auth":   {pcloud.Token},
		"fileid": {strconv.Itoa(fileID)},
	})

	resp, err := pcloud.Client.Get(URL)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		dump, _ := httputil.DumpResponse(resp, true)

		return "", fmt.Errorf("Server responded with a non 200 (OK) status code. Response dump: \n\n%s", string(dump))
	}

	data, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}

	jsonResp := generateLinkResponse{}

	if err := json.Unmarshal(data, &jsonResp); err != nil {
		return "", err
	}

	if jsonResp.Link == "" {
		dump, _ := httputil.DumpResponse(resp, true)
		return "", fmt.Errorf("Something went wrong when generating the public link. Response was: \n\n%s", string(dump))
	}

	return jsonResp.Link, nil
}

// NewPCloudClient returns the PCloudClient to interact with PCloudAPI, or error in case using wrong credentials.
func NewPCloudClient(username string, password string) (*PCloudClient, error) {
	c := &http.Client{}

	// We are hitting the PCloud API when to create the instance.
	token, err := authenticate(c, username, password)

	if err != nil {
		return nil, err
	}

	// PCloudClient the instance needs an HTTP client and the token.
	return &PCloudClient{Client: c, Token: token}, nil
}

// Put receives filename and io.Reader, uploads the file and returns a public link.
func (pcloud *PCloudClient) Put(filename string, r io.Reader) (string, error) {
	fileID, err := uploadFile(pcloud, filename, r)

	if err != nil {
		return "", err
	}

	URL, err := generatePublicLink(pcloud, fileID)

	if err != nil {
		return "", err
	}

	return URL, nil
}
