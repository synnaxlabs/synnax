// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdio>
#include <filesystem>
#include <fstream>

#include "gtest/gtest.h"

#include "x/cpp/fs/fs.h"
#include "x/cpp/xtest/xtest.h"

class FSTest : public ::testing::Test {
protected:
    std::string test_dir;
    std::string test_file;
    std::string empty_file;
    std::string large_file;
    std::string binary_file;
    std::string non_existent_file;
    std::string special_chars_file;

    void SetUp() override {
        test_dir = (std::filesystem::temp_directory_path() / "fs_test").string();
        std::filesystem::create_directories(test_dir);
        test_file = test_dir + "/test.txt";
        empty_file = test_dir + "/empty.txt";
        large_file = test_dir + "/large.txt";
        binary_file = test_dir + "/binary.bin";
        non_existent_file = test_dir + "/non_existent.txt";
        special_chars_file = test_dir + "/special_chars.txt";
        create_test_file();
        create_empty_file();
        create_large_file();
        create_binary_file();
        create_special_chars_file();
    }

    void TearDown() override { std::filesystem::remove_all(test_dir); }

private:
    void create_test_file() const {
        std::ofstream file(test_file);
        file << "Hello, World!\n";
        file << "This is a test file.\n";
        file << "It has multiple lines.";
        file.close();
    }

    void create_empty_file() const {
        std::ofstream file(empty_file);
        file.close();
    }

    void create_large_file() const {
        std::ofstream file(large_file);
        // Create a file larger than the buffer size (1024 bytes)
        for (int i = 0; i < 200; i++) {
            file
                << "Line " << i
                << ": This is a line of text to make the file larger than the buffer size.\n";
        }
        file.close();
    }

    void create_binary_file() const {
        std::ofstream file(binary_file, std::ios::binary);
        unsigned char data[] = {0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD};
        file.write(reinterpret_cast<const char *>(data), sizeof(data));
        file.close();
    }

    void create_special_chars_file() const {
        std::ofstream file(special_chars_file);
        file << "Special chars: \t\n\r";
        file << "Unicode: €£¥";
        file << "\nEnd of file";
        file.close();
    }
};

/// @brief it should successfully read a file with multiple lines.
TEST_F(FSTest, ReadFileSuccess) {
    auto content = ASSERT_NIL_P(fs::read_file(test_file));
    ASSERT_EQ(content, "Hello, World!\nThis is a test file.\nIt has multiple lines.");
}

/// @brief it should successfully read an empty file.
TEST_F(FSTest, ReadEmptyFile) {
    auto content = ASSERT_NIL_P(fs::read_file(empty_file));
    ASSERT_TRUE(content.empty());
}

/// @brief it should successfully read a file larger than the buffer size.
TEST_F(FSTest, ReadLargeFile) {
    auto content = ASSERT_NIL_P(fs::read_file(large_file));
    ASSERT_FALSE(content.empty());
    ASSERT_NE(content.find("Line 0:"), std::string::npos);
    ASSERT_NE(content.find("Line 199:"), std::string::npos);
    ASSERT_GT(content.size(), 1024);
}

/// @brief it should return a NOT_FOUND error for a non-existent file.
TEST_F(FSTest, ReadNonExistentFile) {
    auto content = ASSERT_OCCURRED_AS_P(
        fs::read_file(non_existent_file),
        fs::NOT_FOUND
    );
    ASSERT_TRUE(content.empty());
}

/// @brief it should correctly read binary data from a file.
TEST_F(FSTest, ReadBinaryFile) {
    auto content = ASSERT_NIL_P(fs::read_file(binary_file));
    ASSERT_EQ(content.size(), 6);
    ASSERT_EQ(static_cast<unsigned char>(content[0]), 0x00);
    ASSERT_EQ(static_cast<unsigned char>(content[1]), 0x01);
    ASSERT_EQ(static_cast<unsigned char>(content[2]), 0x02);
    ASSERT_EQ(static_cast<unsigned char>(content[3]), 0xFF);
    ASSERT_EQ(static_cast<unsigned char>(content[4]), 0xFE);
    ASSERT_EQ(static_cast<unsigned char>(content[5]), 0xFD);
}

