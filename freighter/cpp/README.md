**Setting up the Dev Environment**

* **Bazel:** The build system. For more information on how Bazel's build system works, see: https://bazel.build/
  * Installation:
    * MacOS: https://bazel.build/install/os-x`
    * Windows: https://bazel.build/install/windows
    * Linux: https://bazel.build/install/u
  * Usage:
    * Building with Bazel is easy! Just `cd` into `synnax/freighter/cpp`, where the `WORKSPACE` file is found. From there, the library can be built entirely with `bazel build //src:freighter`
* **gRPC:** One of the communication protocols that freighter supports.
  * Bazel is gRPC's preferred build systems. That's one reason why we are using it, aside from being (subjectively) much easier to iterate using compared to cmake.
  * You will need to install `protoc` and `grpc` locally
    * On mac, you can run `brew install gprc`
