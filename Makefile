.PHONY: build check test demo serve fixtures native-setup serve-all demo-all smoke
build:
	npm ci
	npm run build
check:
	npm run check
	go vet ./...
test:
	go test -race ./...
	python3 -m unittest discover -s src/internal/nfc/providers -p "test_*.py"
	python3 -m unittest discover -s src/internal/bluetooth/providers -p "test_*.py"
fixtures:
	python3 fixtures/generate.py
demo: build
	@test -f .data/demo.db || ./bin/wifi-scan demo --db .data/demo.db > /dev/null
	./bin/wifi-scan serve --db .data/demo.db
serve: build
	./bin/wifi-scan serve

# Python dependencies are needed only for native Bluetooth/NFC operations.
native-setup:
	python3 -m venv .venv
	.venv/bin/python -m pip install -r requirements-native.txt
serve-all: build
	./bin/signal-atlas
demo-all: build
	./bin/signal-atlas --demo

smoke:
	python3 scripts/smoke.py
