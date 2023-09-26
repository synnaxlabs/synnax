#pragma once

/// Local headers.

// std.
#include <string>

/// @brief Interface for client.
template <typename response_t, typename request_t, typename stream_t>
class Client
{
public:
    /// @brief Interface for unary send.
    virtual response_t send(std::string target, request_t &request) = 0;

    /// @brief Interface for stream.
    virtual stream_t stream(std::string target) = 0;
};