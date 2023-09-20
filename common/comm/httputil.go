package comm

import (
	"bytes"
	"fmt"
	"github.com/rongzer/blockchain/common/conf"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rongzer/blockchain/common/log"
	cb "github.com/rongzer/blockchain/protos/common"
)

// Client is the blockchain-ca client object
type HttpClient struct {
	ch         chan int
	httpClient *http.Client
}

func NewHttpClient() *HttpClient {
	hc := &HttpClient{ch: make(chan int, 100)}
	//并发100个http请求
	hc.httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second, //连接超时时间
				KeepAlive: 30 * time.Second, //连接保持超时时间
			}).DialContext,
			MaxIdleConnsPerHost:   100,
			MaxIdleConns:          300,
			ResponseHeaderTimeout: time.Second * 5,
			IdleConnTimeout:       time.Second * 90,
			TLSHandshakeTimeout:   10 * time.Second,
		},
	}

	return hc
}

func (hc *HttpClient) Reqest(httpRequest *cb.RBCHttpRequest) ([]byte, error) {

	httpRequest.Endpoint = strings.TrimSpace(httpRequest.Endpoint)

	curl := httpRequest.Endpoint
	if !strings.HasPrefix(curl, "http") { //有http前缀，不作地址处理
		urlStr := conf.V.Sealer.CaAddress
		if len(urlStr) < 1 {
			return nil, fmt.Errorf("ca url not config")
		}
		if strings.HasSuffix(urlStr, "/") {
			urlStr = urlStr[0 : len(urlStr)-1]
		}

		if strings.HasPrefix(httpRequest.Endpoint, "/") {
			httpRequest.Endpoint = httpRequest.Endpoint[1:]
		}

		curl = fmt.Sprintf("%s/%s", urlStr, httpRequest.Endpoint)
	}

	//并发发送处理
	//hc.ch <- 1
	if httpRequest.Method == "POST" {
		bReturn, err := hc.post(curl, httpRequest.Body, httpRequest.Headers, httpRequest.Params, 3)
		//<-hc.ch
		return bReturn, err
	}
	bReturn, err := hc.get(curl, httpRequest.Params)
	//<-hc.ch
	return bReturn, err
}

func (hc *HttpClient) post(curl string, reqBody []byte, headers map[string]string, params map[string]string, nRetry int) ([]byte, error) {
	vbody := bytes.NewReader(reqBody)

	if params != nil && len(params) > 0 {
		kv := url.Values{}
		for k, v := range params {
			kv.Add(k, v)
		}

		vbody = bytes.NewReader([]byte(kv.Encode())) //把form数据编下码
	}

	req, err := http.NewRequest("POST", curl, vbody)

	if err != nil {
		return nil, fmt.Errorf("Failed posting to %s", curl)
	}

	if params != nil && len(params) > 0 {
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	}

	//设置header
	if headers != nil {
		if len(headers["user"]) > 0 && len(headers["secret"]) > 0 {
			req.SetBasicAuth(headers["user"], headers["secret"])
			delete(headers, "user")
			delete(headers, "secret")
		}

		for k, v := range headers {
			req.Header.Add(k, v)
		}
	}

	//log.Logger.Infof("http req : %s curl : %s", req, curl)
	resp, err := hc.httpClient.Do(req)

	if err != nil {

		if nRetry > 0 {
			log.Logger.Warnf("request sync(%d) %d err:%s", nRetry, len(hc.ch), err)

			nRetry--
			return hc.post(curl, reqBody, headers, params, nRetry)
		}

	}

	if resp == nil || resp.Body == nil {
		log.Logger.Warnf("request url %s return null", curl)

		return nil, fmt.Errorf("Http Request Connect  err")
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Warnf("request read body err:%s", err)

		return nil, err
	}

	return body, nil

}

func (hc *HttpClient) get(curl string, params map[string]string) ([]byte, error) {

	if params != nil && len(params) > 0 {
		if strings.Index(curl, "?") < 0 {
			curl = curl + "?"
		}
		for k, v := range params {
			curl = curl + k + "=" + v + "&"
		}

		if strings.HasSuffix(curl, "&") {
			curl = curl[:len(curl)-1]
		}
	}

	log.Logger.Info("get curl : ", curl)
	resp, err := hc.httpClient.Get(curl)
	if err != nil {
		log.Logger.Warnf("get sync %d err:%s", len(hc.ch), err)

		return nil, err
	}

	if resp == nil || resp.Body == nil {
		log.Logger.Warnf("get url %s return null", curl)

		return nil, fmt.Errorf("Http Request Connect  err")
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Logger.Warnf("get url %s read body err:%s", curl, err)

		return nil, err
	}

	return body, nil
}
