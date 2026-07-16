package ghostcaptcha

import "strings"

// lineEndingReplacer normalizes CRLF/CR to LF and expands tabs to spaces,
// since fonts have no glyph for raw control characters.
var lineEndingReplacer = strings.NewReplacer("\r\n", "\n", "\r", "\n", "\t", "    ")
