build:
ifeq ($(OS),Windows_NT)
	go build -o build/swap-backend.exe main.go
else
	go build -o build/swap-backend main.go
endif

install:
ifeq ($(OS),Windows_NT)
	go install main.go
else
	go install main.go
endif

build-abi:
ifeq ($(OS),Windows_NT)
	abigen --abi=abi/ERC721SwapAgent.json --type=ERC721SwapAgent --pkg=abi --out=abi/ERC721SwapAgent.go
else
	abigen --abi=abi/ERC721SwapAgent.json --type=ERC721SwapAgent --pkg=abi --out=abi/ERC721SwapAgent.go
endif

.PHONY: build install
