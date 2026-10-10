# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.7.5] (2026-10-10)

### Added
- Multiple Master fonts: `type1` reads them, and `Font.Instantiate`
  turns one into an ordinary single-master font at a given design
  coordinate.  `Font.VariationAxes` reports the design axes and the
  default coordinate.  Writing a Multiple Master font writes its
  default instance.
- `type1.Font.CapHeightPDF` and `Font.XHeightPDF` measure cap height
  and x-height from the glyph outlines, since the Type 1 format
  records neither.
- `type1.CheckFontName`, `RepairFontName`, `CheckGlyphName`,
  `RepairGlyphName` and `MaxFontNameLen` validate and repair
  PostScript font and glyph names.  The Type 1 and AFM readers and
  writers use them, so a name a font file cannot carry is never
  emitted.
- `type1.PrivateDict.Repair` and `Validate` repair and validate a
  font's Private dictionary (`BlueValues`, `OtherBlues`, `BlueScale`,
  `BlueShift`, `BlueFuzz`, `StdHW`, `StdVW`).
- `ParseNumber` is exported, parsing a PostScript number token
  following PLRM 3.1 syntax.

### Changed
- `afm.Metrics.Kern` now holds `[]KernPair` instead of `[]*KernPair`,
  and `KernPair.Adjust` is `float64` instead of `funit.Int16`,
  matching the AFM format's real-valued kerning adjustments.
- `afm.Metrics` gains `FamilyName` and `Weight` fields, read from and
  written to the file directly instead of being derived by splitting
  `FullName` at its first space.
- `type1.PrivateDict.BlueValues`, `OtherBlues`, `BlueShift` and
  `BlueFuzz` are now `float64` rather than `funit.Int16`/`int32`,
  since CFF allows fractional and larger values there.  The Type 1
  reader rounds and range-limits these on read, and the writer
  refuses a value it cannot write as a Type 1 integer.
- `FontInfo.PostScriptName` is removed; it only returned `FontName`.
- The scanner's number parser now follows PLRM 3.2: a token
  that is not a number becomes an operator name, a decimal integer
  beyond the implementation limit becomes a real, and a real beyond
  the limit raises `limitcheck`; a radix number such as
  `16#FFFFFFFFFFFFFFFF` keeps its twos-complement value.
- The interpreter's `bind` no longer substitutes a builtin operator
  for a literal name, only for an executable name, since a literal
  name is data rather than an operator reference.
- The Type 1 `closepath` charstring command no longer moves the
  current point; drawing that continues after it now starts a new
  sub-path, matching the Type 1 format rather than the PostScript
  `closepath` operator.
- The charstring operand-stack limit is raised to 98 for Multiple
  Master fonts, to admit a 16-master blend of 6 values; other fonts
  keep the limit of 24.

### Fixed
- `cvi` and `cvr` parse strings with the scanner's number syntax,
  accepting radix numbers and surrounding white space and rejecting
  forms only `strconv` accepts.
- Integer overflow in the arithmetic operators no longer wraps
  silently, and `round` no longer rounds up values just below one half
  or odd values beyond 2^52.
- AFM reading no longer fails on a malformed file: an unusable number
  is replaced by its default, a name the format cannot carry is
  repaired, and an entry that survives neither is dropped, so one bad
  field no longer costs the rest of the file.  A size claimed by a
  repeated section header no longer triggers a fresh allocation each
  time it recurs (previously up to 2.9 GB for 1 MB of repeated
  headers), and lines, kerning pairs and ligatures are bounded so that
  what is read can always be written back.
- `type1.Glyph.IsBlank` no longer panics on a glyph with no outline.

## [v0.7.4] (2026-06-25)

### Added
- `funit.Uint16` for non-negative font-design-unit quantities (the
  OpenType UFWORD type), such as advance widths.

### Changed
- The interpreter and Type 1 font reader allocate against an explicit
  memory budget, bounding peak allocation on untrusted input.

### Fixed
- Type 1 charstring decoding is bounded to prevent a subroutine
  fan-out denial of service.
- CMap and Type 1 readers validate single-entry input.

## [v0.7.3] (2026-05-19)

### Changed
- Whitespace handling now matches PLRM 3.1 exactly: only NUL, TAB,
  LF, FF, CR, and SP separate tokens.  Control bytes below 32 that
  the previous parser silently treated as whitespace are now
  rejected.

