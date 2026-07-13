package main

import "crypto/sha256"

func digest(value []byte) [32]byte { return sha256.Sum256(value) }
