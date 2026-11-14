// Package releaseassets carries public, pinned qualification inputs with the CLI.
// No upstream application source, credentials or private keys are bundled.
package releaseassets

import _ "embed"

//go:embed stack.lock.yaml
var Manifest []byte

//go:embed asterisk-keyring.gpg
var Keyring []byte

//go:embed asterisk.asc
var Signature []byte

// Origins is the release-reviewed private lab source retention policy.
//
//go:embed acquisition-origins.json
var Origins []byte
