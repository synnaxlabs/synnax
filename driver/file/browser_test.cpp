// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>
#include <fstream>
#include <string>

#include "gtest/gtest.h"

#include "x/cpp/fs/fs.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"
#include "x/cpp/x509/testutil/testutil.h"

#include "driver/file/file.h"

namespace driver::file {
namespace {
void write(const std::filesystem::path &path, const std::string &data) {
    std::ofstream out(path, std::ios::binary);
    out << data;
}

/// @brief the details of x::x509::testutil::CERTIFICATE, as the Core stores them.
const x::json::json CERTIFICATE_DETAILS = {
    {"subject", "CN=Test Client,O=Synnax"},
    {"issuer", "CN=Test Client,O=Synnax"},
    {"not_before", "1767225600000000000"},
    {"not_after", "4922899199000000000"},
    {"sha1", "915d90fff8fb63289202f5155ce6cbc027cc1c57"},
    {"sha256", "7b42752d0bb377f9c35ac660d491fa8fa46e5b8885fd41a2650f19f8bd52309e"},
};

/// @returns the entry named name in a reply, or null.
x::json::json entry(const x::json::json &data, const std::string &name) {
    for (const auto &e: data["entries"])
        if (e["name"] == name) return e;
    return nullptr;
}

class BrowserTest : public ::testing::Test {
protected:
    std::filesystem::path dir;
    std::filesystem::path secrets;

    void SetUp() override {
        this->dir = std::filesystem::temp_directory_path() /
                    ("file-browser-" + x::uuid::create().to_string());
        this->secrets = this->dir / "secrets";
        std::filesystem::create_directories(this->secrets);
    }

    void TearDown() override { std::filesystem::remove_all(this->dir); }

    [[nodiscard]] Browser browser() const {
        return Browser(
            Config{.roots = {"/"}, .home = this->dir / "home", .secrets_dir = secrets}
        );
    }
};
}

/// @brief it should list the roots, the home folder, and the secrets folder at the
/// top level.
TEST_F(BrowserTest, TopLevel) {
    const auto data = ASSERT_NIL_P(this->browser().list(""));
    const x::json::json expected = {
        {"path", ""},
        {"parent", ""},
        {"secrets_dir", this->secrets.string()},
        {"secret", false},
        {"truncated", false},
        {"entries",
         {
             {{"name", "/"}, {"path", "/"}, {"kind", "directory"}},
             {{"name", "home"},
              {"path", (this->dir / "home").string()},
              {"kind", "directory"}},
             {{"name", "secrets"},
              {"path", this->secrets.string()},
              {"kind", "directory"}},
         }},
    };
    ASSERT_EQ(data, expected);
}

/// @brief it should leave the secrets folder out of the top level when none is set.
TEST_F(BrowserTest, TopLevelWithoutSecrets) {
    const Browser browser(Config{.roots = {"/"}});
    const auto data = ASSERT_NIL_P(browser.list(""));
    ASSERT_EQ(data["secrets_dir"], "");
    ASSERT_EQ(data["entries"].size(), 1);
    ASSERT_EQ(data["entries"][0]["path"], "/");
}

/// @brief it should list folders first, then files, each in byte order of name.
TEST_F(BrowserTest, ListOrder) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder / "b");
    std::filesystem::create_directories(folder / "B");
    write(folder / "a.txt", "hello");
    write(folder / "Z.txt", "");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    std::vector<std::string> names;
    for (const auto &e: data["entries"])
        names.push_back(e["name"]);
    ASSERT_EQ(names, std::vector<std::string>({"B", "b", "Z.txt", "a.txt"}));
    ASSERT_EQ(data["path"], folder.string());
    ASSERT_EQ(data["parent"], this->dir.string());
    ASSERT_EQ(data["secrets_dir"], this->secrets.string());
    ASSERT_EQ(data["secret"], false);
    ASSERT_EQ(data["truncated"], false);
}

/// @brief it should give each entry its kind, size, and modified time.
TEST_F(BrowserTest, EntryFields) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder / "sub");
    write(folder / "a.txt", "hello");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    const auto file = entry(data, "a.txt");
    ASSERT_EQ(file["path"], (folder / "a.txt").string());
    ASSERT_EQ(file["kind"], "file");
    ASSERT_EQ(file["size"], 5);
    const auto modified = to_timestamp(
        std::filesystem::last_write_time(folder / "a.txt")
    );
    ASSERT_EQ(file["modified"], std::to_string(modified.nanoseconds()));
    ASSERT_FALSE(file.contains("certificate"));
    const auto sub = entry(data, "sub");
    ASSERT_EQ(sub["kind"], "directory");
    ASSERT_EQ(sub["size"], 0);
    ASSERT_TRUE(sub.contains("modified"));
}

/// @brief it should give the details of a certificate file, as the Core stores them.
TEST_F(BrowserTest, CertificateDetails) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    write(folder / "server.pem", x::x509::testutil::CERTIFICATE);
    write(folder / "upper.CRT", x::x509::testutil::CERTIFICATE);
    write(folder / "broken.cer", "not a certificate");
    write(folder / "server.txt", x::x509::testutil::CERTIFICATE);
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    ASSERT_EQ(entry(data, "server.pem")["certificate"], CERTIFICATE_DETAILS);
    ASSERT_EQ(entry(data, "upper.CRT")["certificate"], CERTIFICATE_DETAILS);
    ASSERT_FALSE(entry(data, "broken.cer").contains("certificate"));
    ASSERT_FALSE(entry(data, "server.txt").contains("certificate"));
}

