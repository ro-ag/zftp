package zftp_test

import (
	"fmt"
	"gopkg.in/ro-ag/zftp.v0"
	"reflect"
	"testing"
)

func TestFTPSession_StatusOf(t *testing.T) {
	// Create a new FTP session
	s, err := zftp.Open(hostname)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// Login to the FTP server
	err = s.Login(username, password)
	if err != nil {
		t.Fatal(err)
	}

	l, err := s.XStat("FifoIoTime")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(l)

	tp := reflect.TypeOf(s.StatusOf())
	vl := reflect.ValueOf(s.StatusOf())

	for i := 0; i < tp.NumMethod(); i++ {
		method := tp.Method(i)
		t.Run(method.Name, func(t *testing.T) {
			fmt.Printf("Calling %s\n", method.Name)
			function := vl.MethodByName(method.Name)
			results := function.Call(nil)
			if len(results) == 2 {

				errRes := results[1].Interface()
				if err, ok := errRes.(error); ok && err != nil {
					t.Errorf("Error occurred: %s\n", err.Error())
				}

				switch results[0].Interface().(type) {
				case string:
					strRes := results[0].Interface().(string)
					if strRes == "" {
						t.Errorf("Empty string returned")
					}
					t.Logf("%s: \"%s\"", method.Name, strRes)
				case int:
					t.Logf("%s: %d", method.Name, results[0].Interface().(int))
				case bool:
					t.Logf("%s: %t", method.Name, results[0].Interface().(bool))
				default:
					t.Errorf("Unknown type returned: %s\n", results[0].Type().String())
				}
			}
		})

	}

	return
}
