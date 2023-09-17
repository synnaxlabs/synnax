#pragma once

/// Abstract class.
#include "src/freighter.h"

/// std.
#include <memory>

/// @brief gRPC specific class
/// NOTE: stub_t comes from the generated protobuf file.
template <typename response_t, typename request_t, typename stream_t, typename stub_t>
class gRPC : public Client<response_t, request_t, stream_t>
{
public:
    /// @brief ctor, instantiates with new stub.
    gRPC(stub_t *new_stub) : stub(new_stub) {}

    /// @brief Interface for unary send.
    /// @param target
    /// @param request Should be of a generated proto message type.
    /// @returns Should be of a generated proto message type.
    response_t send(std::string target, request_t &request);

    /// @brief Interface for stream.
    stream_t stream(std::string target);

private:
    /// Stub to manage connection.
    std::unique_ptr<stub_t> stub;
};