/// @brief it should not read a certificate file larger than 1 MiB.
TEST_F(BrowserTest, LargeCertificateNotRead) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    write(
        folder / "big.pem",
        x::x509::testutil::CERTIFICATE + std::string(1 << 20, '\n')
    );
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    const auto big = entry(data, "big.pem");
    ASSERT_GT(big["size"].get<std::uintmax_t>(), MAX_CERTIFICATE_SIZE);
    ASSERT_FALSE(big.contains("certificate"));
}

/// @brief it should give only the name, path, and kind of each entry in the secrets
/// folder.
TEST_F(BrowserTest, SecretsFolderNamesOnly) {
    std::filesystem::create_directories(this->secrets / "nested");
    write(this->secrets / "password", "hunter2");
    write(this->secrets / "client.pem", x::x509::testutil::CERTIFICATE);
    const auto data = ASSERT_NIL_P(this->browser().list(this->secrets.string()));
    ASSERT_EQ(data["secret"], true);
    const x::json::json expected = {
        {{"name", "nested"},
         {"path", (this->secrets / "nested").string()},
         {"kind", "directory"}},
        {{"name", "client.pem"},
         {"path", (this->secrets / "client.pem").string()},
         {"kind", "file"}},
        {{"name", "password"},
         {"path", (this->secrets / "password").string()},
         {"kind", "file"}},
    };
    ASSERT_EQ(data["entries"], expected);
    const auto nested = ASSERT_NIL_P(
        this->browser().list((this->secrets / "nested").string())
    );
    ASSERT_EQ(nested["secret"], true);
}

/// @brief it should give only the name of the secrets folder and of a link into it.
TEST_F(BrowserTest, LinkIntoSecretsNamesOnly) {
    write(this->secrets / "client.pem", x::x509::testutil::CERTIFICATE);
    std::error_code ec;
    std::filesystem::create_symlink(
        this->secrets / "client.pem",
        this->dir / "link.pem",
        ec
    );
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto data = ASSERT_NIL_P(this->browser().list(this->dir.string()));
    ASSERT_EQ(data["secret"], false);
    const auto link = entry(data, "link.pem");
    ASSERT_EQ(link.size(), 3);
    ASSERT_EQ(link["kind"], "file");
    ASSERT_EQ(entry(data, "secrets").size(), 3);
}

/// @brief it should leave out a link whose target is missing.
TEST_F(BrowserTest, BrokenLinkLeftOut) {
    std::error_code ec;
    std::filesystem::create_symlink(this->dir / "missing", this->dir / "broken", ec);
    if (ec) GTEST_SKIP() << "cannot create a link: " << ec.message();
    const auto data = ASSERT_NIL_P(this->browser().list(this->dir.string()));
    ASSERT_TRUE(entry(data, "broken").is_null());
}

/// @brief it should keep the first 1000 entries of a larger folder.
TEST_F(BrowserTest, Truncated) {
    const auto folder = this->dir / "many";
    std::filesystem::create_directories(folder);
    for (size_t i = 0; i <= MAX_ENTRIES; i++)
        write(folder / ("f" + std::to_string(10000 + i)), "");
    const auto data = ASSERT_NIL_P(this->browser().list(folder.string()));
    ASSERT_EQ(data["truncated"], true);
    ASSERT_EQ(data["entries"].size(), MAX_ENTRIES);
    ASSERT_EQ(data["entries"][0]["name"], "f10000");
}

/// @brief it should give the folder without a trailing separator or dot segments.
TEST_F(BrowserTest, NormalPath) {
    const auto folder = this->dir / "certs";
    std::filesystem::create_directories(folder);
    const auto data = ASSERT_NIL_P(
        this->browser().list((this->dir / "secrets" / ".." / "certs").string() + "/")
    );
    ASSERT_EQ(data["path"], folder.string());
}

/// @brief it should give a root no parent.
TEST_F(BrowserTest, RootHasNoParent) {
    const auto data = ASSERT_NIL_P(this->browser().list("/"));
    ASSERT_EQ(data["path"], "/");
    ASSERT_EQ(data["parent"], "");
}

/// @brief it should reject a relative path.
TEST_F(BrowserTest, RelativePath) {
    const auto [data, err] = this->browser().list("certs");
    ASSERT_MATCHES(err, x::fs::INVALID_PATH);
    ASSERT_EQ(err.data, "certs is not an absolute path.");
}

/// @brief it should reject a path where nothing is.
TEST_F(BrowserTest, MissingPath) {
    const auto missing = this->dir / "missing";
    const auto [data, err] = this->browser().list(missing.string());
    ASSERT_MATCHES(err, x::fs::NOT_FOUND);
    ASSERT_EQ(err.data, missing.string() + " does not exist.");
}

/// @brief it should reject the path of a file.
TEST_F(BrowserTest, FilePath) {
    write(this->dir / "a.txt", "");
    const auto file = this->dir / "a.txt";
    const auto [data, err] = this->browser().list(file.string());
    ASSERT_MATCHES(err, x::fs::INVALID_PATH);
    ASSERT_EQ(err.data, file.string() + " is not a folder.");
}
}
