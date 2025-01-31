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

# git_repository(
#     name = "com_github_gflags_gflags",
#     remote = "https://github.com/gflags/gflags.git",
#     tag = "v2.2.2"
# )

# git_repository(
#     name = "com_github_google_glog",
#     remote = "git@github.com:google/glog.git",
#     tag = "v0.6.0"
# )

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

# |||||||| THIS SECTION SHOULD BE COMMENTED IN FOR MACOS BUILDS ||||||||

# These two dependencies override the versions of the default deps loaded by
# rules_proto_grpc. For some reason symlinks fail on macOS with the default versions, so
# we need to override them.

#http_archive(
#    name = "com_google_protobuf",
#    sha256 = "a700a49470d301f1190a487a923b5095bf60f08f4ae4cac9f5f7c36883d17971",
#    strip_prefix = "protobuf-23.4",
#    urls = ["https://github.com/protocolbuffers/protobuf/archive/v23.4.tar.gz"],
#)
#
#http_archive(
#    name = "com_github_grpc_grpc",
#    sha256 = "8393767af531b2d0549a4c26cf8ba1f665b16c16fb6c9238a7755e45444881dd",
#    strip_prefix = "grpc-1.57.0",
#    urls = ["https://github.com/grpc/grpc/archive/v1.57.0.tar.gz"],
#)

# |||||||| END OF MACOS SECTION ||||||||

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
    path = "driver/vendor/labjack",
    build_file = "@//driver/vendor/labjack:BUILD.bazel",
)

# /////////////////////////////////////////////////////////////////////////////////////
# /////////////////////////////////////// NIDAQMX /////////////////////////////////////
# /////////////////////////////////////////////////////////////////////////////////////

# new_local_repository(
#     name = "nidaqmx",
#     path = "C:\\Program Files (x86)\\National Instruments\\Shared\\ExternalCompilerSupport\\C\\lib64\\msvc",
#     build_file="@//driver/vendor/nidaqmx:BUILD.bazel"
# )

# new_local_repository(
#     name = "nisyscfg",
#     path = "C:\\Program Files (x86)\\National Instruments\\Shared\\ExternalCompilerSupport\\C\\lib64\\msvc",
#     build_file="@//driver/vendor/nisyscfg:BUILD.bazel"
# )

git_repository(
    name = "nidaqmx",
    commit = "e85c66553cd9d0e1b07380a409729adfc71d3d98",
    remote = "https://github.com/synnaxlabs/vendor-libraries",
)

git_repository(
    name = "nisyscfg",
    commit = "e85c66553cd9d0e1b07380a409729adfc71d3d98",
    remote = "https://github.com/synnaxlabs/vendor-libraries",
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
