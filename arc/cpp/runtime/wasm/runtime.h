// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <cstdint>
#include <span>
#include <string>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "wasm_export.h"

namespace arc {

/// @brief WASM value type for function arguments and results.
struct WasmValue {
    enum class Kind { I32, I64, F32, F64 };

    Kind kind;
    union {
        int32_t i32;
        int64_t i64;
        float f32;
        double f64;
    };

    WasmValue(): kind(Kind::I32), i32(0) {}
    explicit WasmValue(int32_t v): kind(Kind::I32), i32(v) {}
    explicit WasmValue(int64_t v): kind(Kind::I64), i64(v) {}
    explicit WasmValue(float v): kind(Kind::F32), f32(v) {}
    explicit WasmValue(double v): kind(Kind::F64), f64(v) {}
};

/// @brief RAII wrapper for WAMR (WebAssembly Micro Runtime) in AOT mode.
///
/// Provides a C++ interface to load, instantiate, and execute AOT-compiled
/// WebAssembly modules. Designed for real-time constraints:
/// - AOT compilation eliminates JIT non-determinism
/// - Pre-allocated argument/result buffers
/// - Fixed memory size (no growth)
/// - RAII ensures proper cleanup
///
/// Usage:
///   Runtime rt;
///   rt.load_aot_module(aot_bytes);
///   rt.instantiate(64 * 1024, 0, host_context);
///   auto [func, err] = rt.find_function("main");
///   rt.call_function(func, args, results);
class Runtime {
    wasm_module_t module_ = nullptr;
    wasm_module_inst_t module_inst_ = nullptr;
    wasm_exec_env_t exec_env_ = nullptr;

    // Pre-allocated buffers for function calls (avoid per-call allocation)
    static constexpr size_t MAX_ARGS = 16;
    static constexpr size_t MAX_RESULTS = 16;
    std::array<uint32_t, MAX_ARGS> arg_buffer_;
    std::array<uint32_t, MAX_RESULTS>
        result_buffer_; // TODO: Use for multi-value results

    bool initialized_ = false;

public:
    Runtime() = default;
    ~Runtime();

    // Move-only (runtime is tied to WAMR resources)
    Runtime(Runtime &&other) noexcept;
    Runtime &operator=(Runtime &&other) noexcept;
    Runtime(const Runtime &) = delete;
    Runtime &operator=(const Runtime &) = delete;

    /// @brief Initialize WAMR runtime system (must be called once globally).
    /// @return Error status (NIL on success).
    /// @note Call this before creating any Runtime instances.
    static xerrors::Error initialize_runtime();

    /// @brief Destroy WAMR runtime system (must be called at shutdown).
    /// @note Call this after all Runtime instances are destroyed.
    static void destroy_runtime();

    /// @brief Load an AOT-compiled WASM module.
    /// @param aot_bytes AOT bytecode (compiled with wamrc).
    /// @return Error status (NIL on success).
    /// @note Must be called during initialization, not in RT loop.
    xerrors::Error load_aot_module(const std::vector<uint8_t> &aot_bytes);

    /// @brief Instantiate the module with fixed memory.
    /// @param stack_size Stack size in bytes (e.g., 64 * 1024).
    /// @param heap_size Heap size in bytes (0 to disable WASM heap).
    /// @return Error status (NIL on success).
    /// @note Must be called after load_aot_module(), during initialization.
    ///       User data (NodeState) is set later via set_user_data().
    xerrors::Error instantiate(uint32_t stack_size, uint32_t heap_size);

    /// @brief Set user data for host functions.
    /// @param user_data Pointer passed to host functions (typically NodeState*).
    /// @note Call this before executing WASM functions.
    void set_user_data(void *user_data);

    /// @brief Find an exported function by name.
    /// @param name Function name (e.g., "main", "add").
    /// @return Function instance and error status.
    /// @note Must be called during initialization to cache function pointers.
    std::pair<wasm_function_inst_t, xerrors::Error>
    find_function(const std::string &name);

    /// @brief Call a WASM function (RT-safe if AOT-compiled).
    /// @param func Function instance (from find_function).
    /// @param args Function arguments.
    /// @param results Output buffer for results.
    /// @return Error status (NIL on success).
    /// @note RT-safe: AOT execution is deterministic, uses pre-allocated buffers.
    xerrors::Error call_function(
        wasm_function_inst_t func,
        std::span<const WasmValue> args,
        std::span<WasmValue> results
    );

    /// @brief Get execution environment (for host function context).
    /// @return Execution environment pointer.
    wasm_exec_env_t exec_env() const { return exec_env_; }

    /// @brief Check if runtime is initialized and ready.
    /// @return true if module is loaded and instantiated.
    bool is_ready() const {
        return module_ != nullptr && module_inst_ != nullptr && exec_env_ != nullptr;
    }
};

} // namespace arc
