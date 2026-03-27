package helpers

import (
	"bytes"
	"io"
	"net"
	"net/http"
)

func DoJSON(t TestReporter, method string, endpoint string, body []byte, expectedStatus int) []byte {
	t.Helper()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		t.Fatalf("create %s %s request: %v", method, endpoint, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s %s response: %v", method, endpoint, err)
	}
	if resp.StatusCode != expectedStatus {
		t.Fatalf("unexpected status from %s %s: got=%d want=%d body=%s", method, endpoint, resp.StatusCode, expectedStatus, string(respBody))
	}

	return respBody
}

func FreeTCPPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = ln.Close() }()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errUnexpectedAddrType(ln.Addr())
	}

	return addr.Port, nil
}

type unexpectedAddrTypeErr struct {
	addr net.Addr
}

func (e unexpectedAddrTypeErr) Error() string {
	return "unexpected listener address type: " + e.addr.Network()
}

func errUnexpectedAddrType(addr net.Addr) error {
	return unexpectedAddrTypeErr{addr: addr}
}
