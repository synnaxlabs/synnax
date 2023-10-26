**Freighter gRPC/C++**

*Quick Start*

This directory has a minimal client application built with Freighter gRPC in C++. To start, first install
Bazel: https://bazel.build/

+ WORKSPACE file
  + The workspace file is used to load all dependencies for Freighter into your computer without having to explicitly download these libraries locally.
```WORKSPACE
load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

# Freighter.
http_archive(
    name = "Freighter", 
    urls = ["https://github.com/synnaxlabs/synnax/raw/sy-182-streaming-interface-cpp/freighter/cpp/releases/freighter-1.0.tar.gz"],
    strip_prefix = "cpp"
)

# gRPC.
http_archive(
    name = "rules_proto_grpc",
    sha256 = "9ba7299c5eb6ec45b6b9a0ceb9916d0ab96789ac8218269322f0124c0c0d24e2",
    strip_prefix = "rules_proto_grpc-4.5.0",
    urls = ["https://github.com/rules-proto-grpc/rules_proto_grpc/releases/download/4.5.0/rules_proto_grpc-4.5.0.tar.gz"],
)

load("@rules_proto_grpc//:repositories.bzl", "rules_proto_grpc_toolchains", "rules_proto_grpc_repos")
rules_proto_grpc_toolchains()
rules_proto_grpc_repos()

load("@rules_proto//proto:repositories.bzl", "rules_proto_dependencies", "rules_proto_toolchains")
rules_proto_dependencies()
rules_proto_toolchains()

load ("@rules_proto_grpc//cpp:repositories.bzl", "cpp_repos")

rules_proto_grpc_cpp_repos()

load("@com_github_grpc_grpc//bazel:grpc_deps.bzl", "grpc_deps")

grpc_deps()

load("@com_github_grpc_grpc//bazel:grpc_extra_deps.bzl", "grpc_extra_deps")

grpc_extra_deps()
```
+ BUILD.bazel
 + Create a file for all of your source files to live in — in this case, we've named ours `src/`. In it, include a `BUILD.bazel` file.
 + This file will manage all of the compilation and linking of your cpp project.

```src/BUILD.bzl

cc_binary(
    name = "basic_stream",
    srcs = ["basic_stream.cpp"],
    deps = ["@Freighter//freighter/gRPC:gRPC", "//src/protos:message_service"]
)
```
In our example, we are compiling one main file, `basic_stream.cpp`. You can see we've included `"@Freighter//freighter/gRPC:gRPC"`, which includes all of the necessary Freighter gRPC deps. We also include `"//src/protos:message_service"`, which we will see in a moment is a rule for us to build the message service proto.

Create a directory in `src/` to hold your protos. In this example, we've named it `src/protos'. Add a `BUILD.bazel` file to compile the `.proto` files:

```src/protos/BUILD.bazel
proto_library(
    name = "message_service_proto",
    srcs = ["message_service.proto"],
)

load("@rules_proto_grpc//cpp:defs.bzl", "cpp_grpc_library")

cpp_grpc_library(
    name = "message_service",
    protos = [":message_service_proto"],
    visibility = ["//visibility:public"]
)
```

Let's look at the actual `.proto` file. 

```src/protos/message_service.proto

syntax = "proto3";

package masa;

service Communication
{
    rpc Unary(Data) returns (Data) {}

    rpc Stream(stream Data) returns (stream Data) {}
}

message Data
{
    string name = 1;
    repeated int32 values = 2;
}
```

Here, we define a package to uniquely identify this file, in this case `masa`. We create a service `Communication` which defines a unary and bidrectional rpc, `Unary` and `Stream`. **THESE TWO RPCS MUST BE INCLUDED FOR FREIGHTER TO COMPILE**. We also define a message `Data`, which defines the data to be send and received between the client and the server.

Note that the message does not need to conform to the above schema; it can be named and can contain any data that protobuf allows. 

*Finally*, let's jump into the code!

We first start with best practices: type aliusing.

```cpp
// response_t: The proto compiled response type.
using response_t = masa::Data;

// request_t: The proto compiled request type.
using request_t = masa::Data;

// err_t: In this case, grpc::Status. DO NOT use another type.
using err_t = grpc::Status;

// rpc_t: the service defined in our proto file.
using rpc_t = masa::Communication;

// stream_t: a gRPCStreamer of type gRPCStreamer<response_t, request_t, err_t, rpc_t>
using stream_t = gRPCStreamer<response_t, request_t, err_t, rpc_t>;
```

The `gRPC` object takes a lot of templates, and to make it clear what is what without muddying up the code, it is preferred to type alius all of your templates before instantiating the `client object`.

**The Meat**

Below is pretty much all of the lines of code needed to send and receive data.
```cpp
int main()
{
    // We start by creating a client object with our templates.
    auto client = gRPC<response_t, request_t, stream_t, err_t, rpc_t>();

    // We then choose the target that we want to send to.
    std::string target("localhost:8080");

    // We then create a streamer object using stream.
    auto streamer = client.stream(target);

    // To send a payload, we construct the proto defined Data object,
    // set a payload, and send.
    // The return will be a grpc status, which we can check to 
    // see if the message was sent successfully.
    auto payload = masa::Data();
    payload.set_name("Hey there!");
    payload.mutable_values()->Add(3);

    std::cout << "Sending data: " << payload.name() << std::endl;
    auto sent_status = streamer.send(payload);

    if (!sent_status.ok())
    {
        std::cout << "Error: unable to send message. Terminating program..." << std::endl;
        exit(sent_status.error_code()); 
    }

    // Let's receive a message from the server!
    // To do this, we can simply call receive(). This
    // returns a pair of 
    auto [response, receive_status] = streamer.receive();

    if (!receive_status.ok())
    {
        std::cout << "Error: unable to receive message. Terminating program..." << std::endl;
        exit(receive_status.error_code()); 
    }

    std::cout << "Received message: " << response.name() << std::endl;

    // If we don't want to send any more messages, we can call closeSend()
    streamer.closeSend();

    return 0;
}
```

As long as we call `closeSend()` after instantiating a `streamer`, we can pretty much do as we please. We can even create multiple streamers to different targets, and send and receive messages as we please. 

**NOTE**: To prevent undefined behavior, it is recommended to check the status before continuing sending and receiving payloads.

**Building the Example**

From `synnax/client/cpp/gRPC`, run `bazel build //src:basic_stream`. 

Note that this doesn't come with an implemented server, so running `bazel run //src:basic_stream` will cause an error since there is no server to connect to. But if you do want to build out the server side for this, go ahead :D working examples of server client interaction can be found in `synnax/freighter/cpp/freighter/gRPC/test`.





