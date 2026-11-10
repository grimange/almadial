package provision

import "strings"

// M0-03 native closure, with M0-06 speech/AGI and recording readiness.
// Load order matters: res_speech exports symbols used by res_agi.
func AsteriskModules() []string {
	return strings.Fields(`res_crypto res_http_websocket res_pjproject res_sorcery_config res_sorcery_memory res_sorcery_astdb res_sorcery_memory_cache res_pjsip res_pjsip_config_wizard res_pjsip_authenticator_digest res_pjsip_endpoint_identifier_user res_pjsip_endpoint_identifier_ip res_pjsip_registrar res_pjsip_session res_pjsip_sdp_rtp res_pjsip_nat res_pjsip_transport_websocket res_rtp_asterisk res_srtp res_timing_timerfd res_pjsip_pubsub chan_pjsip pbx_config app_echo bridge_simple bridge_builtin_features bridge_builtin_interval_features codec_ulaw codec_alaw codec_resample format_pcm format_wav func_callerid func_volume func_jitterbuffer bridge_softmix app_confbridge res_speech res_agi app_mixmonitor`)
}

const asteriskWork = "/var/lib/almadial/asterisk-s04"
const asteriskTree = asteriskWork + "/asterisk-22.12.0"
const asteriskUnit = `[Unit]
Description=AlmaDial pinned Asterisk 22
After=network.target
[Service]
Type=simple
User=asterisk
Group=asterisk
RuntimeDirectory=asterisk
RuntimeDirectoryMode=0750
UMask=0027
ExecStart=/usr/sbin/asterisk -f -C /etc/asterisk/asterisk.conf
ExecStop=/usr/sbin/asterisk -rx "core stop now"
Restart=on-failure
RestartSec=3
TimeoutStopSec=30
[Install]
WantedBy=multi-user.target
`

func asteriskFiles() map[string]string {
	modules := "[modules]\nautoload=no\n"
	for _, m := range AsteriskModules() {
		modules += "load = " + m + ".so\n"
	}
	return map[string]string{
		"asterisk":   "[directories]\nastetcdir => /etc/asterisk\nastmoddir => /usr/lib/asterisk/modules\nastvarlibdir => /var/lib/asterisk\nastdbdir => /var/lib/asterisk\nastkeydir => /etc/asterisk/keys\nastdatadir => /var/lib/asterisk\nastagidir => /var/lib/asterisk/agi-bin\nastspooldir => /var/spool/asterisk\nastrundir => /run/asterisk\nastlogdir => /var/log/asterisk\n[options]\nrunuser = asterisk\nrungroup = asterisk\n",
		"modules":    modules,
		"manager":    "[general]\nenabled=yes\nport=5038\nbindaddr=127.0.0.1\ndisplayconnects=no\n",
		"http":       "[general]\nenabled=yes\nbindaddr=127.0.0.1\nbindport=8088\ntlsenable=no\n",
		"pjsip":      "; M3 owns endpoint and transport acceptance. No SIP listener in S04.\n",
		"extensions": "[general]\nstatic=yes\nwriteprotect=yes\n[default]\n",
		"confbridge": "[default_bridge]\ntype=bridge\nmixing_interval=20\n[default_user]\ntype=user\n",
		"logger":     "[general]\n[logfiles]\nconsole=notice,warning,error\n",
	}
}
func AsteriskBuildPolicyDigest() string {
	return hash(struct {
		Modules     []string
		Files       map[string]string
		Unit, Build string
	}{AsteriskModules(), asteriskFiles(), asteriskUnit, "22.12.0/bundled-pjproject-2.17/without-dahdi/BUILD_NATIVE-disabled/no-sounds/j4/staged-install/v1"})
}

type AsteriskModule struct {
	Name                             string
	Source, Built, Installed, Loaded bool
	Capability                       string
	Pass                             bool
}
type AsteriskSnapshot struct {
	Host, Foundation, Database, Source, Toolchain, Build, Install, Configuration, Unit, ServiceID string
	EnvironmentOK, Active, Enabled, RuntimeOK, RestartVerified                                    bool
	Modules                                                                                       []AsteriskModule
}

func (s AsteriskSnapshot) Digest() string { return hash(s) }
func asteriskReady(s AsteriskSnapshot) bool {
	if !s.EnvironmentOK || !shaPattern.MatchString(s.Host) || !shaPattern.MatchString(s.Foundation) || !shaPattern.MatchString(s.Database) || !shaPattern.MatchString(s.Source) || !shaPattern.MatchString(s.Toolchain) {
		return false
	}
	if s.Install != "resources-missing" && s.Install != "libraries-missing" && s.Install != "absent" && !shaPattern.MatchString(s.Install) {
		return false
	}
	for _, v := range []string{s.Build, s.Configuration, s.Unit} {
		if v != "absent" && !shaPattern.MatchString(v) {
			return false
		}
	}
	return true
}
func asteriskSatisfied(s AsteriskSnapshot) bool {
	if !asteriskReady(s) || s.Build == "absent" || !shaPattern.MatchString(s.Install) || s.Configuration == "absent" || s.Unit == "absent" || !s.Active || !s.Enabled || !s.RuntimeOK || !s.RestartVerified || s.ServiceID == "" || len(s.Modules) != len(AsteriskModules()) {
		return false
	}
	for j, m := range s.Modules {
		if m.Name != AsteriskModules()[j] || !m.Source || !m.Built || !m.Installed || !m.Loaded || !m.Pass {
			return false
		}
	}
	return true
}

func AsteriskPolicyDigest() string {
	return hash(struct{ Build, Install string }{AsteriskBuildPolicyDigest(), "versioned-runtime-libraries-xml-accepted-parent-v4"})
}
