// This is the client library for gRPC freighter.
// It contains the standard freighter interface.
#include "freighter/gRPC/client.h"

// Our generated proto file that we wish to send. 
#include "src/protos/message_service.grpc.pb.h"

// std libraries.
#include <string>
#include <iostream>

// To make code cleaner, we always want to use type aliuses. 
// The gRPC client object takes four template types: 
// response_t, request_t, stream_t, err_t, and rpc_t.

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