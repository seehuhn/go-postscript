// seehuhn.de/go/postscript - a rudimentary PostScript interpreter
// Copyright (C) 2023  Jochen Voss <voss@seehuhn.de>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package psenc

var StandardEncoding = [256]string{
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	"space",
	"exclam",
	"quotedbl",
	"numbersign",
	"dollar",
	"percent",
	"ampersand",
	"quoteright",
	"parenleft",
	"parenright",
	"asterisk",
	"plus",
	"comma",
	"hyphen",
	"period",
	"slash",
	"zero",
	"one",
	"two",
	"three",
	"four",
	"five",
	"six",
	"seven",
	"eight",
	"nine",
	"colon",
	"semicolon",
	"less",
	"equal",
	"greater",
	"question",
	"at",
	"A",
	"B",
	"C",
	"D",
	"E",
	"F",
	"G",
	"H",
	"I",
	"J",
	"K",
	"L",
	"M",
	"N",
	"O",
	"P",
	"Q",
	"R",
	"S",
	"T",
	"U",
	"V",
	"W",
	"X",
	"Y",
	"Z",
	"bracketleft",
	"backslash",
	"bracketright",
	"asciicircum",
	"underscore",
	"quoteleft",
	"a",
	"b",
	"c",
	"d",
	"e",
	"f",
	"g",
	"h",
	"i",
	"j",
	"k",
	"l",
	"m",
	"n",
	"o",
	"p",
	"q",
	"r",
	"s",
	"t",
	"u",
	"v",
	"w",
	"x",
	"y",
	"z",
	"braceleft",
	"bar",
	"braceright",
	"asciitilde",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	"exclamdown",
	"cent",
	"sterling",
	"fraction",
	"yen",
	"florin",
	"section",
	"currency",
	"quotesingle",
	"quotedblleft",
	"guillemotleft",
	"guilsinglleft",
	"guilsinglright",
	"fi",
	"fl",
	".notdef",
	"endash",
	"dagger",
	"daggerdbl",
	"periodcentered",
	".notdef",
	"paragraph",
	"bullet",
	"quotesinglbase",
	"quotedblbase",
	"quotedblright",
	"guillemotright",
	"ellipsis",
	"perthousand",
	".notdef",
	"questiondown",
	".notdef",
	"grave",
	"acute",
	"circumflex",
	"tilde",
	"macron",
	"breve",
	"dotaccent",
	"dieresis",
	".notdef",
	"ring",
	"cedilla",
	".notdef",
	"hungarumlaut",
	"ogonek",
	"caron",
	"emdash",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	"AE",
	".notdef",
	"ordfeminine",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	"Lslash",
	"Oslash",
	"OE",
	"ordmasculine",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
	"ae",
	".notdef",
	".notdef",
	".notdef",
	"dotlessi",
	".notdef",
	".notdef",
	"lslash",
	"oslash",
	"oe",
	"germandbls",
	".notdef",
	".notdef",
	".notdef",
	".notdef",
}

// StandardEncodingRev is the standard encoding for Type 1 fonts.
var StandardEncodingRev = map[string]byte{
	"space":        32,
	"exclam":       33,
	"quotedbl":     34,
	"numbersign":   35,
	"dollar":       36,
	"percent":      37,
	"ampersand":    38,
	"quoteright":   39,
	"parenleft":    40,
	"parenright":   41,
	"asterisk":     42,
	"plus":         43,
	"comma":        44,
	"hyphen":       45,
	"period":       46,
	"slash":        47,
	"zero":         48,
	"one":          49,
	"two":          50,
	"three":        51,
	"four":         52,
	"five":         53,
	"six":          54,
	"seven":        55,
	"eight":        56,
	"nine":         57,
	"colon":        58,
	"semicolon":    59,
	"less":         60,
	"equal":        61,
	"greater":      62,
	"question":     63,
	"at":           64,
	"A":            65,
	"B":            66,
	"C":            67,
	"D":            68,
	"E":            69,
	"F":            70,
	"G":            71,
	"H":            72,
	"I":            73,
	"J":            74,
	"K":            75,
	"L":            76,
	"M":            77,
	"N":            78,
	"O":            79,
	"P":            80,
	"Q":            81,
	"R":            82,
	"S":            83,
	"T":            84,
	"U":            85,
	"V":            86,
	"W":            87,
	"X":            88,
	"Y":            89,
	"Z":            90,
	"bracketleft":  91,
	"backslash":    92,
	"bracketright": 93,
	"asciicircum":  94,
	"underscore":   95,
	"quoteleft":    96,
	"a":            97,
	"b":            98,
	"c":            99,
	"d":            100,
	"e":            101,
	"f":            102,
	"g":            103,
	"h":            104,
	"i":            105,
	"j":            106,
	"k":            107,
	"l":            108,
	"m":            109,
	"n":            110,
	"o":            111,
	"p":            112,
	"q":            113,
	"r":            114,
	"s":            115,
	"t":            116,
	"u":            117,
	"v":            118,
	"w":            119,
	"x":            120,
	"y":            121,
	"z":            122,
	"braceleft":    123,
	"bar":          124,
	"braceright":   125,
	"asciitilde":   126,

	"exclamdown":     161,
	"cent":           162,
	"sterling":       163,
	"fraction":       164,
	"yen":            165,
	"florin":         166,
	"section":        167,
	"currency":       168,
	"quotesingle":    169,
	"quotedblleft":   170,
	"guillemotleft":  171,
	"guilsinglleft":  172,
	"guilsinglright": 173,
	"fi":             174,
	"fl":             175,

	"endash":         177,
	"dagger":         178,
	"daggerdbl":      179,
	"periodcentered": 180,

	"paragraph":      182,
	"bullet":         183,
	"quotesinglbase": 184,
	"quotedblbase":   185,
	"quotedblright":  186,
	"guillemotright": 187,
	"ellipsis":       188,
	"perthousand":    189,

	"questiondown": 191,

	"grave":      193,
	"acute":      194,
	"circumflex": 195,
	"tilde":      196,
	"macron":     197,
	"breve":      198,
	"dotaccent":  199,
	"dieresis":   200,

	"ring":    202,
	"cedilla": 203,

	"hungarumlaut": 205,
	"ogonek":       206,
	"caron":        207,
	"emdash":       208,

	"AE": 225,

	"ordfeminine": 227,

	"Lslash":       232,
	"Oslash":       233,
	"OE":           234,
	"ordmasculine": 235,

	"ae": 241,

	"dotlessi": 245,

	"lslash":     248,
	"oslash":     249,
	"oe":         250,
	"germandbls": 251,
}
