package proto

import (
	"bytes"
	"fmt"
	"github.com/ninepub/grpc-mock/internal/types"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/golang/protobuf/protoc-gen-go/plugin"
	"golang.org/x/tools/imports"
)

const (
	methodTypeStandard = "standard"
	// service to client stream
	methodTypeServerStream = "server-stream"
	// client to service stream
	methodTypeClientStream  = "client-stream"
	methodTypeBidirectional = "bidirectional"
)

func generateServer(param *types.Proto, t string) error {
	if param.Writer == nil {
		param.Writer = os.Stdout
	}

	tmpl := template.New("service")
	tmpl, err := tmpl.Parse(t)
	if err != nil {
		return fmt.Errorf("template parse %v", err)
	}

	buf := new(bytes.Buffer)
	err = tmpl.Execute(buf, param)
	if err != nil {
		return fmt.Errorf("template execute %v", err)
	}

	byt := buf.Bytes()
	bytProcessed, err := imports.Process("", byt, nil)
	if err != nil {
		return fmt.Errorf("formatting: %v \n%s", err, string(byt))
	}

	_, err = param.Writer.Write(bytProcessed)
	return err
}

func resolveDependencies(servicePkg string, protos []*descriptor.FileDescriptorProto) map[string]string {
	depsFile := make([]string, 0)
	for _, p := range protos {
		depsFile = append(depsFile, p.GetDependency()...)
	}
	deps := map[string]string{}
	aliases := map[string]bool{}
	aliasNum := 1
	for _, dep := range depsFile {
		for _, p := range protos {
			alias, pkg := getGoPackage(p)

			// skip whether its not intended deps
			// or has empty Go package
			if p.GetName() != dep || pkg == "" || pkg == servicePkg {
				continue
			}
			// in case of found same alias
			if ok := aliases[alias]; ok {
				alias = fmt.Sprintf("%s%d", alias, aliasNum)
				aliasNum++
			} else {
				aliases[alias] = true
			}
			deps[pkg] = alias
		}
	}
	return deps
}

func getOutFile(proto *descriptor.FileDescriptorProto, pkg string) (string, string) {
	r := regexp.MustCompile(`(.+)/([^/]+)`)
	matches := r.FindStringSubmatch(*proto.Name)
	return matches[1], matches[1] + "/" + pkg + "_impl.go"
}

func getGoPackage(proto *descriptor.FileDescriptorProto) (alias string, goPackage string) {
	goPackage = proto.GetOptions().GetGoPackage()
	if goPackage == "" {
		return
	}

	// support go_package alias declaration
	// https://github.com/golang/protobuf/issues/139
	if splits := strings.Split(goPackage, ";"); len(splits) > 1 {
		goPackage = splits[0]
		alias = splits[1]
	} else {
		splitSlash := strings.Split(proto.GetName(), "/")
		split := strings.Split(splitSlash[len(splitSlash)-1], ".")
		alias = split[0]
	}
	return
}

func inArray(val string, array []string) (exists bool, index int) {
	exists = false
	index = -1
	for i, v := range array {
		if val == v {
			index = i
			exists = true
			return
		}
	}
	return
}

// change the structure also translate method type
func extractServices(param *types.Proto, protos []*descriptor.FileDescriptorProto) {
	svcTmp := make([]types.Service, 0, 0)
	var p *descriptor.FileDescriptorProto
	for _, p = range protos {
		if generate, _ := inArray(p.GetName(), param.FilesToGenerate); generate {
			for _, svc := range p.GetService() {
				s := types.Service{Name: svc.GetName()}
				methods := make([]types.MethodTemplate, len(svc.Method))
				for j, method := range svc.Method {
					tipe := methodTypeStandard
					if method.GetServerStreaming() && !method.GetClientStreaming() {
						tipe = methodTypeServerStream
					} else if !method.GetServerStreaming() && method.GetClientStreaming() {
						tipe = methodTypeClientStream
					} else if method.GetServerStreaming() && method.GetClientStreaming() {
						tipe = methodTypeBidirectional
					}
					_, pkg := getGoPackage(p)
					methods[j] = types.MethodTemplate{
						Name:        strings.Title(*method.Name),
						ServiceName: svc.GetName(),
						Input:       getMessageType(pkg, protos, p.GetDependency(), method.GetInputType()),
						Output:      getMessageType(pkg, protos, p.GetDependency(), method.GetOutputType()),
						MethodType:  tipe,
					}
				}
				s.Methods = methods
				svcTmp = append(svcTmp, s)
			}
		}

	}
	if len(svcTmp) != 0 {
		_, pkg := getGoPackage(p)
		path, outFile := getOutFile(p, pkg)
		param.Services = svcTmp
		param.Package = pkg
		param.PackagePath = path
		param.OutFile = outFile
	}
}

func getMessageType(servicePkg string, protos []*descriptor.FileDescriptorProto, deps []string, tipe string) string {
	split := strings.Split(tipe, ".")[1:]
	targetPackage := strings.Join(split[:len(split)-1], ".")
	targetType := split[len(split)-1]
	for _, dep := range deps {
		for _, p := range protos {
			if p.GetName() != dep || p.GetPackage() != targetPackage {
				continue
			}

			for _, msg := range p.GetMessageType() {
				if msg.GetName() == targetType {
					alias, pkg := getGoPackage(p)
					if pkg != servicePkg && alias != "" {
						alias += "."
						return fmt.Sprintf("%s%s", alias, msg.GetName())
					} else {
						return msg.GetName()
					}

				}
			}
		}
	}
	return targetType
}

func writePackageFile(param *types.Proto) {
	f, err := os.OpenFile("package", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open the file %v", err)
	}
	_, err = f.WriteString(fmt.Sprintf("%s%s\n", param.PackageSuffix, param.PackagePath))
	if err != nil {
		log.Fatalf("Failed to write package data to file %v", err)
	}
	err = f.Close()
	if err != nil {
		log.Fatalf("Failed close the file %v", err)
	}
}

func GenerateGrpcCode(t string) {
	gen := generator.New()
	byt, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Failed to read input: %v", err)
	}

	err = proto.Unmarshal(byt, gen.Request)
	if err != nil {
		log.Fatalf("Failed to unmarshal proto: %v", err)
	}

	gen.CommandLineParameters(gen.Request.GetParameter())

	buf := new(bytes.Buffer)

	param := &types.Proto{
		Writer:          buf,
		PackageSuffix:   gen.Param["pkg-suffix"],
		FilesToGenerate: gen.Request.GetFileToGenerate(),
	}

	protos := gen.Request.ProtoFile
	extractServices(param, protos)
	if len(param.Services) != 0 {
		param.Dependencies = resolveDependencies(param.Package, protos)
		err := generateServer(param, t)
		if err != nil {
			log.Fatalf("Failed to generate server %v", err)
		}
		gen.Response.File = []*plugin_go.CodeGeneratorResponse_File{
			{
				Name:    proto.String(param.OutFile),
				Content: proto.String(buf.String()),
			},
		}

		data, err := proto.Marshal(gen.Response)
		if err != nil {
			gen.Error(err, "failed to marshal output proto")
		}
		_, err = os.Stdout.Write(data)
		if err != nil {
			gen.Error(err, "failed to write output proto")
		}
		writePackageFile(param)
	}
}
