#!/bin/sh
echo Installing
echo - apiserver
go get -v github.com/pogointel/opm/apiserver
echo - bancheck
go get -v github.com/pogointel/opm/bancheck
echo - proxyhub
go get -v github.com/pogointel/opm/proxyhub
echo - scanner
go get -v github.com/pogointel/opm/scanner
echo - stats
go get -v github.com/pogointel/opm/stats
echo - opm
go get -v github.com/pogointel/opm
echo Done