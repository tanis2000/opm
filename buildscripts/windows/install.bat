@echo Installing
@echo - apiserver
@go install -v github.com/pogointel/opm/apiserver
@echo - bancheck
@go install -v github.com/pogointel/opm/bancheck
@echo - proxyhub
@go install -v github.com/pogointel/opm/proxyhub
@echo - scanner
@go install -v github.com/pogointel/opm/scanner
@echo - stats
@go install -v github.com/pogointel/opm/stats
@echo - opm
@go install -v github.com/pogointel/opm
@echo Done