// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

use tauri;
use serde;
use sled::{IVec,Db};
use home::{home_dir};

#[derive(serde::Deserialize)]
pub struct KVRequest {
    command: String,
    key: String,
    value: String,
}

#[derive(serde::Serialize)]
pub struct KVResponse {
    error: String,
    value: String
}

type DbState<'a> = tauri::State<'a, Db>;

#[tauri::command]
pub fn kv_exec(
    _window: tauri::Window,
    request: KVRequest,
    db: DbState,
) -> Result<KVResponse, String>{
    match request.command.as_str() {
        "set" => kv_set(db, request.key, request.value),
        "get"=> kv_get(db, request.key),
        "delete" => kv_delete(db, request.key),
        _ => Ok(KVResponse { error: "Invalid operation".to_string(), value: "".to_string()}),
    }
}


pub fn open() -> Result<Db, String> {
    let mut path = home_dir().unwrap();
    path.push(".synnax");
    path.push("delta");
    path.push("data");
    match sled::open(path) {
        Ok(db) => Ok(db),
        Err(e) => Err(e.to_string()),
    }
}

fn kv_set(db: DbState, key: String, value: String) -> Result<KVResponse, String> {
    let b_key = IVec::from(key.as_bytes());
    let b_value = IVec::from(value.as_bytes());
    match db.insert(b_key, b_value) {
        Ok(_v) => Ok(KVResponse { error: "".to_string(), value: "".to_string()}),
        Err(e) => Ok(KVResponse { error: e.to_string(), value: "".to_string()}),
    }
}

fn kv_get(db: DbState, key: String) -> Result<KVResponse, String> {
    let b_key = IVec::from(key.as_bytes());
    match db.get(b_key) {
        Ok(None) => Ok(KVResponse { error: "Key not found".to_string(), value: "".to_string()}),
        Ok(Some(v)) => Ok(KVResponse { error: "".to_string(), value: String::from_utf8(v.to_vec()).unwrap()}),
        Err(e) => Ok(KVResponse { error: e.to_string(), value: "".to_string()}),
    }
}

pub fn kv_delete(
    db:  DbState,
    key: String,
) -> Result<KVResponse, String> {
    let b_key = IVec::from(key.as_bytes());
    match db.remove(b_key) {
        Ok(_) => Ok(KVResponse { error: "".to_string(), value: "".to_string() }),
        Err(e) => Ok(KVResponse { error: e.to_string(), value: "".to_string() }),
    }
}
