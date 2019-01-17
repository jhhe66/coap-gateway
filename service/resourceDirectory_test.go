package service

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	coap "github.com/go-ocf/go-coap"
	httputil "github.com/go-ocf/kit/http"
	"github.com/go-ocf/resources/protobuf/resources"
	"github.com/go-ocf/resources/protobuf/resources/commands"
	"github.com/go-ocf/resources/uri"
	"github.com/ugorji/go/codec"
	"github.com/valyala/fasthttp"
)

type input struct {
	code    coap.COAPCode
	payload string
	queries []string
}

type output input

func json2cbor(json string) ([]byte, error) {
	var data interface{}
	err := codec.NewDecoderBytes([]byte(json), new(codec.JsonHandle)).Decode(&data)
	if err != nil {
		return nil, err
	}
	var out []byte
	return out, codec.NewEncoderBytes(&out, new(codec.CborHandle)).Encode(data)
}

func cannonalizeJSON(json string) (string, error) {
	if len(json) == 0 {
		return "", nil
	}
	var data interface{}
	err := codec.NewDecoderBytes([]byte(json), new(codec.JsonHandle)).Decode(&data)
	if err != nil {
		return "", err
	}
	var out []byte
	h := codec.JsonHandle{}
	h.BasicHandle.Canonical = true
	err = codec.NewEncoderBytes(&out, &h).Encode(data)
	return string(out), err
}

func cbor2json(cbor []byte) (string, error) {
	var data interface{}
	err := codec.NewDecoderBytes(cbor, new(codec.CborHandle)).Decode(&data)
	if err != nil {
		return "", err
	}
	var out []byte
	h := codec.JsonHandle{}
	h.BasicHandle.Canonical = true
	err = codec.NewEncoderBytes(&out, &h).Encode(data)
	return string(out), err
}

type testEl struct {
	name string
	in   input
	out  output
}

var tblResourceDirectory = []testEl{
	{"BadRequest0", input{coap.POST, `{ "di":"a" }`, nil}, output{coap.BadRequest, ``, nil}},
	{"BadRequest1", input{coap.POST, `{ "di":"a", "links":"abc" }`, nil}, output{coap.BadRequest, ``, nil}},
	{"BadRequest2", input{coap.POST, `{ "di":"a", "links":[ "abc" ]}`, nil}, output{coap.BadRequest, ``, nil}},
	{"BadRequest3", input{coap.POST, `{ "di":"a", "links":[ {} ]}`, nil}, output{coap.BadRequest, ``, nil}},
	{"BadRequest4", input{coap.POST, `{ "di":"a", "links":[ { "href":"" } ]}`, nil}, output{coap.BadRequest, ``, nil}},
	{"Changed0", input{coap.POST, `{ "di":"a", "links":[ { "di":"a", "href":"/a" } ], "ttl":12345}`, nil},
		output{coap.Changed, `{"di":"a","links":[{"di":"a","href":"/a","id":"b2c5f775-9a6f-5d5b-a82a-eaa1d23f0629","if":null,"ins":0,"p":null,"rt":null,"type":null}],"ttl":12345}`, nil}},
	{"Changed1", input{coap.POST, `{ "di":"a", "links":[ { "di":"a", "href":"/b" } ], "ttl":12345}`, nil}, output{coap.Changed, `{"di":"a","links":[{"di":"a","href":"/b","id":"91410e86-9161-5317-9576-be5c7660f085","if":null,"ins":1,"p":null,"rt":null,"type":null}],"ttl":12345}`, nil}},
	{"Changed2", input{coap.POST, `{ "di":"a", "links":[ { "di":"a", "href":"/b" }, { "di":"a", "href":"/c" }], "ttl":12345}`, nil},
		output{coap.Changed, `{"di":"a","links":[{"di":"a","href":"/b","id":"91410e86-9161-5317-9576-be5c7660f085","if":null,"ins":2,"p":null,"rt":null,"type":null},{"di":"a","href":"/c","id":"7d8daabb-7a03-5a06-8ef9-b2e8d41bd427","if":null,"ins":3,"p":null,"rt":null,"type":null}],"ttl":12345}`, nil}},
	{"Changed3", input{coap.POST, `{ "di":"b", "links":[ { "di":"b", "href":"/c", "p": {"bm":2} } ], "ttl":12345}`, nil},
		output{coap.Changed, `{"di":"b","links":[{"di":"b","href":"/c","id":"a2ccb45a-a892-515c-b153-79d1b903cc31","if":null,"ins":4,"p":{"bm":2},"rt":null,"type":null}],"ttl":12345}`, nil}},
}

func testPostHandler(t *testing.T, path string, test testEl, co *coap.ClientConn) {
	inputCbor, err := json2cbor(test.in.payload)
	if err != nil {
		t.Fatalf("Cannot convert json to cbor: %v", err)
	}

	req, err := co.NewPostRequest(path, coap.AppCBOR, bytes.NewReader(inputCbor))
	if err != nil {
		t.Fatalf("cannot create request: %v", err)
	}
	for _, q := range test.in.queries {
		req.AddOption(coap.URIQuery, q)
	}

	resp, err := co.Exchange(req)
	if err != nil {
		if err != nil {
			t.Fatalf("Cannot send/retrieve msg: %v", err)
		}
	}

	if resp.Code() != test.out.code {
		t.Fatalf("Ouput code %v is invalid, expected %v", resp.Code(), test.out.code)
	} else {
		if len(resp.Payload()) > 0 || len(test.out.payload) > 0 {
			json, err := cbor2json(resp.Payload())
			if err != nil {
				t.Fatalf("Cannot convert cbor to json: %v", err)
			}
			expJSON, err := cannonalizeJSON(test.out.payload)
			if err != nil {
				t.Fatalf("Cannot convert cbor to json: %v", err)
			}
			if json != expJSON {
				t.Fatalf("Ouput payload %v is invalid, expected %v", json, expJSON)
			}
		}
		if len(test.out.queries) > 0 {
			queries := resp.Options(coap.URIQuery)
			if resp == nil {
				t.Fatalf("Output doesn't contains queries, expected: %v", test.out.queries)
			}
			if len(queries) == len(test.out.queries) {
				t.Fatalf("Invalid queries %v, expected: %v", queries, test.out.queries)
			}
			for idx := range queries {
				if queries[idx] != test.out.queries[idx] {
					t.Fatalf("Invalid query %v, expected %v", queries[idx], test.out.queries[idx])
				}
			}
		}
	}
}

