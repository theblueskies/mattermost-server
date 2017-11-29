// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package model

import (
	"net/http"
	"unicode/utf8"
)

const (
	KEY_VALUE_KEY_MAX_RUNES = 128
)

type PluginKeyValue struct {
	Key   string `json:"key" db:"PKey"`
	Value []byte `json:"value" db:"PValue"`
}

func (kv *PluginKeyValue) IsValid() *AppError {
	if len(kv.Key) == 0 || utf8.RuneCountInString(kv.Key) > KEY_VALUE_KEY_MAX_RUNES {
		return NewAppError("PluginKeyValue.IsValid", "model.plugin_key_value.is_valid.key.app_error", map[string]interface{}{"Max": KEY_VALUE_KEY_MAX_RUNES, "Min": 0}, "key="+kv.Key, http.StatusBadRequest)
	}

	return nil
}
