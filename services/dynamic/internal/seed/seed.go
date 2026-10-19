// Package seed: definisi content type bawaan (format JSON = API /content-types).
// Created at service start when missing; existing ones are never overwritten.
package seed

import _ "embed"

var (
	//go:embed assets.json
	assets []byte
	//go:embed master.json
	master []byte
	//go:embed profile.json
	profile []byte
)

// Files: seed name -> JSON content. New built-in menu = add a file + an entry here and in Order.
var Files = map[string][]byte{"assets": assets, "master": master, "profile": profile}

// Order: master dulu (Domain, Site, Organization dipakai seed lain).
var Order = []string{"master", "profile", "assets"}