/// @brief it should correctly read a file containing special characters and unicode.
TEST_F(FSTest, ReadSpecialCharsFile) {
    auto content = ASSERT_NIL_P(fs::read_file(special_chars_file));
    ASSERT_NE(content.find("Special chars: \t\n\r"), std::string::npos);
    ASSERT_NE(content.find("Unicode: €£¥"), std::string::npos);
    ASSERT_NE(content.find("\nEnd of file"), std::string::npos);
}

/// @brief it should correctly read a file with spaces in its path.
TEST_F(FSTest, ReadFileWithSpacesInPath) {
    std::string path_with_spaces = test_dir + "/file with spaces.txt";
    std::ofstream file(path_with_spaces);
    file << "Content in file with spaces";
    file.close();
    auto content = ASSERT_NIL_P(fs::read_file(path_with_spaces));
    ASSERT_EQ(content, "Content in file with spaces");
    std::filesystem::remove(path_with_spaces);
}

/// @brief it should return consistent content when reading the same file multiple
/// times.
TEST_F(FSTest, ReadFileMultipleReads) {
    auto content1 = ASSERT_NIL_P(fs::read_file(test_file));
    auto content2 = ASSERT_NIL_P(fs::read_file(test_file));
    ASSERT_EQ(content1, content2);
}

/// @brief it should return a NOT_FOUND error when given an empty path.
TEST_F(FSTest, ReadFileEmptyPath) {
    auto content = ASSERT_OCCURRED_AS_P(fs::read_file(""), fs::NOT_FOUND);
    ASSERT_TRUE(content.empty());
}

/// @brief it should correctly read a file using a relative path.
TEST_F(FSTest, ReadFileRelativePath) {
    std::string relative_file = "test_relative.txt";
    std::ofstream file(relative_file);
    file << "Relative path content";
    file.close();
    auto content = ASSERT_NIL_P(fs::read_file(relative_file));
    ASSERT_EQ(content, "Relative path content");
    std::filesystem::remove(relative_file);
}

/// @brief it should define the correct error types.
TEST_F(FSTest, ErrorTypeVerification) {
    ASSERT_EQ(fs::FS_ERROR.type, "fs");
    ASSERT_EQ(fs::NOT_FOUND.type, "fs.not_found");
    ASSERT_EQ(fs::INVALID_PATH.type, "fs.invalid_path");
    ASSERT_EQ(fs::PERMISSION_DENIED.type, "fs.permission_denied");
    ASSERT_EQ(fs::READ_ERROR.type, "fs.read_error");
}

/// @brief it should correctly read a file with different newline styles.
TEST_F(FSTest, ReadFileWithNewlines) {
    std::string newline_file = test_dir + "/newlines.txt";
    std::ofstream file(newline_file);
    file << "Line1\n";
    file << "Line2\r\n";
    file << "Line3\r";
    file << "Line4";
    file.close();
    auto content = ASSERT_NIL_P(fs::read_file(newline_file));
    ASSERT_EQ(content, "Line1\nLine2\r\nLine3\rLine4");
}

/// @brief it should correctly read a file exactly matching the buffer size.
TEST_F(FSTest, ReadFileExactBufferSize) {
    std::string exact_buffer_file = test_dir + "/exact_buffer.txt";
    std::ofstream file(exact_buffer_file);
    std::string data(1024, 'A');
    file << data;
    file.close();
    auto content = ASSERT_NIL_P(fs::read_file(exact_buffer_file));
    ASSERT_EQ(content.size(), 1024);
    ASSERT_EQ(content, data);
}

/// @brief it should correctly read a file one byte larger than the buffer size.
TEST_F(FSTest, ReadFileOneByteOverBuffer) {
    std::string over_buffer_file = test_dir + "/over_buffer.txt";
    std::ofstream file(over_buffer_file);
    std::string data(1025, 'B');
    file << data;
    file.close();
    auto content = ASSERT_NIL_P(fs::read_file(over_buffer_file));
    ASSERT_EQ(content.size(), 1025);
    ASSERT_EQ(content, data);
}
