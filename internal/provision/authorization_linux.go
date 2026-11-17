//go:build linux

package provision

// Empty mode and the historical stage scopes retain their original meaning and
// JSON bytes. Standard authorization never synthesizes managed evidence. The
// executor independently requires the exact plan digest and fresh host binding.
const StandardAuthorization = "standard"

func (a InstallAuthorization) valid(host string) bool {
	if a.Host == "" || a.Host != host {
		return false
	}
	if a.Mode == StandardAuthorization {
		return a.TaskID == "" && a.SnapshotReference == "" && a.OperatorEvidenceSHA256 == "" && !a.CandidateAcknowledged && !a.ManagementRecoveryVerified
	}
	return a.Mode == "" && (a.TaskID == "xu7wxvch94ii2iw38hpb8d58" || a.TaskID == "it8aq4utqix0uz8sjgxrtssk" || a.TaskID == "a1doi4qsyn1lspqtydw84smh") && shaPattern.MatchString(a.SnapshotReference) && shaPattern.MatchString(a.OperatorEvidenceSHA256) && a.CandidateAcknowledged && a.ManagementRecoveryVerified
}

func (a InstallAuthorization) scope(legacy string) string {
	if a.Mode == StandardAuthorization {
		return StandardAuthorization
	}
	return legacy
}

func stageAuthorization(scope, legacy, snapshot, evidence string, candidate, recovery, requireRecovery bool) bool {
	if scope == StandardAuthorization {
		return snapshot == "" && evidence == "" && !candidate && !recovery
	}
	return scope == legacy && shaPattern.MatchString(snapshot) && shaPattern.MatchString(evidence) && candidate && (!requireRecovery || recovery)
}
