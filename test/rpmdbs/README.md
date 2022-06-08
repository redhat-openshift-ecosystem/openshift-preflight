# RPMDB fixtures

This directory contains rpmdb fixtures for testing the **certification/internal/rpm** package.

The RPMDB currently supports a sqlite and berkeleydb backend, and we need both for this test
as of this writing.

The sqlite3 backend was created using `rpmdb --initdb --dbpath $PWD` on Fedora 36.
The berkeleydb backend was created using `rpmdb --initdb --dbpath $PWD` on UBI:7

Both have a single TZData package installed to save space in the repository.

It was installed using `rpm -i --dbpath $PWD <packagename>`.

