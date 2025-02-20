load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// GLOG ////////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

http_archive(
    name = "com_github_gflags_gflags",
    sha256 = "34af2f15cf7367513b352bdcd2493ab14ce43692d2dcd9dfc499492966c64dcf",
    strip_prefix = "gflags-2.2.2",
    urls = ["https://github.com/gflags/gflags/archive/v2.2.2.tar.gz"],
)

http_archive(
    name = "com_github_google_glog",
    sha256 = "122fb6b712808ef43fbf80f75c52a21c9760683dae470154f02bddfc61135022",
    strip_prefix = "glog-0.6.0",
    urls = ["https://github.com/google/glog/archive/v0.6.0.zip"],
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// GTEST ////////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

http_archive(
    name = "com_google_googletest",
    strip_prefix = "googletest-5ab508a01f9eb089207ee87fd547d290da39d015",
    urls = ["https://github.com/google/googletest/archive/5ab508a01f9eb089207ee87fd547d290da39d015.zip"],
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// GRPC ////////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

http_archive(
    name = "rules_proto_grpc",
    sha256 = "9ba7299c5eb6ec45b6b9a0ceb9916d0ab96789ac8218269322f0124c0c0d24e2",
    strip_prefix = "rules_proto_grpc-4.5.0",
    urls = ["https://github.com/rules-proto-grpc/rules_proto_grpc/releases/download/4.5.0/rules_proto_grpc-4.5.0.tar.gz"],
)

load("@rules_proto_grpc//:repositories.bzl", "rules_proto_grpc_repos", "rules_proto_grpc_toolchains")

rules_proto_grpc_toolchains()

rules_proto_grpc_repos()

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")

rules_proto_dependencies()

rules_proto_toolchains()

load("@rules_proto_grpc//cpp:repositories.bzl", "cpp_repos")

cpp_repos()

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")

grpc_deps()

load("@com_github_grpc_grpc//bazel:grpc_extra_deps.bzl", "grpc_extra_deps")

grpc_extra_deps()

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// NLOHMANN JSON ///////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

git_repository(
    name = "nlohmann_json",
    commit = "9cca280a4d0ccf0c08f47a99aa71d1b0e52f8d03",
    remote = "https://github.com/nlohmann/json",
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// LABJACK /////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

new_local_repository(
    name = "labjack",
    build_file = "@//driver/vendor/labjack:BUILD.bazel",
    path = "driver/vendor/labjack",
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// OPEN2541 /////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

new_local_repository(
    name = "open62541",
    build_file = "@//driver/vendor/open62541:BUILD.bazel",
    path = "./driver/vendor/open62541/open62541/out",
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// OPENSSL /////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

new_local_repository(
    name = "mbedtls_win",
    build_file = "@//driver/vendor/mbedtls:BUILD.bazel",
    path = "./driver/vendor/mbedtls/mbedtls-install",
)

new_local_repository(
    name = "mbedtls_macos",
    build_file = "@//driver/vendor/mbedtls:BUILD.bazel",
    path = "/opt/homebrew/Cellar/mbedtls/3.6.2",
)

new_local_repository(
    name = "mbedtls_linux",
    build_file = "@//driver/vendor/mbedtls:BUILD.bazel",
    path = "/usr/lib/x86_64-linux-gnu/",
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// SKYLIB ///////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

http_archive(
    name = "bazel_skylib",
    strip_prefix = "bazel-skylib-master",
    urls = ["https://github.com/bazelbuild/bazel-skylib/archive/master.zip"],
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// LUA ///////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

http_archive(
    name = "lua",
    build_file = "@//driver/vendor/lua:BUILD.bazel",
    sha256 = "7d5ea1b9cb6aa0b59ca3dde1c6adcb57ef83a1ba8e5432c0ecd06bf439b3ad88",
    strip_prefix = "lua-5.4.6",
    urls = ["https://www.lua.org/ftp/lua-5.4.6.tar.gz"],  # Replace with the latest version
)

git_repository(
    name = "bazel_clang_tidy",
    commit = "bff5c59c843221b05ef0e37cef089ecc9d24e7da",
    remote = "https://github.com/erenon/bazel_clang_tidy.git",
)

http_archive(
    name = "hedron_compile_commands",
    url = "https://github.com/hedronvision/bazel-compile-commands-extractor/archive/ed994039a951b736091776d677f324b3903ef939.tar.gz",
    strip_prefix = "bazel-compile-commands-extractor-ed994039a951b736091776d677f324b3903ef939",
    sha256 = "085bde6c5212c8c1603595341ffe7133108034808d8c819f8978b2b303afc9e7",
)
load("@hedron_compile_commands//:workspace_setup.bzl", "hedron_compile_commands_setup")
hedron_compile_commands_setup()