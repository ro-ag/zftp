# Real-world z/OS FTP `LIST` corpus — provenance

These fixtures exercise `hfs.ParseInfoDataset` and `hfs.ParseInfoPdsMember`
against the full range of dataset-listing and PDS-member-listing shapes a z/OS
FTP server emits in the wild.

## What these files are (and are not)

They are **project-authored test data that models the real output formats**
observed in the public sources cited below. The column geometry, the special
"state" rows (Migrated, Not Mounted, archived/non-DASD, Pseudo Directory, …),
the `+++++` track-overflow marker, `**NONE**` referred dates, quoted DSNs, and
the name-only PDS member rows are all reproductions of formats documented in the
sources. Volume serials and dataset names are neutral placeholders
(`MVSnnn`, `HLQ.PROJ.*`), not captures from any one system.

This approach is deliberate: the listing **formats** are factual/functional
(an FTP interoperability contract), but several of the richest public corpora
ship under copyleft licenses (GPL-2.0, EPL). To keep this Apache-2.0 project's
licensing trivially auditable, no third-party fixture file is copied verbatim —
the formats are reproduced as original test data with the sources cited for
verification. Sanitization removed everything that is not part of the server's
rendered listing: FTP reply codes (`125`/`200`/`227`/`250`), `ftp>` / `>>>` /
`EZA…I` client trace prefixes, byte-count trailers, and client-log timestamps.

## Format targeted

The IBM z/OS Communications Server FTP dataset list, header:

```
Volume Unit    Referred Ext Used Recfm Lrecl BlkSz Dsorg Dsname
```

(dataset name begins at column 56). The Co:Z SFTP `Tracks`-column variant, USS
`ls -l` output, JES spool listings, and IDCAMS `LISTCAT` are **different formats
and intentionally excluded.**

## Files

| File | Models |
|------|--------|
| `dataset_listings/01_canonical.txt` | normal rows: PS/PO/PO-E, F/FB/VB/VBS/VBA/VA/FBA/U recfm |
| `dataset_listings/02_special_states.txt` | Migrated, Not Mounted, Tape unit, ARCIVE "Not Direct Access Device", "< Not a DASD device >", "< File not on volume >", "Error determining attributes", "Pseudo Directory", "User catalog connector", VSAM (blank + set volume), GDG base, PATH |
| `dataset_listings/03_overflow_none_quoted.txt` | `+++++` track-overflow, `**NONE**` referred date, quoted DSNs |
| `dataset_listings/04_legacy_date.txt` | legacy OS/390 `Volume Unit Date …` header, 2-digit `mm/dd/yy` dates |
| `dataset_listings/05_listlevel2.txt` | LISTLEVEL 2 wide format (no Unit; de-overflowed track counts up to 9 digits) |
| `dataset_listings/06_coz_tracks.txt` | Co:Z SFTP `Volume Referred Ext Tracks Used …` (distinct Tracks column, no Unit, `?` unknown Used) |
| `pds_members/01_with_stats.txt` | PDS members with ISPF statistics |
| `pds_members/02_name_only.txt` | PDS members with no ISPF statistics (name-only rows) interleaved with stat rows |

## Parsing strategy (multi-format)

All dataset-listing geometries are fixed-width and positional — which is what lets
the parser split jammed columns such as the `1+++++` / `165535` Ext+Used overflow
that carries no gap between values. Each geometry is described by a
`datasetLayout` (column offsets + dataset-name start); a single engine slices a
row by the selected layout. `hfs.NewDatasetListParser(header)` inspects the header
line and picks the layout — modern (default), legacy `Date`, LISTLEVEL 2, or Co:Z
— and `ListDatasets` builds it from the listing's header automatically.
`ParseInfoDataset` is the modern-layout shortcut. `FieldDate` accepts both
`yyyy/mm/dd` and 2-digit `mm/dd/yy`; `FieldInt` accepts the `?` unknown marker
alongside `+++++` overflow.

The fixtures `04_legacy_date.txt` / `05_listlevel2.txt` / `06_coz_tracks.txt` are
column-aligned to their layout. The modern layout's offsets are pinned by the
golden fixtures; the alternate-format offsets are modeled from public samples and
would be further hardened by a real-LPAR capture of each server/LISTLEVEL.

## Sources observed (verification)

