module github.com/get-skipper/skipper-go/ginkgo

go 1.21

require (
	github.com/get-skipper/skipper-go/core v0.0.0-00010101000000-000000000000
	github.com/onsi/ginkgo/v2 v2.19.0
	github.com/onsi/gomega v1.34.0
)

replace github.com/get-skipper/skipper-go/core => ../core
