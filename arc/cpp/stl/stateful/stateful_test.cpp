// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <memory>
#include <string>
#include <vector>

#include "gtest/gtest.h"
#include "wasmtime.hh"

#include "arc/cpp/stl/stateful/state.h"
#include "arc/cpp/stl/stateful/stateful.h"

namespace arc::stl::stateful {

const std::string_view BOOL_WAT = R"wat(
(module
  (import "stateful" "load_bool" (func $load_bool (param i32 i32) (result i32)))
  (import "stateful" "store_bool" (func $store_bool (param i32 i32)))
  (import "stateful" "load_series_bool" (func $load_series_bool (param i32 i32) (result i32)))
  (import "stateful" "store_series_bool" (func $store_series_bool (param i32 i32)))

  (func (export "load_bool") (param i32 i32) (result i32)
    (call $load_bool (local.get 0) (local.get 1)))
  (func (export "store_bool") (param i32 i32)
    (call $store_bool (local.get 0) (local.get 1)))
  (func (export "load_series_bool") (param i32 i32) (result i32)
    (call $load_series_bool (local.get 0) (local.get 1)))
  (func (export "store_series_bool") (param i32 i32)
    (call $store_series_bool (local.get 0) (local.get 1)))
)
)wat";

struct Fixture {
    std::shared_ptr<Variables> vars;
    std::shared_ptr<series::State> series_state;
    std::shared_ptr<strings::State> str_state;
    Module mod;
    wasmtime::Engine engine;
    wasmtime::Store store;
    wasmtime::Linker linker;
    wasmtime::Instance instance;

    Fixture(const std::string &wat):
        vars(std::make_shared<Variables>()),
        series_state(std::make_shared<series::State>()),
        str_state(std::make_shared<strings::State>()),
        mod(vars, series_state, str_state),
        store(engine),
        linker(engine),
        instance(setup(wat)) {}

    wasmtime::Func get(const std::string &name) {
        return std::get<wasmtime::Func>(*instance.get(store, name));
    }

    int32_t call2(const std::string &name, int32_t a, int32_t b) {
        return this->get(name)
            .call(this->store, {wasmtime::Val(a), wasmtime::Val(b)})
            .unwrap()[0]
            .i32();
    }

    void call2_void(const std::string &name, int32_t a, int32_t b) {
        (void) this->get(name)
            .call(this->store, {wasmtime::Val(a), wasmtime::Val(b)})
            .unwrap();
    }

private:
    wasmtime::Instance setup(const std::string &wat) {
        mod.bind_to(linker, store);
        auto wasm_mod = wasmtime::Module::compile(engine, wat).unwrap();
        return linker.instantiate(store, wasm_mod).unwrap();
    }
};

x::telem::Series make_bool_series(const std::vector<uint8_t> &values) {
    return x::telem::Series(values, x::telem::BOOL_T);
}

TEST(StatefulModule, LoadBoolSeedsInitialValueAndPersistsStores) {
    Fixture f{std::string(BOOL_WAT)};
    f.vars->set_current_node_key("node1");
    EXPECT_EQ(f.call2("load_bool", 0, 1), 1);
    EXPECT_EQ(f.call2("load_bool", 0, 0), 1);
    f.call2_void("store_bool", 0, 0);
    EXPECT_EQ(f.call2("load_bool", 0, 1), 0);
}

TEST(StatefulModule, StoreBoolNormalizesNonzero) {
    Fixture f{std::string(BOOL_WAT)};
    f.vars->set_current_node_key("node1");
    f.call2_void("store_bool", 0, 42);
    EXPECT_EQ(f.call2("load_bool", 0, 0), 1);
}

TEST(StatefulModule, LoadAndStoreBoolSeriesViaHandles) {
    Fixture f{std::string(BOOL_WAT)};
    f.vars->set_current_node_key("node1");
    const auto init_h = static_cast<int32_t>(
        f.series_state->store(make_bool_series({1, 0, 1}))
    );
    const auto rh = f.call2("load_series_bool", 0, init_h);
    EXPECT_EQ(rh, init_h);
    ASSERT_NE(f.series_state->get(rh), nullptr);
    EXPECT_EQ(f.series_state->get(rh)->size(), 3);
    const auto new_h = static_cast<int32_t>(
        f.series_state->store(make_bool_series({0, 1}))
    );
    f.call2_void("store_series_bool", 0, new_h);
    const auto rh2 = f.call2("load_series_bool", 0, 0);
    const auto *ser2 = f.series_state->get(rh2);
    ASSERT_NE(ser2, nullptr);
    EXPECT_EQ(ser2->size(), 2);
    EXPECT_EQ(ser2->at<uint8_t>(0), 0);
}

TEST(StatefulModule, ClearNodeResetsBoolState) {
    Fixture f{std::string(BOOL_WAT)};
    f.vars->set_current_node_key("node1");
    f.call2_void("store_bool", 0, 1);
    f.vars->clear_node("node1");
    EXPECT_EQ(f.call2("load_bool", 0, 0), 0);
}
}
