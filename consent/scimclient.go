package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	scimattrib "github.com/fabbricadigitale/scimd/api/attr"
	scimfilter "github.com/fabbricadigitale/scimd/api/filter"
	"github.com/pkg/errors"
)

// Resource ...
type Resource struct {
	Schemas    []string `json:"schemas"`
	ID         string   `json:"id"`
	ExternalID string   `json:"externalId"`
	Meta       struct {
		ResourceType string    `json:"resourceType"`
		Created      time.Time `json:"created"`
		LastModified time.Time `json:"lastModified"`
		Location     string    `json:"location"`
		Version      string    `json:"version"`
	} `json:"meta"`
	UserName string `json:"userName"`
	Language string `json:"preferredLanguage"`
	Emails   []struct {
		Value string `json:"value"`
	} `json:"emails"`
}

// ListResponse ...
type ListResponse struct {
	Schemas      []string   `json:"schemas"`
	TotalResults int        `json:"totalResults"`
	ItemsPerPage int        `json:"itemsPerPage"`
	StartIndex   int        `json:"startIndex"`
	Resources    []Resource `json:"Resources"`
}

var data ListResponse

func checkErrors(c *http.Response) error {
	code := c.StatusCode
	switch code {
	case 400:
		return errors.New("Bad request")
	case 500:
		return errors.New("Server unavailable")
	default:
		return errors.New("An error occurred")
	}
}

func eq(lx string, rx interface{}) scimfilter.Filter {
	return scimfilter.AttrExpr{
		Path:  scimattrib.Path{Name: lx},
		Op:    scimfilter.OpEqual,
		Value: rx,
	}
}

// GetUserByCredential ...
func GetUserByCredential(username string, password string) (*Resource, error) {
	userURL := url.URL{
		Scheme: "http",
		Host:   "scimd:8787",
		Path:   "/v2/Users",
	}

	filter := scimfilter.And{
		Left:  eq("username", username),
		Right: eq("password", password),
	}

	query := url.Values{}
	query.Add("filter", filter.String())
	userURL.RawQuery = query.Encode()

	resp, err := http.Get(userURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err2 := json.Unmarshal(body, &data)
	if err2 != nil {
		return nil, err2
	}

	// Check the user's credentials
	if resp.StatusCode == http.StatusOK {
		if data.TotalResults == 0 || data.Resources[0].UserName != username {
			return nil, errors.New("Invalid credentials")
		}
	} else {
		err := checkErrors(resp)
		return nil, err
	}

	return &data.Resources[0], nil
}
