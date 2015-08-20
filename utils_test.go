package main

import (
	"crypto/rand"
	"io/ioutil"
	"net/http"
	"regexp"
	"testing"

	"github.com/facebookgo/ensure"
	"github.com/facebookgo/jsonpipe"
	"github.com/facebookgo/parse"
)

func TestBothEmpty(t *testing.T) {
	t.Parallel()
	ensure.False(t, numericLessThan("", ""))
}

func TestOnlyAEmpty(t *testing.T) {
	t.Parallel()
	ensure.True(t, numericLessThan("", "b"))
}

func TestOnlyBEmpty(t *testing.T) {
	t.Parallel()
	ensure.False(t, numericLessThan("a", ""))
}

func TestBothEqual(t *testing.T) {
	t.Parallel()
	ensure.False(t, numericLessThan("abc", "abc"))
}

func TestAPrefixOfB(t *testing.T) {
	t.Parallel()
	ensure.True(t, numericLessThan("abc", "abcd"))
}

func TestBPrefixOfA(t *testing.T) {
	t.Parallel()
	ensure.False(t, numericLessThan("abcd", "abc"))
}

func TestABSansNumbers(t *testing.T) {
	t.Parallel()

	const max = 10
	dictionary := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var rb [max]byte
	randomString := func(length int) string {
		bytes := rb[:length]
		_, err := rand.Read(bytes)
		ensure.Nil(t, err)
		for k, v := range bytes {
			bytes[k] = dictionary[v%byte(len(dictionary))]
		}
		return string(bytes)
	}

	for i := 1; i < max; i++ {
		for j := 1; j < max; j++ {
			a, b := randomString(i), randomString(j)
			ensure.DeepEqual(t, numericLessThan(a, b), a < b)
		}
	}
}

func TestABWithNumbers(t *testing.T) {
	ensure.True(t, numericLessThan("abc9", "abc12"))
	ensure.False(t, numericLessThan("abc12", "abc9"))
	ensure.DeepEqual(t, numericLessThan("abc001", "abc1"), "abc001" < "abc1")
}

func TestGetHostFromURL(t *testing.T) {
	urlStr := "https://api.parse.com/1/"
	host, err := getHostFromURL(urlStr)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, host, "api.parse.com")

	urlStr = "https://api.example.com:8080/1/"
	host, err = getHostFromURL(urlStr)
	ensure.Nil(t, err)
	ensure.DeepEqual(t, host, "api.example.com")

	urlStr = "api.example.com:8080:90"
	host, err = getHostFromURL(urlStr)
	ensure.Err(t, err, regexp.MustCompile("not a valid url"))
}

func TestIsSupportedWarning(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/supported")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: ioutil.NopCloser(
				jsonpipe.Encode(
					map[string]string{"warning": "please update"},
				),
			),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	message, err := checkIfSupported(h.env, "2.0.2")
	ensure.Nil(t, err)
	ensure.DeepEqual(t, message, "please update")
}

func TestIsSupportedError(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	defer h.Stop()

	ht := transportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/supported")
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body: ioutil.NopCloser(
				jsonpipe.Encode(
					map[string]string{"error": "not supported"},
				),
			),
		}, nil
	})
	h.env.ParseAPIClient = &ParseAPIClient{apiClient: &parse.Client{Transport: ht}}
	_, err := checkIfSupported(h.env, "2.0.2")
	ensure.Err(t, err, regexp.MustCompile("not supported"))
}