### Fixed
- Scanner now bounds the size of strings, names/operators, and DSC
  comment values to defend against parser-bomb input.  Strings and
  names error with `eLimitcheck` at the cap; DSC values are silently
  truncated so the surrounding parse continues.
- Float overflow in `parseNumber` now yields `eLimitcheck` instead of
  falling through to a syntax error (`strconv.ParseFloat` wraps
  `ErrRange` in `*NumError`, which the previous comparison missed).
- Literal-string scanner: a CR followed by multiple LFs no longer
  swallows every LF; only the LF completing a CR-LF pair is consumed.
- `findresource` no longer panics on an unchecked type assertion when
  the resource catalogue contains a non-Dict value.

## [v0.7.2] (2026-05-11)

### Changed
- type1: reading a font now substitutes a blank glyph (preserving the advance width) when a charstring is malformed, instead of failing the whole font load.

### Fixed
- Fixed integer-overflow panics in `copy` and `putinterval` on near-`MaxInt` arguments from malicious PostScript input.
- type1: rejected negative `lenIV` values and added bounds and arity checks on `callothersubr` to close panic vectors in the Type 1 charstring decoder.

## [v0.7.1] (2026-03-31)

### Changed
- type1: `FontInfo` is now optional when reading Type 1 fonts.
- Dropped `golang.org/x/exp` dependency in favour of standard library.

### Fixed
- type1: flex handling in charstring decoder.

## [v0.7.0] (2025-01-25)

### Changed
- **API Change**: type1.Glyph now stores outlines as `Outline *path.Data` instead of `Cmds []GlyphOp`, using the standardized path representation from go-geom
- **API Change**: type1.Glyph.Path() iterator now uses `vec.Vec2` instead of `path.Point` for point coordinates
- **AFM writer** now outputs floating-point values with full precision instead of truncating to integers
- Updated dependency seehuhn.de/go/geom for new vector and path types

### Fixed
- **AFM parser** now handles malformed numeric values gracefully by skipping invalid entries instead of returning errors
- **AFM parser** silently ignores extremely large values, NaN, and infinities
- Fuzzing failure addressed

### Removed
- **GlyphOp, GlyphOpType**, and Op* constants removed (use `path.Command` and `path.Data` instead)

## [v0.6.0] (2025-06-30)

### Added
- **25 new PostScript builtin operators** including mathematical functions (atan, cos, sin, sqrt, ln, log, exp, floor, ceiling, round, truncate, neg), arithmetic operators (div, idiv, mod), comparison operators (ge, gt, le, lt), bitwise operations (bitshift, xor), and type conversion functions (cvi, cvr)
- **IsBlank() method to Glyph type** in type1 package that returns true if the glyph has no drawing commands (LineTo or CurveTo operations)
- **New Outlines type** for separating outline data from Font struct, with methods for glyph bounding box calculations and blank glyph detection
- **Glyph name validation** with new IsValid() function in type1/names package
- **Dx() and Dy() methods** for funit.Rect16 type
- **SystemInfo type** in cid package with String() method for character collection identification

### Changed
- **Font struct refactored** to embed *Outlines instead of direct Glyphs, Private, and Encoding fields - methods NumGlyphs(), GlyphList(), and BuiltinEncoding() moved from Font to Outlines type
- **Glyph.BBox() method replaced** with Path() method that returns path.Path using updated geom/path package functionality
- **type1/names API updated** - ToUnicode and FromUnicode functions now use string values instead of runes
- **Updated dependency** seehuhn.de/go/geom to v0.6.0 with enhanced path functionality

### Fixed
- **Improved random key generation** in PostScript dictionary comparison function
- **Documentation typo** in CMap dictionary description

### Removed
- **Glyph.BBox() method** (replaced with Path() method for better integration with geom package)


## [v0.5.0] (2024-05-02)

### Added
- GitHub Actions workflow for automated CI/CD with build and test pipeline
- Contributing guidelines and documentation for project contributors

### Changed
- Updated Go runtime requirement from version 1.20 to 1.22.2
- Updated golang.org/x/exp dependency to latest version (2024-04-09)

### Security
- GitHub Actions workflow uses pinned action versions for enhanced security
