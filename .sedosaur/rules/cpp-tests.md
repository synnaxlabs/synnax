+++
description = "C++ test names and briefs"
paths = ["**/*_test.cpp", "**/*_tests.cpp"]
+++

A new C++ test takes the name form that its file already uses, PascalCase or `testX`. Do
not report a PascalCase name, or a name without "Should". A new or renamed test has a
`/// @brief it should ...` comment on the line above its `TEST` or `TEST_F`.
