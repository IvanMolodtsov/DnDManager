package dndmanager

import "embed"

// AssetFS is web/, locales/, and migrations/ compiled into the binary.
// @vercel/go includeFiles globs from api/ and cannot use ../; embed is packed at build.
//
//go:embed all:web all:locales all:migrations
var AssetFS embed.FS
