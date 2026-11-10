package main

import "embed"

//go:embed skills/* extensions/* themes/* config/*
var EmbeddedAssets embed.FS
