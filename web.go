package remotedesktop

import "embed"

//go:embed web/index.html web/static/*
var WebFS embed.FS
