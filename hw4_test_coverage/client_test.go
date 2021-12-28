package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const CorrectToken = "correctToken"

type Rows struct {
	List []Row `xml:"row"`
}

type Row struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

func getParamFromRequest(r *http.Request, key string) (string, error) {
	params, ok := r.URL.Query()[key]
	if !ok || len(params[0]) < 1 {
		return "", fmt.Errorf("Url Param %s is missing", key)
	}

	return params[0], nil
}

// func with responses for 100% coverage of FindUsers method
func SearchServer(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("AccessToken")
	if token != CorrectToken {
		w.WriteHeader(http.StatusUnauthorized)
	}

	// orderBy, _ := getParamFromRequest(r, "order_by")
	orderField, _ := getParamFromRequest(r, "order_field")
	query, _ := getParamFromRequest(r, "query")
	limit, _ := getParamFromRequest(r, "limit")
	// offset, _ := getParamFromRequest(r, "offset")

	// TestErrorBadOrderField
	if orderField != "Id" && orderField != "Name" && orderField != "Age" && orderField != "" {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "ErrorBadOrderField"}`)
		return
	}

	switch query {

	case "badRequestBadJSON": // TestBadRequestAndBadJSON
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "bad query"`)
		return

	case "fatalError": // TestServerFatalError
		w.WriteHeader(http.StatusInternalServerError)
		return

	case "unknownBadRequest": // TestUnknownBadRequestError
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error": "Unknown error"}`)
		return

	case "unknownError": // TestUnknownError
		w.WriteHeader(http.StatusMovedPermanently)
		return

	case "timeoutError": // TestTimeOutError
		time.Sleep(time.Millisecond * 1100)
		return

	case "unpackError": // TestUnpackError
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{}`)
		return
	}

	rows := new(Rows)

	file, err := os.Open("dataset.xml")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	xmlData, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	err = xml.Unmarshal(xmlData, &rows)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	users := make([]User, len(rows.List))
	for i, row := range rows.List {
		users[i] = User{
			Id:     row.Id,
			Name:   row.FirstName + " " + row.LastName,
			Gender: row.Gender,
			About:  row.About,
			Age:    row.Age,
		}
	}

	w.WriteHeader(http.StatusOK)
	intLimit, _ := strconv.Atoi(limit)
	var usersJSON []byte
	if query == "onePageResult" {
		usersJSON, _ = json.Marshal(users[0:2])
	} else {
		usersJSON, _ = json.Marshal(users[0:intLimit])
	}

	w.Write(usersJSON)

}

func TestNegativeLimit(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: "token",
		URL:         ts.URL,
	}

	request := SearchRequest{
		Limit: -1,
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for negative limit")
	}
	ts.Close()
}

func TestNegativeOffset(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: "token",
		URL:         ts.URL,
	}

	request := SearchRequest{
		Offset: -2,
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for negative offset")
	}
	ts.Close()
}

func TestIncorrectToken(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: "badToken",
		URL:         ts.URL,
	}

	result, err := client.FindUsers(SearchRequest{})

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for incorrect token")
	}
	if err.Error() != "Bad AccessToken" {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestErrorBadOrderField(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		OrderField: "unknownField",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for bad request")
	}
	if err.Error() != "OrderFeld unknownField invalid" {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestBadRequestAndBadJSON(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Query: "badRequestBadJSON",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for bad request and bad response json")
	}
	if !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestServerFatalError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Query: "fatalError",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for server fatal error")
	}
	if err.Error() != "SearchServer fatal error" {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestUnknownBadRequestError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Query: "unknownBadRequest",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for bad request")
	}
	if !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestUnknownError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Query: "unknownError",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for unknownError case")
	}
	if !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestTimeoutError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Query: "timeoutError",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for timeoutError")
	}
	if !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestUnpackError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Query: "unpackError",
	}

	result, err := client.FindUsers(request)

	if result != nil || err == nil {
		t.Errorf("Error and nil result expected for unpack error")
	}
	if !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("Unexpected error: %#v", err)
	}
	ts.Close()
}

func TestLimitMore25(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Limit: 50,
	}

	result, err := client.FindUsers(request)

	if result == nil || err != nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if len(result.Users) != 25 || !result.NextPage {
		t.Errorf("Incorrect result")
	}
	ts.Close()
}

func TestOnePage(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	client := &SearchClient{
		AccessToken: CorrectToken,
		URL:         ts.URL,
	}

	request := SearchRequest{
		Limit: 10,
		Query: "onePageResult",
	}

	result, err := client.FindUsers(request)

	if result == nil || err != nil {
		t.Errorf("Unexpected error: %#v", err)
	}

	if result.NextPage {
		t.Errorf("Result list is too big")
	}
	ts.Close()
}
