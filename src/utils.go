package src

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func Base64EncodeF(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}

	r := bufio.NewReader(f)
	content, _ := io.ReadAll(r)
	return base64.StdEncoding.EncodeToString(content), nil
}

func Base64EncodeR(r io.Reader) (io.Reader, error) {
	content, _ := io.ReadAll(r)
	return strings.NewReader(base64.StdEncoding.EncodeToString(content)), nil
}

// Download fetches a web resource at "uri" and returns a file handle to the downloaded response.
func Download(uri string) (string, error) {
	if _, err := url.ParseRequestURI(uri); err != nil {
		return "", err
	}

	// Create temp file for download.
	f, err := os.CreateTemp("", "purity-img")
	if err != nil {
		return "", err
	}

	res, err := http.Get(uri)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("request to download image at: %s returned a 404", uri)
	}

	_, err = io.Copy(f, res.Body)
	if err != nil {
		return "", err
	}
	return f.Name(), nil
}

// Hash returns a sha256 hash of the string argument in hex format.
func Hash(str string) string {
	h := sha256.New()
	h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// StringSliceRemove removes element at index i from slice arr.
func StringSliceRemove(arr []string, i int) ([]string, error) {
	if i > len(arr)-1 {
		return nil, fmt.Errorf("index %d must be between 0 and the length of argument arr (%d)", i, len(arr)-1)
	}
	arr[i] = arr[len(arr)-1]
	arr = arr[:len(arr)-1]
	return arr, nil
}
