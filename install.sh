#!/bin/bash

printf "Installing...\n"
go install
pandoc -s -t man man/netwc.md -o ./man/netwc.1
mv ./man/netwc.1 /usr/local/share/man/man1
gzip /usr/local/share/man/man1/netwc.1
printf "Done!\n"