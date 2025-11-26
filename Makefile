.PHONY: build install clean test lint run

# ビルド出力ディレクトリ
BUILD_DIR := ./bin
BINARY := vive

# Goの設定
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOGET := $(GOCMD) get

# ビルドフラグ
LDFLAGS := -s -w

# デフォルトターゲット
all: build

# 依存関係の取得
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# ビルド
build: deps
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) ./cmd/vive

# インストール（$GOPATH/binに配置）
install: deps
	$(GOCMD) install ./cmd/vive

# クリーンアップ
clean:
	rm -rf $(BUILD_DIR)

# テスト
test:
	$(GOTEST) -v ./...

# リント
lint:
	golangci-lint run ./...

# 開発用: ビルドして実行
run: build
	$(BUILD_DIR)/$(BINARY) $(ARGS)

# クロスコンパイル
build-all: deps
	mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/vive
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/vive
	GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/vive
	GOOS=windows GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/vive

# ヘルプ
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  install    - Install to GOPATH/bin"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  lint       - Run linter"
	@echo "  run        - Build and run (use ARGS= for arguments)"
	@echo "  build-all  - Build for all platforms"
	@echo "  deps       - Download dependencies"
