# Security policy

Copy Text Pos is a local-only Windows tray application. It does not make network
requests and does not collect telemetry.

## Reporting a vulnerability

Please do not open a public issue for security-sensitive reports. Contact the
maintainer privately through the repository's security advisory feature or the
contact address listed on the repository profile.

Include:

- affected version;
- reproduction steps;
- impact (for example, clipboard data loss, local privilege boundary, settings
  token exposure);
- whether the VS Code bridge was installed.

## Scope notes

- The settings server binds to `127.0.0.1` and requires a random per-process
  token in the URL. Pages refuse cross-origin POSTs and set a strict CSP.
- The app cannot send synthetic input into elevated windows because of Windows
  UIPI. Running both apps at the same integrity level is intentional.
- Sending synthetic `Ctrl+C` may be observed by Windows Clipboard History or a
  third-party clipboard manager. This is a platform behavior, not a hidden data
  channel.
- The bridge snapshot never contains selected text.
