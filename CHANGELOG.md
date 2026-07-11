# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.0] - 2026-07-12

### Security
- SMTP credentials are no longer sent over an unencrypted connection by
  default. When `TLS` is `TLSNone` and `Username` is set, `Send` returns
  `ErrInsecureAuth` unless the new `SMTPConfig.AllowInsecureAuth` is set, which
  is intended only for local testing. This can break a configuration that
  relied on cleartext auth; move it to `TLSStartTLS`/`TLSImplicit` or opt in.

### Fixed
- A failed `QUIT` after the server accepted the message (context cancel, dirty
  close) is no longer reported as a delivery failure, so a retrying caller does
  not send a duplicate.
- A `RCPT` error for a `Bcc` recipient no longer includes the address, so a
  caller that logs or shows the error cannot leak a blind recipient.
- `crypto/rand` failure when generating a MIME boundary or `Message-Id` is now
  returned from `build` instead of being ignored, so a body can never be
  framed with a deterministic (all-zero) boundary.
- An attachment filename is bounded to 200 bytes (keeping a short extension),
  so a very long name cannot produce a `Content-Type`/`Content-Disposition`
  line a MIME parser rejects.
- `CaptureSender.Messages` and `Last` now return a deep copy, so a caller that
  mutates the returned messages cannot corrupt the captured state.

## [0.1.0]

First release: RFC 5322/MIME message building and SMTP delivery on the standard
library, with logger and capture transports for development and tests.
