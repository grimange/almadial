# Native VICIdial Perl CSV import dependencies

Qualified on dialer01, AlmaLinux 10.2, system Perl 5.40.2, retained VICIdial SVN r4037, 2026-11-09 UTC. This is a host-specific dependency qualification, not an accepted AlmaDial installer stack. Evidence: [functional gaps acceptance](../qa/VICIDIAL_PRD_FUNCTIONAL_GAPS_01_ACCEPTANCE.md) and [dependency provenance](../qa/evidence/functional-gaps-01/cpan-provenance.json).

## Failure and required modules

The native fifth-generation loader executes `/var/www/html/vicidial/sheet2tab.pl` version 2.12 for CSV. That converter unconditionally imports both `Spreadsheet::Read` and `Spreadsheet::XLSX`, even for CSV. Missing Read caused exit 2 before output creation:

```text
Can't locate Spreadsheet/Read.pm in @INC ... at ./sheet2tab.pl line 44.
BEGIN failed--compilation aborted at ./sheet2tab.pl line 44.
```

The loader did not stop after failed conversion: its `fopen()` failed, followed by a PHP 8.3 `fgets(false)` TypeError at `admin_listloader_fifth_gen.php:3038`. TSV bypassed conversion. Restoring dependencies removed the failure on actual native CSV imports; PHP was not patched. The unhandled converter-error path remains an upstream robustness issue.

| Module | Qualified version | Method / role |
|---|---|---|
| Spreadsheet::Read | 0.95 | Pinned CPAN; required converter import and CSV dispatcher |
| Data::Peek | 0.54 | Pinned CPAN; required Read dependency, compiled XS against existing Perl |
| Spreadsheet::XLSX | 0.20 | Signed configured EPEL RPM `perl-Spreadsheet-XLSX-0.20-1.el10_2` (unconditional converter import) |
| Text::CSV_XS | 1.65 | Already installed; working CSV backend |
| Text::CSV_PP | Absent | Optional alternative backend; unnecessary with XS; not installed |
| Test::Warnings | 0.033 | Signed configured CRB RPM `perl-Test-Warnings-0.033-3.el10`; build tests |
| Test::NoWarnings | 1.06 | Signed configured CRB RPM `perl-Test-NoWarnings-1.06-10.el10`; build tests |

Configured repositories had no `perl-Spreadsheet-Read` or `perl-Data-Peek` package. No repository, framework, compiler or Perl replacement was installed. Three RPMs and two CPAN distributions were installed. Existing core/build dependencies and XLSX prerequisites were reused, including Archive::Zip and Spreadsheet::ParseExcel (the RPM requirements are retained in final-config-verification.txt). `ExtUtils::MakeMaker`, `XSLoader`, `Data::Dumper`, `Carp`, `Exporter`, `Encode`, `File::Temp`, `List::Util`, `Test::More` and existing ZIP/Excel support satisfied the relevant chain. See full required/recommended dependency metadata in the provenance JSON; optional spreadsheet formats and optional developer tools were not installed.

## CPAN provenance and installation

Both distributions are authored by HMBRAND and declare the `perl_5` license. Release metadata was obtained from MetaCPAN; downloaded archive SHA-256 values were checked against that metadata before transfer/build. This verifies consistency with the official release metadata, not an independent cryptographic author signature.

| Distribution | Official archive | SHA-256 |
|---|---|---|
| Data-Peek-0.54 | [CPAN archive](https://cpan.metacpan.org/authors/id/H/HM/HMBRAND/Data-Peek-0.54.tgz) | `4dbf2205d43fb7d963ba29902cd563a5ea6c3c6bb49a9493c40ce8b0f8572980` |
| Spreadsheet-Read-0.95 | [CPAN archive](https://cpan.metacpan.org/authors/id/H/HM/HMBRAND/Spreadsheet-Read-0.95.tgz) | `99fe77002189b6b90d9a2d502b9c597ac0beb57b4ad8531a356a5912db72107e` |

Reviewed `Makefile.PL` and declared dependencies before running the bounded builds. No interactive CPAN dependency resolver, `curl | bash`, CPAN repository configuration or upstream VICIdial modification was used. Build directories: `/tmp/gap-perl-build/Data-Peek-0.54` and `/tmp/gap-perl-build/Spreadsheet-Read-0.95`. Build each, in that order, after verifying the archives:

```sh
AUTOMATED_TESTING=1 perl Makefile.PL INSTALLDIRS=site
make
AUTOMATED_TESTING=1 make test
make install
```

Data-Peek: 15 files / 271 tests, PASS. Spreadsheet-Read: 72 files / 3,885 tests, PASS. Logs retain the precise results, including skipped tests for optional formats. Signed RPM package identities are in [provenance-installed.txt](../qa/evidence/functional-gaps-01/provenance-installed.txt).

Site installation preserved RPM-owned system Perl. Installed payloads are tracked by:

```text
/usr/local/lib64/perl5/5.40/auto/Data/Peek/.packlist
/usr/local/lib64/perl5/5.40/auto/Spreadsheet/Read/.packlist
```

Read is under `/usr/local/share/perl5/5.40/Spreadsheet/Read.pm`; Peek and its XS library are under `/usr/local/lib64/perl5/5.40/`. No service restart was needed for the converter.

## Verification

```sh
perl -MSpreadsheet::Read -MSpreadsheet::XLSX -MText::CSV_XS -MData::Peek -e \
 'print "Read=$Spreadsheet::Read::VERSION XLSX=$Spreadsheet::XLSX::VERSION CSV_XS=$Text::CSV_XS::VERSION Peek=$Data::Peek::VERSION\n"'
cd /var/www/html/vicidial
runuser -u apache -- ./sheet2tab.pl /tmp/gap-native-csv.csv /tmp/gap-native-csv.txt
```

The Apache-user conversion produced 21 tab-delimited rows, nine fields per row, from the CRLF fixture with header and 20 records. Quoted comma fields, empty fields and UTF-8 survived conversion with the converter's declared scrub rules. Double quotes and semicolons were intentionally stripped; this is native behavior, not lossless CSV roundtrip acceptance. Embedded multiline CSV fields were not tested. Conversion evidence includes [converted.tsv](../qa/evidence/functional-gaps-01/converted.tsv).

Actual native CSV uploads also passed. Correct Unicode database storage required the separate supported native System Settings option **Use Non-Latin (UTF8) = 1**. Without it, the converter produced UTF-8 correctly but PHP's default connection interpretation stored mojibake. Dependency installation alone does not establish field correctness. See the acceptance report for the isolated before/after batches and mapping.

## Upgrade and rollback

A system Perl ABI upgrade may require rebuilding Data::Peek and verifying `@INC` for Read. Revalidate the Apache execution context, both module loads and actual native CSV upload after Perl/VICIdial upgrades. Site modules are outside RPM lifecycle management: retain this manifest and packlists. Future AlmaDial prerequisite work should explicitly model the converter's unconditional XLSX import, an available CSV backend and site-module ownership; no installer work is included here.

For rollback, first deactivate synthetic campaign/lists and ensure no converter import is running. Review the two packlists and remove only their listed files, then those two packlist files. Do not recursively remove shared Perl directories. Review a `dnf remove --assumeno` transaction for only the three newly installed RPMs before removal; stop if it would remove unrelated packages. Do not remove the pre-existing Text::CSV_XS or system Perl. Rollback restores the known CSV dependency failure and is not the recommended final state. It does not require an Asterisk/MariaDB restart or database restore.
