/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and reactivejob-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
)

// Converts a string to int
func mustAtoi(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		panic(err)
	}
	return i
}

// Converts a json object to json.
func mustToJson(obj interface{}) json.RawMessage {
	raw, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}
	return raw
}

// Generates a hash value of a json object.
func jsonHash(obj interface{}) string {
	sum := sha256.Sum256(mustToJson(obj))
	return hex.EncodeToString(sum[:])
}
