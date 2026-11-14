// Package diag holds source positions and diagnostics (errors, warnings,
// hints) and formats them for the terminal:
//
//	error: cannot add Int and Float
//	  --> core.brz:14:12
//	   |
//	14 |     return a + 2.0
//	   |            ^^^^^^^ Int + Float
//	   |
//	   = help: convert explicitly: Float(a) + 2.0
package diag
