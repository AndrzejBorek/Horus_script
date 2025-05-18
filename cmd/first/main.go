package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
	"strings"

	"github.com/tidwall/gjson"
)

var (
	userApiUrl        string
	client            *http.Client
	includeFields     string
	createUserUrl     string
	batchNumber         int8 = 20
	batchSize     int8 = 5
	emptyResultsError      = errors.New("Empty results array from api.")
)

type User struct {
	FirstName string
	LastName  string
	Username  string
}

func init() {
	userApiUrl = os.Getenv("USER_API_URL")
	if userApiUrl == "" {
		panic("USER_API_URL is not set.")
	}

	//TODO check if url is valid url format

	client = &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     5,
			MaxIdleConnsPerHost: 5,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		Timeout: 5 * time.Second,
	}

	// Checking if server is reponsive
	headRes, err := client.Head(userApiUrl)
	if err != nil {
		panic(fmt.Sprintf("Error when checking url: %s; error: %s", userApiUrl, err))
	}

	// Since in the assignment there was only mention about name, surname and username I can only send request with that parameters
	// and therefore fasten it. Reference: https://randomuser.me/documentation#format
	includeFields = "?inc=name,login"
	createUserUrl = userApiUrl + includeFields
	headRes, err = client.Head(createUserUrl)

	if err != nil || headRes.StatusCode >= 400 {
		panic(fmt.Sprintf("Error when checking url: %s; StatusCode: %d ;Status: %s", createUserUrl, headRes.StatusCode, headRes.Status))
	}
}

func getUserData() (user User, err error) {
	res, err := client.Get(createUserUrl)

	if err != nil {
		return User{}, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)

	if err != nil {
		return User{}, err
	}

	var strBody string = string(body)
	if !gjson.Get(strBody, "results").Exists() || len(gjson.Get(strBody, "results").Array()) == 0 {
		// It seems like api sometimes returns empty results array
		return User{}, emptyResultsError
	}

	firstName := gjson.Get(strBody, "results.0.name.first").String()
	lastName := gjson.Get(strBody, "results.0.name.last").String()
	username := gjson.Get(strBody, "results.0.login.username").String()

	if firstName == "" || lastName == "" || username == "" {
		panic(fmt.Sprintf("Missing fields in %s", strBody))
	}

	user = User{
		FirstName: firstName,
		LastName:  lastName,
		Username:  username,
	}

	return user, nil
}

func fetchUser() (User, error) {
	fmt.Println(time.Now())
	for {
		user, err := getUserData()
		if err != nil {
			if err == emptyResultsError {
				continue
			}
			return user, err
		}
		return user, nil
	}
}

func GetUsersData() (userChan chan User) {
	userChan = make(chan User, int(batchSize*batchNumber))
	var wg sync.WaitGroup

	runtime.GOMAXPROCS(5)

	for i := int8(0); i < batchNumber; i++ {

		fmt.Println("Batch number ", i+1)

		wg.Add(int(batchSize))
		for j := int8(0); j < batchSize; j++ {
			go func() {
				defer wg.Done()
				user, err := fetchUser()

				if err != nil {
					panic(err)
				}
				userChan <- user
			}()

		}
		wg.Wait()
		fmt.Println("Sleeping 2 seconds")
		time.Sleep(2 * time.Second)
	}

	close(userChan)
	return userChan
}



func GenerateUserAddScript(userChan chan User) {
    var sb strings.Builder

    sb.WriteString("#!/bin/bash -xe\n\n")
    for user := range userChan {
	sb.WriteString(fmt.Sprintf("if [ -d \"/home/%s\" ]; then\n\techo \"User %s not added: home directory exists\";\nelse\n\tuseradd -g 100 -m -c \"%s %s\" -d /home/%s -s /bin/bash %s;\nfi\n", user.Username, user.Username, user.FirstName, user.LastName, user.Username, user.Username))
    }

    scriptContent := sb.String()

    err := os.WriteFile("create_users.sh", []byte(scriptContent), 0700)
    if err != nil {
        panic(fmt.Sprintf("Failed to write script: %v", err))
    }

    fmt.Println("Script saved as create_users.sh")
}

func SaveUsersToFile(userChan chan User) {
	var csvData = make([][]string, int(batchNumber*batchSize))

	for i := range csvData {
		csvData[i] = make([]string, 3)
	}

	index := 0
	for user := range userChan {
		csvData[index][0] = user.FirstName
		csvData[index][1] = user.LastName
		csvData[index][2] = user.Username
		index++
	}

	buf := new(bytes.Buffer)
	wr := csv.NewWriter(buf)
	wr.WriteAll(csvData)
	csvString := buf.String()
	fmt.Println(csvString)
}


func main() {
    userChan := GetUsersData()
    GenerateUserAddScript(userChan)
}

