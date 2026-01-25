# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
