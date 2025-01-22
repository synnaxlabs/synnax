cc_toolchain_suite(
    name = "gcc_suite",
    toolchains = {
        "k8": ":gcc_toolchain",
    },
)

cc_toolchain(
    name = "gcc_toolchain",
    all_files = ":empty",
    compiler_files = ":empty",
    dwp_files = ":empty",
    linker_files = ":empty",
    objcopy_files = ":empty",
    strip_files = ":empty",
    supports_param_files = 0,
    toolchain_config = ":gcc_config",
)

filegroup(name = "empty") 