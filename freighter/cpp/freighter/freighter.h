#pragma once

/// Local headers.

// std.
#include <string>
#include <utility>

/// @brief Interface for client.
template <typename response_t, typename request_t, typename stream_t, typename err_t>
class Client
{
public:
    /// @brief Interface for unary send.
    virtual std::pair<response_t, err_t> send(std::string target, request_t &request) = 0;

    /// @brief Interface for stream.
    virtual stream_t stream(std::string target) = 0;
};

/// @brief Interface for Streamer.
template <typename response_t, typename request_t, typename err_t>
class Streamer
{
public:
    /// @brief Interface for streamer send.
    virtual err_t send(request_t &request) = 0;

    /// @brief Interface for streamer receive.
    virtual std::pair<response_t, err_t> receive() = 0;

    /// @brief Closes the sending direction of the stream.
    virtual err_t closeSend() = 0;
};