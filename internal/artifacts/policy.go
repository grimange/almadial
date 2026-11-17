package artifacts

// Reviewed M0 source contract, independent of caller-supplied manifests.
// Changes require provenance review; candidate remains non-installable.
var approvedLeaves = map[string]leaf{
	"schema_version":                 {"1", "int"},
	"profile":                        {"almalinux10-single", "str"},
	"status":                         {"candidate", "str"},
	"os.id":                          {"almalinux", "str"},
	"os.major":                       {"10", "int"},
	"os.release":                     {"10.2", "float"},
	"os.arch":                        {"x86_64", "str"},
	"os.image_sha256":                {"cbee4726fc0f5a60694bb4f289003cc221e715dfbacebcce76b54daf386d8eca", "str"},
	"asterisk.major_candidate":       {"22", "int"},
	"asterisk.version":               {"22.12.0", "str"},
	"asterisk.source":                {"https://downloads.asterisk.org/pub/telephony/asterisk/asterisk-22.12.0.tar.gz", "str"},
	"asterisk.sha256":                {"f8b2ba98aac6c4ff3376bc0a2868bea6ed577ca684efb7ec739ae6e8ce6e35e2", "str"},
	"asterisk.signing_fingerprint":   {"F2FC93DB7587BD1FB49E045A5D984BE337191CE7", "str"},
	"vicidial.source":                {"svn://svn.eflo.net:3690/agc_2-X/trunk", "str"},
	"vicidial.revision":              {"4037", "int"},
	"vicidial.exported_tree_sha256":  {"04a41c36c381981853cd0fc49a82fea9dda611cf39b336e9e8d92d459551eaed", "str"},
	"vicidial.schema_version":        {"1748", "int"},
	"vicidial.provenance_state":      {"official_https_snapshot_byte_matched_to_pinned_export", "str"},
	"vicidial.authenticated_archive": {"https://www.vicidial.org/svn_trunk_nightly/vicidial-trunk-2026-11-05.zip", "str"},
	"vicidial.archive_sha256":        {"ebdd6fab3c9bd56c5ed84f79a3118153c1c9ed38508f24a83a1b71ad803a4058", "str"},
	"vicidial.provenance_evidence":   {"docs/compatibility/m0-04-vicidial-runtime.md", "str"},
	"webrtc.name":                    {"VICIphone", "str"},
	"webrtc.revision":                {"24e98fccfa7b7079fff970e02fca5f002ac1f2cc", "str"},
	"webrtc.sha256":                  {"f42c7ec8a8114b71fae74c9ffa8bfa3927fc789609ac6ef78a9f897c6f3c8ac8", "str"},
	"webrtc.sipjs_version":           {"0.20.1", "str"},
	"webrtc.sipjs_sha256":            {"9eb6524ac04ee19ee106e835a813c0c5cb333478df6a2c71b382ff6cbd50164b", "str"},
	"pjproject.version":              {"2.17", "float"},
	"pjproject.source":               {"https://raw.githubusercontent.com/asterisk/third-party/52e2f32de1b11e40844bb007749a327862f75275/pjproject/2.17/pjproject-2.17.tar.bz2", "str"},
	"pjproject.sha256":               {"04b2eb1f0f01aa0ad1945b167171843448a51aa6b7c3e806496d434f13a112b7", "str"},
	"perl_net_telnet.version":        {"3.05", "float"},
	"perl_net_telnet.source":         {"https://cpan.metacpan.org/authors/id/J/JR/JROGERS/Net-Telnet-3.05.tar.gz", "str"},
	"perl_net_telnet.sha256":         {"677f68ba2cd2a824fae323fa82e183bf7e3d03c3c499c91d923bd6283796a743", "str"},
	"verification.runtime_state":     {"NOT_RUN", "str"},
	"verification.fixture_id":        {"TBD-M0", "str"},
	"verification.evidence_digest":   {"TBD-M0", "str"},
	"mariadb.nevra":                  {"mariadb-server-3:10.11.18-1.el10_2.alma.2.x86_64", "str"},
	"mariadb.repository":             {"appstream", "str"},
	"php.nevra":                      {"php-fpm-0:8.3.36-1.el10_2.x86_64", "str"},
	"php.repository":                 {"appstream", "str"},
	"perl.nevra":                     {"perl-4:5.40.2-515.el10_2.x86_64", "str"},
	"perl.repository":                {"appstream", "str"},
	"apache.nevra":                   {"httpd-0:2.4.63-13.el10_2.8.x86_64", "str"},
	"apache.repository":              {"appstream", "str"},
	"srtp.nevra":                     {"libsrtp-devel-0:2.3.0-16.el10.x86_64", "str"},
	"srtp.repository":                {"crb", "str"},
	"jansson.nevra":                  {"jansson-devel-0:2.14-3.el10.x86_64", "str"},
	"jansson.repository":             {"crb", "str"},
}

// The 10.3 distribution refresh retains the accepted application-source and
// server package identities; only build/development dependencies changed.
func manifestLeaves(release string) map[string]leaf {
	if release == "10.2" {
		return approvedLeaves
	}
	if release != "10.3" {
		return nil
	}
	out := make(map[string]leaf, len(approvedLeaves))
	for k, v := range approvedLeaves {
		out[k] = v
	}
	out["os.release"] = leaf{"10.3", "float"}
	out["os.image_sha256"] = leaf{"845d3a4e41fde973bdbf9763b62a88538c1a1fadc2332a89f6ea01e2c17c78ee", "str"}
	return out
}
