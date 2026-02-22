package utils

import "net/http"

func DefaultTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	return tr
}

func DefaultInsecureTransport() *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig.InsecureSkipVerify = true
	return tr
}
