package parsecmd

import (
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ParsePlatform/parse-cli/parsecli"
	"github.com/facebookgo/ensure"
	"github.com/facebookgo/parse"
)

func TestLogInvalidLevel(t *testing.T) {
	t.Parallel()
	l := logsCmd{}
	h := parsecli.NewHarness(t)
	defer h.Stop()
	err := l.run(h.Env, nil)
	ensure.Err(t, err, regexp.MustCompile(`invalid level: ""`))
}

func TestLogWithoutFollow(t *testing.T) {
	t.Parallel()
	l := logsCmd{level: "INFO"}
	h := parsecli.NewHarness(t)
	defer h.Stop()
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		ensure.DeepEqual(t, r.URL.Path, "/1/scriptlog")
		rows := []logResponse{{Message: "foo bar"}, {Message: "baz"}}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
		}, nil
	})
	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	err := l.run(h.Env, &parsecli.Context{})
	ensure.Nil(t, err)
	ensure.DeepEqual(t, h.Out.String(), "baz\nfoo bar\n")
}

func TestLogWithFollow(t *testing.T) {
	t.Parallel()
	l := logsCmd{level: "INFO", follow: true}
	h := parsecli.NewHarness(t)
	defer h.Stop()
	var round int64
	round1Time := parseTime{ISO: "iso1", Type: "type1"}
	round1TimeStr := jsonStr(t, round1Time)
	round3Time := parseTime{ISO: "iso2", Type: "type2"}
	round3TimeStr := jsonStr(t, round3Time)
	ht := parsecli.TransportFunc(func(r *http.Request) (*http.Response, error) {
		switch atomic.AddInt64(&round, 1) {
		case 1:
			// expect no timestamp, return some data
			ensure.DeepEqual(t, r.FormValue("startTime"), "")
			rows := []logResponse{{Message: "foo bar", Timestamp: round1Time}}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
			}, nil
		case 2:
			// expect the timestamp from case 1, return no new data
			ensure.DeepEqual(t, r.FormValue("startTime"), round1TimeStr)
			rows := []logResponse{}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
			}, nil
		case 3:
			// expect the timestamp from case 1, return some new data
			ensure.DeepEqual(t, r.FormValue("startTime"), round1TimeStr)
			rows := []logResponse{{Message: "baz", Timestamp: round3Time}}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       ioutil.NopCloser(strings.NewReader(jsonStr(t, rows))),
			}, nil
		case 4:
			// expect the timestamp from case 3, return error
			ensure.DeepEqual(t, r.FormValue("startTime"), round3TimeStr)
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Status:     http.StatusText(http.StatusInternalServerError),
				Body:       ioutil.NopCloser(strings.NewReader("a")),
			}, nil
		}
		panic("unexpected request")
	})

	stop := make(chan struct{})
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				h.Clock.Add(logFollowSleepDuration)
			}
		}
	}()

	h.Env.ParseAPIClient = &parsecli.ParseAPIClient{APIClient: &parse.Client{Transport: ht}}
	err := l.run(h.Env, &parsecli.Context{})
	close(stop)

	ensure.Err(t, err, regexp.MustCompile(`parse: error with status=500 and body="a"`))
	ensure.DeepEqual(t, h.Out.String(), "foo bar\nbaz\n")
	ensure.DeepEqual(t, h.Err.String(), "")
	ensure.DeepEqual(t, atomic.LoadInt64(&round), int64(4))
}
