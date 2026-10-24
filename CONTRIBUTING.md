# Contributing

Thanks for taking the time to improve Copy Text Pos.

## Before you start

- Use Windows 10/11 for end-to-end testing. The app is Windows-only.
- Keep the core loop fast. A hotkey action should normally finish in well under
  two seconds.
- Never add telemetry, network access or automatic uploads.
- Do not log selected text, and do not store it in the bridge snapshot.

## Development

```powershell
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go test ./...
.\build.ps1 -Version 0.0.0-dev -Console -Output bin\copy-text-pos-dev.exe
```

The console build keeps `stderr` visible and is much easier to debug than the
release `windowsgui` build.

Useful tests:

```powershell
go test ./internal/textmatch -v
go test ./internal/locate -v
```

The optional local integration test reads actual Cursor workspace metadata:

```powershell
$env:CTP_INTEGRATION = "1"
go test ./internal/locate -run TestResolverIntegrationCursor -v
```

## Pull requests

- Keep changes focused; describe the problem and the approach.
- Add or update tests for parsing, matching, resolution and UI behavior where it
  can be tested without a desktop.
- If you change a window-title format or add an editor adapter, include a
  sanitized title example and the expected path/range.
- Run `gofmt` on Go code and `node --check integrations/vscode/extension.js`.
- Screenshots are welcome for visible settings changes, but do not include real
  project paths or secrets.

## Reporting bugs

Please include:

- editor and version;
- Windows version;
- the window title pattern (redact if necessary);
- the tail of `%LOCALAPPDATA%\CopyTextPos\copy-text-pos.log`;
- whether the VS Code bridge was active.

Do not paste selected code or secrets into a public issue.

## License

By contributing, you agree that your contribution is licensed under the MIT
License (see [LICENSE](LICENSE)).
