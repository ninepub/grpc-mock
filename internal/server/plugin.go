package server

import (
	"bufio"
	"fmt"
	"github.com/ninepub/grpc-mock/internal/types"
	"log"
	"os"
	"text/template"
)

func GenerateServiceRegister(param *types.Server, t string) {
	tmpl := template.New("server")
	tmpl, err := tmpl.Parse(t)
	if err != nil {
		log.Fatalf("template parse error %v", err)
	}
	root := param.Output + "/grpcsrv"
	os.MkdirAll(root, 0700)
	f, err := os.Create(root + "/server.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tmpl.Execute(f, param)
}

func GenerateStub(param *types.Server, t string) {
	tmpl := template.New("stub")
	tmpl, err := tmpl.Parse(t)
	if err != nil {
		log.Fatalf("template parse error %v", err)
	}
	root := param.Output + "/stub"
	os.MkdirAll(root, 0700)
	f, err := os.Create(root + "/stub.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	tmpl.Execute(f, param)
}

func GeneratePackageDef(path string) []types.PackageDef {
	packages := make([]types.PackageDef, 0, 0)
	index := 1
	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		scanner.Text()
		packages = append(packages, types.PackageDef{Path: scanner.Text(), Alias: fmt.Sprintf("%s%d", "p", index)})
		index++
	}
	file.Close()

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	err = os.Remove(path)
	if err != nil {
		log.Fatal(err)
	}
	return packages
}
