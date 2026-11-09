package provision

import (
	"strings"

	"github.com/grimange/almadial/internal/config"
)

const DatabaseName = "asterisk"
const DatabaseUser = "almadial"
const DatabaseSecret = "/etc/almadial/secrets/database"
const DatabaseAdminSecret = "/etc/almadial/secrets/admin"
const databaseConfigPath = "/etc/my.cnf.d/zz-almadial.cnf"
const databaseConfig = "[mariadb]\nbind-address=127.0.0.1\nport=3306\nskip-name-resolve\nsocket=/var/lib/mysql/mysql.sock\ncharacter-set-server=utf8mb3\ncollation-server=utf8mb3_unicode_ci\nsql-mode=STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION\n"

// Native MariaDB catalog reference, generated from the independently imported
// pinned SQL in the separately authorized qualification database. It includes
// every table, column and index, plus triggers/routines/views (none in r4037).
// Qualification must establish this value before the stage can recognize success.
const databaseCatalogSHA = "10f5b889926a3caa1934878fd4ed675c51a2aef980cda0d09d4186f02d056af8"

type DatabaseState string

const (
	DatabaseEmpty        DatabaseState = "EMPTY"
	DatabaseComplete     DatabaseState = "COMPLETE"
	DatabasePartial      DatabaseState = "PARTIAL"
	DatabaseIncompatible DatabaseState = "INCOMPATIBLE"
	DatabaseNotVerified  DatabaseState = "NOT_VERIFIED"
)

// DatabaseSnapshot excludes resolved credentials, password hashes, SQL rows and
// free-form process output. CredentialID binds only high-entropy protected files.
type DatabaseSnapshot struct {
	Host, Foundation, ConfigInputs, ServiceID  string
	EnvironmentOK                              bool
	Config, Secrets, CredentialID              string
	Initialized, Active, Enabled               bool
	RuntimeOK, Exists, SchemaVerified          bool
	Version, CatalogSHA                        string
	Tables, Indexes, OtherObjects              int
	SeedsOK, AdminCredentialOK                 bool
	AccountExists, GrantsOK, ApplicationAuthOK bool
}

func (s DatabaseSnapshot) Digest() string { return hash(s) }
func (s DatabaseSnapshot) State() DatabaseState {
	if !s.EnvironmentOK || (s.Active && !s.SchemaVerified) {
		return DatabaseNotVerified
	}
	if !s.Initialized && !s.Active && !s.Exists && s.Tables == 0 {
		return DatabaseEmpty
	}
	if !s.Active {
		return DatabaseNotVerified
	}
	if !s.Exists || (s.Tables == 0 && s.OtherObjects == 0) {
		return DatabaseEmpty
	}
	if s.Version != "" && s.Version != "1748" {
		return DatabaseIncompatible
	}
	if s.Version == "1748" && s.CatalogSHA == databaseCatalogSHA && s.Tables == 345 && s.Indexes == 1011 && s.SeedsOK && s.AdminCredentialOK {
		return DatabaseComplete
	}
	return DatabasePartial
}
func databaseReady(s DatabaseSnapshot) bool {
	return s.EnvironmentOK && shaPattern.MatchString(s.Host) && shaPattern.MatchString(s.Foundation) && shaPattern.MatchString(s.ConfigInputs) && (s.Config == "absent" || s.Config == "valid") && (s.Secrets == "absent" || s.Secrets == "valid") && (!s.Active || s.RuntimeOK) && (!s.AccountExists || (s.GrantsOK && s.ApplicationAuthOK && s.Secrets == "valid")) && (s.State() == DatabaseEmpty || s.State() == DatabaseComplete)
}
func databaseSatisfied(s DatabaseSnapshot) bool {
	return databaseReady(s) && s.State() == DatabaseComplete && s.Config == "valid" && s.Secrets == "valid" && s.Initialized && s.Active && s.Enabled && s.RuntimeOK && s.AccountExists && s.GrantsOK && s.ApplicationAuthOK
}
func databaseReferences(c config.Config) bool {
	return c.Credentials.Database == (config.SecretRef{Type: config.File, Source: DatabaseSecret}) && c.Credentials.Admin == (config.SecretRef{Type: config.File, Source: DatabaseAdminSecret})
}
func DatabasePolicyDigest() string {
	return hash(struct{ Writer, Config, Catalog, Source, Privileges string }{"S03-only/v1", databaseConfig, databaseCatalogSHA, databaseArchiveSHA, "SELECT,INSERT,UPDATE,DELETE,LOCK TABLES ON asterisk.*; no global privilege"})
}

// S05 must call this read-only recognition contract before using the database.
// It intentionally exposes no initial import function or mutable completion bit.
func VerifyDatabaseForApplication(s DatabaseSnapshot, source DatabaseSource) error {
	if !databaseSatisfied(s) || source.Digest() == "" {
		return errDatabase
	}
	return nil
}
func databaseSafeSQLResult(raw string) string { return strings.TrimSpace(raw) }
