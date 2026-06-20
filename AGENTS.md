# AGENTS.md

## Build conventions

- Always cross-compile from WSL for Windows: `GOOS=windows GOARCH=amd64 go build`
- Output test builds to `build/` — never to the repository root.
- Delete the binary after verifying the build succeeded.
- Example:
  ```
  GOOS=windows GOARCH=amd64 go build -o build/madori-test-build.exe ./cmd/madori/ && rm build/madori-test-build.exe && echo "Build OK"
  ```
  Confirm the output is `Build OK` with no errors.
