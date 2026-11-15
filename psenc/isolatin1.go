// Copyright 2024 Jochen Voss <voss@seehuhn.de>
// SPDX-License-Identifier: BSD-3-Clause

package psenc

// ISOLatin1Encoding is the PostScript ISOLatin1Encoding encoding vector.
// Codes without a character are mapped to ".notdef".
var ISOLatin1Encoding = [256]string{
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
	"space",        // 32
	"exclam",       // 33
	"quotedbl",     // 34
	"numbersign",   // 35
	"dollar",       // 36
	"percent",      // 37
	"ampersand",    // 38
	"quoteright",   // 39
	"parenleft",    // 40
	"parenright",   // 41
	"asterisk",     // 42
	"plus",         // 43
	"comma",        // 44
	"minus",        // 45
	"period",       // 46
	"slash",        // 47
	"zero",         // 48
	"one",          // 49
	"two",          // 50
	"three",        // 51
	"four",         // 52
	"five",         // 53
	"six",          // 54
	"seven",        // 55
	"eight",        // 56
	"nine",         // 57
	"colon",        // 58
	"semicolon",    // 59
	"less",         // 60
	"equal",        // 61
	"greater",      // 62
	"question",     // 63
	"at",           // 64
	"A",            // 65
	"B",            // 66
	"C",            // 67
	"D",            // 68
	"E",            // 69
	"F",            // 70
	"G",            // 71
	"H",            // 72
	"I",            // 73
	"J",            // 74
	"K",            // 75
	"L",            // 76
	"M",            // 77
	"N",            // 78
	"O",            // 79
	"P",            // 80
	"Q",            // 81
	"R",            // 82
	"S",            // 83
	"T",            // 84
	"U",            // 85
	"V",            // 86
	"W",            // 87
	"X",            // 88
	"Y",            // 89
	"Z",            // 90
	"bracketleft",  // 91
	"backslash",    // 92
	"bracketright", // 93
	"asciicircum",  // 94
	"underscore",   // 95
	"quoteleft",    // 96
	"a",            // 97
	"b",            // 98
	"c",            // 99
	"d",            // 100
	"e",            // 101
	"f",            // 102
	"g",            // 103
	"h",            // 104
	"i",            // 105
	"j",            // 106
	"k",            // 107
	"l",            // 108
	"m",            // 109
	"n",            // 110
	"o",            // 111
	"p",            // 112
	"q",            // 113
	"r",            // 114
	"s",            // 115
	"t",            // 116
	"u",            // 117
	"v",            // 118
	"w",            // 119
	"x",            // 120
	"y",            // 121
	"z",            // 122
	"braceleft",    // 123
	"bar",          // 124
	"braceright",   // 125
	"asciitilde",   // 126
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
	"dotlessi",   // 144
	"grave",      // 145
	"acute",      // 146
	"circumflex", // 147
	"tilde",      // 148
	"macron",     // 149
	"breve",      // 150
	"dotaccent",  // 151
	"dieresis",   // 152
	".notdef",
	"ring",    // 154
	"cedilla", // 155
	".notdef",
	"hungarumlaut",   // 157
	"ogonek",         // 158
	"caron",          // 159
	"space",          // 160
	"exclamdown",     // 161
	"cent",           // 162
	"sterling",       // 163
	"currency",       // 164
	"yen",            // 165
	"brokenbar",      // 166
	"section",        // 167
	"dieresis",       // 168
	"copyright",      // 169
	"ordfeminine",    // 170
	"guillemotleft",  // 171
	"logicalnot",     // 172
	"hyphen",         // 173
	"registered",     // 174
	"macron",         // 175
	"degree",         // 176
	"plusminus",      // 177
	"twosuperior",    // 178
	"threesuperior",  // 179
	"acute",          // 180
	"mu",             // 181
	"paragraph",      // 182
	"periodcentered", // 183
	"cedilla",        // 184
	"onesuperior",    // 185
	"ordmasculine",   // 186
	"guillemotright", // 187
	"onequarter",     // 188
	"onehalf",        // 189
	"threequarters",  // 190
	"questiondown",   // 191
	"Agrave",         // 192
	"Aacute",         // 193
	"Acircumflex",    // 194
	"Atilde",         // 195
	"Adieresis",      // 196
	"Aring",          // 197
	"AE",             // 198
	"Ccedilla",       // 199
	"Egrave",         // 200
	"Eacute",         // 201
	"Ecircumflex",    // 202
	"Edieresis",      // 203
	"Igrave",         // 204
	"Iacute",         // 205
	"Icircumflex",    // 206
	"Idieresis",      // 207
	"Eth",            // 208
	"Ntilde",         // 209
	"Ograve",         // 210
	"Oacute",         // 211
	"Ocircumflex",    // 212
	"Otilde",         // 213
	"Odieresis",      // 214
	"multiply",       // 215
	"Oslash",         // 216
	"Ugrave",         // 217
	"Uacute",         // 218
	"Ucircumflex",    // 219
	"Udieresis",      // 220
	"Yacute",         // 221
	"Thorn",          // 222
	"germandbls",     // 223
	"agrave",         // 224
	"aacute",         // 225
	"acircumflex",    // 226
	"atilde",         // 227
	"adieresis",      // 228
	"aring",          // 229
	"ae",             // 230
	"ccedilla",       // 231
	"egrave",         // 232
	"eacute",         // 233
	"ecircumflex",    // 234
	"edieresis",      // 235
	"igrave",         // 236
	"iacute",         // 237
	"icircumflex",    // 238
	"idieresis",      // 239
	"eth",            // 240
	"ntilde",         // 241
	"ograve",         // 242
	"oacute",         // 243
	"ocircumflex",    // 244
	"otilde",         // 245
	"odieresis",      // 246
	"divide",         // 247
	"oslash",         // 248
	"ugrave",         // 249
	"uacute",         // 250
	"ucircumflex",    // 251
	"udieresis",      // 252
	"yacute",         // 253
	"thorn",          // 254
	"ydieresis",      // 255
}
