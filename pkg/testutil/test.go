package testutil

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
)

func handleErr(request *http.Request, response http.ResponseWriter, err error) {
	response.WriteHeader(http.StatusInternalServerError)
	response.Write([]byte(err.Error()))
}

func handleOk(response http.ResponseWriter, body []byte) {
	response.WriteHeader(http.StatusOK)
	response.Write(body)
}

type MethodMap map[string]string
type Router map[string]MethodMap

type mocker func(http.ResponseWriter, *http.Request)

func GetMockAPIResponseFromFile(dataDir string, route MethodMap) mocker {
	return func(response http.ResponseWriter, request *http.Request) {
		fileName := route[request.Method]

		obj, err := LoadBytes(dataDir, fileName)

		if err != nil {
			handleErr(request, response, fmt.Errorf("request method: %s", request.Method))
		}

		handleOk(response, obj)
	}
}

func LoadBytes(dir, name string) ([]byte, error) {
	path := filepath.Join(dir, name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error loading file %s in directory %s, %v", name, dir, err)
	}
	return bytes, nil
}