Dataset-listing formats and special rows were observed in these public sources
(harvested 2026-06-23):

- IBM z/OS Communications Server IP User's Guide & Commands (SC27-3662; V2R5 /
  V2R4 / V1R9 / OS-390 GC31-8305 editions) — `dir`/`LIST` subcommand examples,
  DATASETMODE/DIRECTORYMODE "Pseudo Directory" rows, the legacy `Date`-header
  variant. https://www.ibm.com/docs/en/zos/2.5.0?topic=data-ftp-server-application-format-connection
- Apache Commons Net `MVSFTPEntryParser` (Apache-2.0) — "ARCIVE Not Direct
  Access Device", PO/PO-E, Migrated samples.
  https://commons.apache.org/proper/commons-net/
- FluentFTP `IBMzOSParser.cs` (MIT) — `1+++++` track-overflow marker; LISTLEVEL
  0/1 vs 2 distinction. https://github.com/robinrodricks/FluentFTP
- IBM/zos-node-accessor (EPL-1.0) — large real listing with `**NONE**` dates,
  VSAM cluster `.D`/`.I`, quoted DSNs. https://github.com/IBM/zos-node-accessor
- OpenSalamander FTP plugin `listings_digest.txt` (GPL-2.0) — curated digest of
  Not Mounted / VSAM / GDG / "< File not on volume >" / "< Not a DASD device >" /
  "Error determining attributes" / "User catalog connector" / Pseudo Directory.
  https://github.com/OpenSalamander/salamander
- IndySockets/Indy FTP list fixtures, NFTP `transfer.c`, FarManager `mvs.cpp`,
  bushidocodes/keypunch (MIT) — additional Migrated/VSAM/GDG/PATH variants.
- Real session pastes: colinpaice.blog, www.mslinn.com/mainframe, WinSCP forum
  (t=24974, t=32082), IBM-MAIN listserv archive, Apache SQOOP-3225/3326/3327.

PDS member formats (ISPF-stats and name-only) observed in: Apache Commons Net
Javadoc, IBM/zos-node-accessor, WinSCP forum (t=32082, name-only `LOADLIST`),
Co:Z docs, OpenSalamander digest.

### Remaining limitations

- The alternate-format column offsets are modeled from public samples rather than
  a live capture of each server/LISTLEVEL combination. A real listing whose
  columns are positioned differently would mis-slice — the same fixed-width
  tradeoff the modern layout already makes; a real-LPAR capture of each would
  harden the offsets.
- Multivolume datasets render only their primary volser on the row (per IBM), so
  there is no multi-volser row form to parse.

## JES job-abend and STAT status formats (verification)

The JES abend classification (`classifyJesOutput` → `ErrAbend`,
`InfoJobDetail.AbendCode`) and `ServerStatus.Snapshot` parse two further server
outputs, modeled (neutral job names / placeholder values) from these public
sources (harvested 2026-06-24):

- **Abend completion codes** are alphanumeric — system `Scde` (e.g. S0C4, S806,
  SB37) or user `Ucde` (e.g. U0778) — written on `IEF450I … - ABEND=Scde Ucde
  REASON=…` / `IEF472I … SYSTEM=Scde USER=Ucde` lines, **not** inside `$HASP395`
  (which shows `… ENDED`, `ENDED nnn LINES …`, or `ENDED - RC=nnnn`). Per IBM z/OS
  MVS System Messages (IEF450I/IEF472I/IEF142I) and JES2 Messages ($HASP395). The
  `job_spool_abend.txt` fixture class was corrected from the unreal `ABEND=806` to
  `ABEND=S806`. An abended / JCL-error job's `JESINTERFACELEVEL=2` directory status
  shows `ABEND` / `(JCL Error)` (IBM z/OS Communications Server).
- **STAT status block** — the `quote stat` `211-…` multiline server-status reply
  ending `211 *** end of status ***`. A real full dump (mixing `KEY is VALUE`,
  `Server site variable KEY is set to VALUE`, free-form prose, and wrapped
  continuation lines — which is why `Snapshot` is a best-effort `Lines()` +
  `Values()` convenience, not a typed struct) is at
  https://www.mslinn.com/mainframe/3000-run_jcl.html. The `fakeStatBlock` test
  fixture models this format with a neutral host/user.
