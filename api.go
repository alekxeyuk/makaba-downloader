package main

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
)

type DvachApi struct {
	url    string
	client *http.Client
}

func NewDvachApi(cookies map[string]string) *DvachApi {
	client := &http.Client{}
	client.Jar, _ = cookiejar.New(nil)
	apiUrl, _ := url.Parse("https://2ch.su")
	for k, v := range cookies {
		client.Jar.SetCookies(apiUrl, []*http.Cookie{{Name: k, Value: v}})
	}
	return &DvachApi{
		url:    "https://2ch.su",
		client: client,
	}
}

func (api *DvachApi) catalogGet(board string) ([]byte, error) {
	return api.getJSON(board + "/catalog.json")
}

func (api *DvachApi) threadGet(board, threadNum string) ([]byte, error) {
	return api.getJSON(board + "/res/" + threadNum + ".json")
}

func (api *DvachApi) getJSON(path string) ([]byte, error) {
	req, err := http.NewRequest("GET", api.url+"/"+path, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range map[string]string{
		"content-type":    "application/json",
		"Accept":          "application/json",
		"Accept-encoding": "gzip, deflate, sdch",
	} {
		req.Header.Set(k, v)
	}
	resp, err := api.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer reader.Close()
	default:
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	return body, nil
}
