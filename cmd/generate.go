package main

//go:generate echo "Generating SQLC files..."
//go:generate bash -c "export PATH=$$PATH:~/go/bin && sqlc generate -f ../storage/sqlc.yaml"
//go:generate echo "SQLC files generated"

//go:generate echo "Generating templ files..."
//go:generate bash -c "export PATH=$$PATH:~/go/bin && templ generate -path ../views"
//go:generate bash -c "export PATH=$$PATH:~/go/bin && templ generate -path ../components"
//go:generate echo "templ files generated"

//go:generate echo "CSS generation handled by Air pre-command (npm run build:css)"

// This file contains go:generate directives that will process all templ files
// and generate SQLC code in this project. To generate the Go code from the
// templ templates and SQLC queries, run:
//
// go generate ./...
//
// from the project root directory.