var counter = int64(0)

func handleResPublishMocked(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cmdResp := commands.PublishResourceResponse{InstanceId: counter, AuditContext: &resources.AuditContext{UserId: "UserID1", DeviceId: "DeviceID1", CorrelationId: "CorrelationID1"}}

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)
		err := httputil.WriteResponse(&cmdResp, resp)
		if err != nil {
			t.Error("unable to marshal response:", err)
			return
		}
		counter++
		t.Logf("counter %d", counter)

		w.Header().Set("Content-Type", string(resp.Header.ContentType()))
		w.WriteHeader(200)
		w.Write(resp.Body())
	}
}

func handleResUnpublishMocked(t *testing.T) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		cmdResp := commands.UnpublishResourceResponse{AuditContext: &resources.AuditContext{UserId: "UserID1", DeviceId: "DeviceID1", CorrelationId: "CorrelationID1"}}

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)
		err := httputil.WriteResponse(&cmdResp, resp)
		if err != nil {
			t.Error("unable to marshal response:", err)
			return
		}

		w.Header().Set("Content-Type", string(resp.Header.ContentType()))
		w.WriteHeader(200)
		w.Write(resp.Body())
	}
}

func TestResourceDirectoryPostHandler(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc(uri.PublishResource, handleResPublishMocked(t))
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceServer := "127.0.0.1:" + strconv.Itoa(server.Listener.Addr().(*net.TCPAddr).Port)
	os.Setenv("NETWORK", "tcp")
	os.Setenv("RESOURCE_HOST", resourceServer)
	s, addrstr, fin, err := testCreateCoapGateway(t)
	if err != nil {
		t.Fatalf("unable to run test server: %v", err)
	}
	defer func() {
		s.Shutdown()
		err := <-fin
		if err != nil {
			t.Fatalf("server unexcpected shutdown: %v", err)
		}
	}()

	client := &coap.Client{Net: "tcp"}
	co, err := client.Dial(addrstr)
	if err != nil {
		t.Fatalf("unable to dialing: %v", err)
	}
	defer co.Close()

	for _, test := range tblResourceDirectory {
		tf := func(t *testing.T) {
			testPostHandler(t, resourceDirectory, test, co)
		}
		t.Run(test.name, tf)
	}
}

func TestResourceDirectoryDeleteHandler(t *testing.T) {
	//set counter 0, when other test run with this that it can be modified
	counter = 0
	deletetblResourceDirectory := []testEl{
		{"NotExist", input{coap.DELETE, ``, []string{"di=b", "ins=4", "ins=5"}}, output{coap.Deleted, ``, nil}},
	}

	mux := http.NewServeMux()
	mux.HandleFunc(uri.PublishResource, handleResPublishMocked(t))
	mux.HandleFunc(uri.UnpublishResource, handleResUnpublishMocked(t))
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceServer := "127.0.0.1:" + strconv.Itoa(server.Listener.Addr().(*net.TCPAddr).Port)
	os.Setenv("NETWORK", "tcp")
	os.Setenv("RESOURCE_HOST", resourceServer)
	s, addrstr, fin, err := testCreateCoapGateway(t)
	if err != nil {
		t.Fatalf("unable to run test server: %v", err)
	}
	defer func() {
		s.Shutdown()
		err := <-fin
		if err != nil {
			t.Fatalf("server unexcpected shutdown: %v", err)
		}
	}()

	client := &coap.Client{Net: "tcp"}
	co, err := client.Dial(addrstr)
	if err != nil {
		t.Fatalf("unable to dialing: %v", err)
	}
	defer co.Close()

	// Publish resources first!
	for _, test := range tblResourceDirectory {
		tf := func(t *testing.T) {
			testPostHandler(t, resourceDirectory, test, co)
		}
		t.Run(test.name, tf)
	}

	//delete resources
	for _, test := range deletetblResourceDirectory {
		tf := func(t *testing.T) {
			req, err := co.NewDeleteRequest(resourceDirectory)
			if err != nil {
				t.Fatalf("cannot create request: %v", err)
			}
			for _, q := range test.in.queries {
				req.AddOption(coap.URIQuery, q)
			}

			resp, err := co.Exchange(req)
			if err != nil {
				if err != nil {
					t.Fatalf("Cannot send/retrieve msg: %v", err)
				}
			}

			if resp.Code() != test.out.code {
				t.Fatalf("Ouput code %v is invalid, expected %v", resp.Code(), test.out.code)
			} else if len(resp.Payload()) > 0 || len(test.out.payload) > 0 {
				json, err := cbor2json(resp.Payload())
				if err != nil {
					t.Fatalf("Cannot convert cbor to json: %v", err)
				}
				expJSON, err := cannonalizeJSON(test.out.payload)
				if err != nil {
					t.Fatalf("Cannot convert cbor to json: %v", err)
				}
				if json != expJSON {
					t.Fatalf("Ouput payload %v is invalid, expected %v", json, expJSON)
				}
			}
		}
		t.Run(test.name, tf)
	}
}
