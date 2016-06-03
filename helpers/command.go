package helpers

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type PjRequest struct {
	Address string `json:address`
	Port    string `json:port`
	Class   string `json:class`
	Pwd     string `json:pwd`
	Command string `json:command`
	Param   string `json:param`
}

type PjResponse struct {
	Class   string `json:"class"`
	Command string `json:"command"`
	Code    string `json:"code"`
}

func Test() (string, error) {
	return "check yo head!", nil
}

//wrapper function for all handling activity
//success: returns a populated PjResponse struct, nil error
//failure: returns empty PjResponse struct, error
func HandleRequest(address, port, class, pwd, command, param string) (PjResponse, error) {
	//example values:
	//Address = "10.1.1.3"
	//Port = "4352" - default pjlink port
	//Pwd = "magic123"
	//Class = "1"
	//Command = "POWR"
	//Param = "?"

	request := PjRequest{address, port, class, pwd, command, param}

	error1 := validateRequest(request)

	if error1 != nil { //bad cmd, don't send
		return PjResponse{}, error1
	} else { //send request and parse response into struct
		response, error1 := sendRequest(request)
		if error1 != nil {
			return PjResponse{}, error1
		} else {
			fmt.Println("response received (unparsed): " + response)
			return parseResponse(response), nil
		}
	}
}

//this function validates cmd length, before we send request.
//as of now this function only tests for 4 chars, which is pjlink standard cmd length
func validateRequest(request PjRequest) error {
	if len(request.Command) != 4 {
		return errors.New("your command doesn't have char length of 4")
	} else {
		return nil
	}
}

//handle and parse response
//returns a populated PjResponse struct
func parseResponse(response string) PjResponse {
	//if password is wrong, response will be 'PJLINK ERRA'
	if strings.Contains(response, "ERRA") {
		//authentication failed and returned 'ERRA'
		return PjResponse{"0", "ERRA", "0"}
	} else { //authentication succeeded
		//example response: "%1POWR=0"
		//returned params are class, command, and response code, respectively
		return PjResponse{response[1:2], response[2:6], response[7:len(response)]}
	}
}

//send pjlink request to device
//success: returns response string, nil error
//failure: returns empty string, error
func sendRequest(request PjRequest) (string, error) {
	//pjlink always uses tcp
	protocol := "tcp"

	//establish TCP connection with PJLink device
	pjConn, error1 := connectToPjlink(protocol, request.Address, request.Port)
	//fmt.Println("pjConn: " + pjConn.)

	if error1 != nil {
		//attempt to create a TCP socket with specified device failed
		return "", error1
	}

	//setup scanner
	scanner := bufio.NewScanner(pjConn)
	scanner.Split(bufio.ScanWords)

	sResponse := make([]string, 3)

	//grab initial response
	for i := 0; i < 3; i++ {
		scanner.Scan()
		sResponse[i] = scanner.Text()
	}

	//verify PJLink and correct class
	if !verifyPjlink(sResponse) {
		error := errors.New("Not a PJLINK class 1 connection!")
		return "", error
	}

	strCmd := generateCmd(sResponse[2], request.Pwd, request.Class,
		request.Command, request.Param)

	//test
	fmt.Println("sending: " + strCmd + "\n")

	//send command
	pjConn.Write([]byte(strCmd + "\r"))

	scanner.Scan()

	//if authentication failed, we received 'PJLINK ERRA', so return 'ERRA'
	if scanner.Text() == "PJLINK" {
		scanner.Scan()
	}
	pjConn.Close()

	return scanner.Text(), nil
}

//returns pjlink command string
func generateCmd(seed, pjlinkPwd, pjlinkClass, pjlinkCmd, pjlinkParam string) string {
	return createEncryptedMsg(seed, pjlinkPwd) + "%" + pjlinkClass + pjlinkCmd +
		" " + pjlinkParam
}

//generates a hash given seed and password
//returns string hash
func createEncryptedMsg(seed, pjlinkPwd string) string {

	//generate MD5
	data := []byte(seed + pjlinkPwd)
	hash := md5.Sum(data)

	//cast to string
	strHash := hex.EncodeToString(hash[:])

	return strHash
}

//verify we receive a pjlink class 1 response
//success: returns true
//failure: returns false
func verifyPjlink(sResponse []string) bool {

	if sResponse[0] != "PJLINK" {
		return false
	}

	if sResponse[1] != "1" {
		return false
	}

	return true
}

//attempts to establish a TCP socket with the specified IP:port
//success: returns populated pjlinkConn struct and nil error
//failure: returns empty pjlinkConn and error
func connectToPjlink(protocolType, ip, port string) (net.Conn, error) {
	timeout := 5 //represents seconds
	pjlinkConn, err := net.DialTimeout(protocolType, ip+":"+port,
		time.Duration(timeout)*time.Second)
	if err != nil {
		return pjlinkConn, errors.New("failed to establish a connection with " +
			"pjlink device. error msg: " + err.Error())
	}
	return pjlinkConn, err
}