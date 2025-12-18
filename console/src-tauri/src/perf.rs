// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Performance monitoring commands for the Console dashboard.
//!
//! This module provides Tauri commands for collecting system metrics:
//! - CPU usage (via sysinfo crate)
//! - Memory usage (via sysinfo crate)
//! - GPU usage (via NVML on Windows/Linux, unavailable on macOS)

use std::sync::Mutex;
use sysinfo::{Pid, ProcessRefreshKind, System};

#[cfg(any(target_os = "windows", target_os = "linux"))]
use nvml_wrapper::Nvml;

/// Shared system state for CPU tracking.
/// CPU usage requires persistent state between calls to calculate deltas.
static SYSTEM: Mutex<Option<System>> = Mutex::new(None);

/// Shared NVML instance for GPU tracking (Windows/Linux only).
/// Initialized lazily on first use.
#[cfg(any(target_os = "windows", target_os = "linux"))]
static NVML_INSTANCE: Mutex<Option<Nvml>> = Mutex::new(None);

/// Returns the current process memory usage in bytes.
#[tauri::command]
pub fn get_memory_usage() -> Result<u64, String> {
    let pid = Pid::from_u32(std::process::id());
    let mut sys = System::new();
    sys.refresh_processes(sysinfo::ProcessesToUpdate::Some(&[pid]), true);

    if let Some(process) = sys.process(pid) {
        Ok(process.memory())
    } else {
        Err("Could not find current process".to_string())
    }
}

/// Returns the current process CPU usage as a percentage.
#[tauri::command]
pub fn get_cpu_usage() -> Result<f32, String> {
    let pid = Pid::from_u32(std::process::id());
    let mut guard = SYSTEM.lock().map_err(|e| e.to_string())?;

    // Initialize system if not already done
    let sys = guard.get_or_insert_with(System::new);

    // Refresh CPU info for our process
    sys.refresh_processes_specifics(
        sysinfo::ProcessesToUpdate::Some(&[pid]),
        true,
        ProcessRefreshKind::new().with_cpu(),
    );

    if let Some(process) = sys.process(pid) {
        Ok(process.cpu_usage())
    } else {
        Err("Could not find current process".to_string())
    }
}

/// Returns GPU utilization percentage for NVIDIA GPUs (Windows/Linux).
/// Returns None when no NVIDIA GPU is available or NVML fails to initialize.
#[cfg(any(target_os = "windows", target_os = "linux"))]
#[tauri::command]
pub fn get_gpu_usage() -> Option<f32> {
    let mut guard = NVML_INSTANCE.lock().ok()?;

    // Initialize NVML if not already done
    let nvml = guard.get_or_insert_with(|| Nvml::init().ok()).as_ref()?;

    // Get the first GPU device (device 0)
    let device = nvml.device_by_index(0).ok()?;
    let utilization = device.utilization_rates().ok()?;

    Some(utilization.gpu as f32)
}

/// Cached key index for GPU utilization (macOS).
/// Once we find which key works for this GPU, we remember it.
#[cfg(target_os = "macos")]
static GPU_KEY_INDEX: Mutex<Option<usize>> = Mutex::new(None);

/// Known GPU utilization key names (varies by GPU model).
#[cfg(target_os = "macos")]
const GPU_UTIL_KEYS: &[&str] = &[
    "Device Utilization %",
    "GPU Activity(%)",
    "GPU Core Utilization",
    "gpuUtilization",
];

/// Returns GPU utilization percentage on macOS via IOKit.
/// Queries IOAccelerator service for PerformanceStatistics.
/// Caches the working key name after first successful lookup.
#[cfg(target_os = "macos")]
#[tauri::command]
pub fn get_gpu_usage() -> Option<f32> {
    use core_foundation::base::{CFType, TCFType};
    use core_foundation::dictionary::CFDictionary;
    use core_foundation::number::CFNumber;
    use core_foundation::string::CFString;
    use io_kit_sys::types::io_iterator_t;
    use io_kit_sys::*;

    const KERN_SUCCESS: i32 = 0;

    // Helper to extract utilization from a CFNumber
    fn extract_utilization(value: &CFType) -> Option<f32> {
        let num_ref = value.as_CFTypeRef() as *const _;
        let cf_num = unsafe { CFNumber::wrap_under_get_rule(num_ref) };
        cf_num
            .to_f32()
            .or_else(|| cf_num.to_i64().map(|v| v as f32))
    }

    // Helper to try a specific key in the perf_dict
    fn try_key(perf_dict: &CFDictionary<CFString, CFType>, key_name: &str) -> Option<f32> {
        let util_key = CFString::new(key_name);
        perf_dict
            .find(&util_key)
            .and_then(|v| extract_utilization(&v))
    }

    unsafe {
        // Find IOAccelerator services (GPU drivers)
        let matching = IOServiceMatching(b"IOAccelerator\0".as_ptr() as *const i8);
        if matching.is_null() {
            return None;
        }

        let mut iterator: io_iterator_t = 0;
        #[allow(deprecated)]
        let result =
            IOServiceGetMatchingServices(kIOMasterPortDefault, matching, &mut iterator);
        if result != KERN_SUCCESS {
            return None;
        }

        let cached_index = GPU_KEY_INDEX.lock().ok().and_then(|g| *g);

        // Iterate through GPU services
        loop {
            let service = IOIteratorNext(iterator);
            if service == 0 {
                break;
            }

            // Get the properties dictionary
            let mut props: core_foundation::base::CFTypeRef = std::ptr::null();
            let result = IORegistryEntryCreateCFProperties(
                service,
                &mut props as *mut _ as *mut _,
                std::ptr::null(),
                0,
            );

            IOObjectRelease(service);

            if result != KERN_SUCCESS || props.is_null() {
                continue;
            }

            let dict: CFDictionary<CFString, CFType> =
                CFDictionary::wrap_under_create_rule(props as *const _ as *mut _);

            // Look for PerformanceStatistics
            let perf_key = CFString::new("PerformanceStatistics");
            if let Some(perf_stats) = dict.find(&perf_key) {
                let perf_dict: CFDictionary<CFString, CFType> =
                    CFDictionary::wrap_under_get_rule(perf_stats.as_CFTypeRef() as *mut _);

                // Fast path: use cached key if available
                if let Some(idx) = cached_index {
                    if let Some(utilization) = try_key(&perf_dict, GPU_UTIL_KEYS[idx]) {
                        IOObjectRelease(iterator);
                        return Some(utilization);
                    }
                }

                // Slow path: try all keys and cache the one that works
                for (idx, key_name) in GPU_UTIL_KEYS.iter().enumerate() {
                    if let Some(utilization) = try_key(&perf_dict, key_name) {
                        // Cache this key index for future calls
                        if let Ok(mut guard) = GPU_KEY_INDEX.lock() {
                            *guard = Some(idx);
                        }
                        IOObjectRelease(iterator);
                        return Some(utilization);
                    }
                }
            }
        }

        IOObjectRelease(iterator);
        None
    }
}
