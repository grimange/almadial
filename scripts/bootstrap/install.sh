#!/usr/bin/env bash
# Rendered by prepare_release.py: inspect this script before running it.
set -euo pipefail
umask 077
readonly RELEASE_VERSION='alm46-native-rc1'
readonly DEFAULT_BINARY_URL=''
readonly BINARY_SHA256='a58235bdc435cbd48c43e444aead04f77415af682743055b550535efdca62cc3'
# A mirror selects a location, never a different release or trusted checksum.
readonly BINARY_URL="${ALMADIAL_BINARY_URL:-$DEFAULT_BINARY_URL}"
fail() { printf 'AlmaDial bootstrap: %s\n' "$*" >&2; exit 1; }
[[ ${EUID} == 0 ]] || fail 'run as root on the authorized target'
for utility in curl sha256sum uname mktemp install mv rm; do
  command -v "$utility" >/dev/null || fail "required bootstrap utility missing: $utility"
done
[[ -f /etc/os-release ]] || fail 'cannot identify OS'
# This is an OS-owned local file, not a downloaded script.
. /etc/os-release
[[ ${ID:-} == almalinux && ${VERSION_ID:-} == 10.2 ]] || fail 'supported OS is AlmaLinux 10.2'
[[ $(uname -m) == x86_64 ]] || fail 'supported architecture is x86_64'
[[ $BINARY_SHA256 =~ ^[a-f0-9]{64}$ ]] || fail 'release is unconfigured; obtain an approved rendered bootstrap'
[[ -n $BINARY_URL ]] || fail 'no binary endpoint configured; obtain an approved CLI URL from the release owner and set ALMADIAL_BINARY_URL (the release checksum stays pinned)'
# Match the release generator policy: no credentials, query, fragment or shell syntax.
[[ $BINARY_URL =~ ^https://[A-Za-z0-9][A-Za-z0-9.:/_-]*$ ]] || fail 'ALMADIAL_BINARY_URL must be an approved HTTPS URL without credentials, query or fragment'
trust_args=()
if [[ -n ${ALMADIAL_CA_FILE:-} ]]; then
  [[ -f $ALMADIAL_CA_FILE && -r $ALMADIAL_CA_FILE ]] || fail 'ALMADIAL_CA_FILE must name a readable approved public CA file'
  trust_args=(--cacert "$ALMADIAL_CA_FILE")
fi
[[ -d /usr/local/sbin && ! -L /usr/local/sbin ]] || fail '/usr/local/sbin must be a real directory'
staging=$(mktemp -d /usr/local/sbin/.almadial-bootstrap.XXXXXXXX)
trap 'rm -rf -- "$staging"' EXIT
# GitHub release assets redirect to its HTTPS asset CDN. Bound redirects and
# prohibit transport downgrade; the release-pinned hash still gates execution.
curl --fail --silent --show-error --location --max-redirs 3 --proto '=https' --proto-redir '=https' --tlsv1.2 --connect-timeout 15 --max-time 180 --max-filesize 67108864 "${trust_args[@]}" --output "$staging/almadial" "$BINARY_URL"
printf '%s  %s\n' "$BINARY_SHA256" "$staging/almadial" | sha256sum --check --status || fail 'binary checksum mismatch; nothing installed or executed'
# Revision acquisition uses upstream SVN with the accepted complete tree digest.
# Obtain only the utility from signed OS repositories; never run source scripts.
if [[ ! -x /usr/bin/svn ]]; then
  command -v dnf >/dev/null || fail 'signed AlmaLinux subversion package is required; dnf is unavailable'
  dnf install -y --setopt=gpgcheck=1 --setopt=localpkg_gpgcheck=1 subversion || fail 'cannot install signed SVN acquisition utility; existing CLI preserved'
  [[ -x /usr/bin/svn ]] || fail 'SVN acquisition utility missing after package installation'
fi
install -o root -g root -m 0755 "$staging/almadial" "$staging/verified"
# Only verified bytes are executed. Check before replacing an existing CLI.
"$staging/verified" --version || fail 'verified CLI launch failed; existing executable preserved'
mv -fT -- "$staging/verified" /usr/local/sbin/almadial
printf 'AlmaDial %s installed in /usr/local/sbin/almadial\nNext: almadial install\n' "$RELEASE_VERSION"
