package types

import (
	"io"
)

type Server struct {
	GrpcHost      string
	GrpcPort      string
	Output        string
	Packages      []PackageDef
	PackageSuffix string
	StubHost      string
	StubPort      string
}

type PackageDef struct {
	Alias string
	Path  string
}

type ServiceDetails struct {
	Alias   string
	Package string
	Path    string
	Service string
}

type Proto struct {
	Dependencies    map[string]string
	FilesToGenerate []string
	OutFile         string
	Package         string
	PackagePath     string
	PackageSuffix   string
	Services        []Service
	Writer          io.Writer
}

type Service struct {
	Methods []MethodTemplate
	Name    string
}

type MethodTemplate struct {
	Input       string
	MethodType  string
	Name        string
	Output      string
	ServiceName string
}

const ServerTemplate = `
// Auto generated code . DO NOT EDIT.
package grpcsrv

import (
	"context"
	"log"
	"net"
	"strconv"

	"google.golang.org/grpc"

    "{{.PackageSuffix}}stub"
	{{ range .Packages }}
    {{.Alias}} "{{.Path}}"
    {{end}}
)

type Server struct{
	Host string
	Port int
	StubAddr string
}

func StartGrpcMockServer(ctx context.Context, params *Server) {
	addr := params.Host + ":" + strconv.Itoa(params.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	stub.RegisterStubAddress(params.StubAddr)

	s := grpc.NewServer()
	registerServices(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve grpc server: %v", err)
		}
	}()
	log.Println("GRPC Mock Server :: Started : ", addr)

	defer func() {
		log.Println("GRPC Mock Server :: Shutdown command is issued..")
		s.Stop()
		log.Println("GRPC Mock Server :: Exited Properly..")
	}()

	<-ctx.Done()
	return
}


func registerServices(s *grpc.Server) {
    {{ range .Packages }}
    {{.Alias}}.Register(s)
    {{ end }}
}
`

const StubTemplate = `
// Auto generated code . DO NOT EDIT.
package stub

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

const (
	findEndPoint = "/find"
)

var Addr string

type payload struct {
	Service string      ` + "`" + `json:"service"` + "`" + `
	Method  string      ` + "`" + `json:"method"` + "`" + `
	Data    interface{} ` + "`" + `json:"data"` + "`" + `
}

type response struct {
	Data  interface{} ` + "`" + `json:"data"` + "`" + `
	Error string      ` + "`" + `json:"error"` + "`" + `
}

func RegisterStubAddress (addr string){
	Addr = addr
}

func Find(service, method string, in interface{}) (interface{}, error) {
	pyl := payload{
		Service: service,
		Method:  method,
		Data:    in,
	}
	byt, err := json.Marshal(pyl)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewReader(byt)
	stubEndpoint := Addr + findEndPoint
	resp, err := http.DefaultClient.Post(stubEndpoint, "application/json", reader)
	if err != nil {
		return nil, fmt.Errorf("error request to stub service %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf(string(body))
	}

	respRPC := new(response)
	err = json.NewDecoder(resp.Body).Decode(respRPC)
	if err != nil {
		return nil, fmt.Errorf("decoding json response %v", err)
	}

	if respRPC.Error != "" {
		return nil, fmt.Errorf(respRPC.Error)
	}
	return respRPC.Data, nil
}


func Load(in, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("decoding json response %v", err)
	}
	return jsonpb.Unmarshal(bytes.NewReader(data), out.(proto.Message))
}

func FindAndLoad(service, method string, in, out interface{}) error {
	data, err := Find(service, method, in)
	if err != nil {
		return err
	}
	return Load(data, out)
}
`

const ServiceTemplate = `
// Auto generated code . DO NOT EDIT.
package {{.Package}}

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"{{.PackageSuffix}}stub"
)
{{ range $package, $alias := .Dependencies }}
import {{$alias}} "{{$package}}"
{{end}}

{{ range .Services }}
{{ template "services" . }}
{{ end }}

{{ define "services" }}
type {{.Name}} struct{}

{{ template "methods" .}}
{{ end }}

{{ define "methods" }}
{{ range .Methods}}
	{{ if eq .MethodType "standard"}}
		{{ template "standard_method" .}}
	{{ else if eq .MethodType "server-stream"}}
		{{ template "server_stream_method" .}}
	{{ else if eq .MethodType "client-stream"}}
		{{ template "client_stream_method" .}}
	{{ else if eq .MethodType "bidirectional"}}
		{{ template "bidirectional_method" .}}
	{{ end }}
{{end}}
{{end}}

{{ define "standard_method" }}
func (s *{{.ServiceName}}) {{.Name}}(ctx context.Context, in *{{.Input}}) (*{{.Output}},error){
	out := &{{.Output}}{}
	err := stub.FindAndLoad("{{.ServiceName}}","{{.Name}}", in, out)
	return out, err
}
{{ end }}

{{ define "server_stream_method" }}
func (s *{{.ServiceName}}) {{.Name}}(in *{{.Input}},stream {{.ServiceName}}_{{.Name}}Server) error {
	out := &{{.Output}}{}
	data, err := stub.Find("{{.ServiceName}}", "{{.Name}}", in)
	if err != nil {
		return err
	}
	for _, d := range data.([]interface{}) {
		err := stub.Load(d, out)
		if err != nil {
			return err
		}
		if err := stream.Send(out); err != nil {
			return err
		}
	}
	return nil
}
{{ end }}

{{ define "client_stream_method"}}
func (s *{{.ServiceName}}) {{.Name}}(stream {{.ServiceName}}_{{.Name}}Server) error {
	out := &{{.Output}}{}
	for {
		input,err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(out)
		}
		err = stub.FindAndLoad("{{.ServiceName}}","{{.Name}}",input,out)
		if err != nil {
			return err
		}
		if err := stream.Send(out); err != nil {
			return err
		}
	}
}
{{ end }}

{{ define "bidirectional_method"}}
func (s *{{.ServiceName}}) {{.Name}}(stream {{.ServiceName}}_{{.Name}}Server) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		out := &{{.Output}}{}
		err = stub.FindAndLoad("{{.ServiceName}}","{{.Name}}",in,out)
		if err != nil {
			return err
		}

		if err := server.Send(out); err != nil{
			return err
		}
	}
}
{{end}}

func Register(s *grpc.Server) {
	{{ range .Services }}
    {{ template "register" . }}
    {{ end }}
}

{{ define "register" }}
	Register{{.Name}}Server(s, &{{.Name}}{})
{{ end }}
`
