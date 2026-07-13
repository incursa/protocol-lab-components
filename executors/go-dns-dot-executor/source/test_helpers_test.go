package main

import "encoding/hex"

func fmtHex(value []byte) string { return hex.EncodeToString(value) }
