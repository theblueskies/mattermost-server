// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package sqlstore

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/store"
)

type SqlPluginStore struct {
	SqlStore
}

func NewSqlPluginStore(sqlStore SqlStore) store.PluginStore {
	s := &SqlPluginStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.PluginKeyValue{}, "PluginKeyValueStore").SetKeys(false, "Key")
		table.ColMap("Key").SetMaxSize(128)
		table.ColMap("Value").SetMaxSize(8192)
	}

	return s
}

func (ps SqlPluginStore) CreateIndexesIfNotExists() {
}

func (ps SqlPluginStore) SaveOrUpdate(kv *model.PluginKeyValue) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		if result.Err = kv.IsValid(); result.Err != nil {
			return
		}

		if ps.DriverName() == model.DATABASE_DRIVER_POSTGRES {
			// Unfortunately PostgreSQL pre-9.5 does not have an atomic upsert, so we use
			// separate update and insert queries to accomplish our upsert
			if rowsAffected, err := ps.GetMaster().Update(kv); err != nil {
				result.Err = model.NewAppError("SqlPluginStore.SaveOrUpdate", "store.sql_plugin_store.save.app_error", nil, err.Error(), http.StatusInternalServerError)
				return
			} else if rowsAffected == 0 {
				// No rows were affected by the update, so let's try an insert
				if err := ps.GetMaster().Insert(kv); err != nil {
					// If the error is from unique constraints violation, it's the result of a
					// valid race and we can report success. Otherwise we have a real error and
					// need to return it
					if !IsUniqueConstraintError(err, []string{"PRIMARY", "Key", "PKey"}) {
						result.Err = model.NewAppError("SqlPluginStore.SaveOrUpdate", "store.sql_plugin_store.save.app_error", nil, err.Error(), http.StatusInternalServerError)
						return
					}
				}
			}
		} else if ps.DriverName() == model.DATABASE_DRIVER_MYSQL {
			if _, err := ps.GetMaster().Exec("INSERT INTO PluginKeyValueStore (PKey, PValue) VALUES(:Key, :Value) ON DUPLICATE KEY UPDATE PValue = :Value", map[string]interface{}{"Key": kv.Key, "Value": kv.Value}); err != nil {
				result.Err = model.NewAppError("SqlPluginStore.SaveOrUpdate", "store.sql_plugin_store.save.app_error", nil, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		result.Data = kv
	})
}

func (ps SqlPluginStore) Get(key string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		var kv *model.PluginKeyValue

		if err := ps.GetReplica().SelectOne(&kv, "SELECT * FROM PluginKeyValueStore WHERE PKey = :Key", map[string]interface{}{"Key": key}); err != nil {
			if err == sql.ErrNoRows {
				result.Err = model.NewAppError("SqlPluginStore.Get", "store.sql_plugin_store.get.app_error", nil, fmt.Sprintf("key=%v, err=%v", key, err.Error()), http.StatusNotFound)
			} else {
				result.Err = model.NewAppError("SqlPluginStore.Get", "store.sql_plugin_store.get.app_error", nil, fmt.Sprintf("key=%v, err=%v", key, err.Error()), http.StatusInternalServerError)
			}
		} else {
			result.Data = kv
		}
	})
}

func (ps SqlPluginStore) Delete(key string) store.StoreChannel {
	return store.Do(func(result *store.StoreResult) {
		if _, err := ps.GetMaster().Exec("DELETE FROM PluginKeyValueStore WHERE PKey = :Key", map[string]interface{}{"Key": key}); err != nil {
			result.Err = model.NewAppError("SqlPluginStore.Delete", "store.sql_plugin_store.delete.app_error", nil, fmt.Sprintf("key=%v, err=%v", key, err.Error()), http.StatusInternalServerError)
		} else {
			result.Data = true
		}
	})
}
