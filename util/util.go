// Copyright 2016 Sam Dukhovni <dukhovni@mit.edu>
//
// Licensed under the MIT License
// (https://opensource.org/licenses/MIT)
//
// util contains miscellaneous functions useful for clyde-go.

package util

import (
	"github.com/zephyr-im/zephyr-go"
)

func MessageZSig(r zephyr.MessageReaderResult) string {
	zsig := ""
	fields := len(r.Message.Body)
	if fields > 1 {
		zsig = r.Message.Body[fields-2]
	}
	return zsig
}

func MessageBody(r zephyr.MessageReaderResult) string {
	body := ""
	fields := len(r.Message.Body)
	if fields > 0 {
		body = r.Message.Body[fields-1]
	}
	return body
}
