package main

import (
	"net/http"
	"net/url"

	"github.com/facebookgo/parse"
)

// Client is the http client used by parse-cli
type Client struct {
	client *parse.Client
}

func (c *Client) appendCommonHeaders(header http.Header) http.Header {
	if header == nil {
		header = make(http.Header)
	}
	header.Add("User-Agent", userAgent)
	return header
}

// Get performs a GET method call on the given url and unmarshal response into
// result.
func (c *Client) Get(u *url.URL, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "GET", URL: u}, nil, result)
}

// Post performs a POST method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Post(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "POST", URL: u}, body, result)
}

// Put performs a PUT method call on the given url with the given body and
// unmarshal response into result.
func (c *Client) Put(u *url.URL, body, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "PUT", URL: u}, body, result)
}

// Delete performs a DELETE method call on the given url and unmarshal response
// into result.
func (c *Client) Delete(u *url.URL, result interface{}) (*http.Response, error) {
	return c.Do(&http.Request{Method: "DELETE", URL: u}, nil, result)
}

// RoundTrip is a wrapper for parse.Client.RoundTrip
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header = c.appendCommonHeaders(req.Header)
	return c.client.RoundTrip(req)
}

// Do is a wrapper for parse.Client.Do
func (c *Client) Do(req *http.Request, body, result interface{}) (*http.Response, error) {
	req.Header = c.appendCommonHeaders(req.Header)
	return c.client.Do(req, body, result)
}

// WithCredentials is a wrapper for parse.Client.WithCredentials
func (c *Client) WithCredentials(cr parse.Credentials) *Client {
	c.client = c.client.WithCredentials(cr)
	return c
}
