// Package seed: definisi content type bawaan (format JSON = API /content-types).
// Dibuat saat service start bila belum ada; yang sudah ada tidak ditimpa.
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

// Files: nama seed -> isi JSON. Tambah menu bawaan baru = tambah file + entry di sini & di Order.
var Files = map[string][]byte{"assets": assets, "master": master, "profile": profile}

// Order: master dulu (Domain, Site, Organization dipakai seed lain).
var Order = []string{"master", "profile", "assets"}
