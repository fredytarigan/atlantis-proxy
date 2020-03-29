package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type StandardResponse struct {
	Message    string
	StatusCode string
}

type CommitHashID struct {
	PullRequest PullRequest `json:"pullrequest"`
}

type PullRequest struct {
	Source Source `json:"source"`
}

type Source struct {
	Commit Commit `json:"commit"`
}

type Commit struct {
	Hash string `json:"hash"`
}

const bitbucketEventTypeHeader = "X-Event-Key"
const bitbucketCloudRequestIDHeader = "X-Request-UUID"

// Get the url for a given proxy condition
func getProxyUrl(proxyConditionRaw string) string {
	proxyCondition := proxyConditionRaw

	a_dev_url := os.Getenv("ATLANTIS_DEV_URL")
	a_prd_url := os.Getenv("ATLANTIS_PRD_URL")
	default_atlantis_url := os.Getenv("ATLANTIS_DEFAULT_URL")

	if proxyCondition == "dev" {
		return a_dev_url
	}

	if proxyCondition == "prd" {
		return a_prd_url
	}

	return default_atlantis_url
}

func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// update the headers
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// serve proxy
	proxy.ServeHTTP(res, req)

}

// Get a json decoder for a given requests body
func requestBodyDecoder(request *http.Request) *json.Decoder {
	// Read body to buffer
	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		panic(err)
	}

	// Because go lang is a pain in the ass if you read the body then any susequent calls
	// are unable to read the body again....
	request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	return json.NewDecoder(ioutil.NopCloser(bytes.NewBuffer(body)))
}

// Given a request send it to the appropriate url
func handleRequestAndRedirect(res http.ResponseWriter, req *http.Request) {
	var c CommitHashID
	commitHash := c.PullRequest.Source.Commit.Hash

	// Checkout to commit hash
	environment, err := gitClone(commitHash)

	if err != nil {
		log.Printf("%s", err)
	}

	url := getProxyUrl(environment)

	serveReverseProxy(url, res, req)
}
